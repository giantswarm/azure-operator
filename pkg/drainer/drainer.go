package drainer

import (
	"context"
	"time"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/to"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Drainer struct {
	logger    micrologger.Logger
	wcClients k8sclient.Interface
}

type Config struct {
	Logger    micrologger.Logger
	WCClients k8sclient.Interface
}

func New(config Config) (*Drainer, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.WCClients == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.WCClients must not be empty", config)
	}

	return &Drainer{
		logger:    config.Logger,
		wcClients: config.WCClients,
	}, nil
}

func (d *Drainer) DrainNode(ctx context.Context, nodename string, timeout time.Duration) error {
	d.logger.Debugf(ctx, "Getting node %q for draining", nodename)
	node := corev1.Node{}
	err := d.wcClients.CtrlClient().Get(ctx, client.ObjectKey{Name: nodename}, &node)
	if apierrors.IsNotFound(err) {
		d.logger.Debugf(ctx, "Node %q was not found, it was probably already drained and deleted", nodename)
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	d.logger.Debugf(ctx, "Cordoning node %q", nodename)
	err = d.cordon(ctx, node)
	if err != nil {
		return microerror.Mask(err)
	}

	d.logger.Debugf(ctx, "Evicting pods on node %q", nodename)
	return d.evictPods(ctx, node, timeout)
}

func (d *Drainer) cordon(ctx context.Context, node corev1.Node) error {
	p := client.MergeFrom(node.DeepCopy())
	node.Spec.Unschedulable = true
	err := d.wcClients.CtrlClient().Patch(ctx, &node, p)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (d *Drainer) evictPods(ctx context.Context, node corev1.Node, timeout time.Duration) error {
	o := func() error {
		var customPods []corev1.Pod
		var kubesystemPods []corev1.Pod
		{
			podList := corev1.PodList{}
			err := d.wcClients.CtrlClient().List(ctx, &podList, client.MatchingFields{"spec.nodeName": node.GetName()})
			if err != nil {
				return microerror.Mask(err)
			}

			for _, pod := range podList.Items {
				if isCriticalPod(pod.Name) {
					// ignore critical pods (api, controller-manager and scheduler)
					// they are static pods so kubelet will recreate them anyway and it can cause other issues
					continue
				}
				if isDaemonSetPod(pod) {
					// ignore daemonSet owned pods
					// daemonSets pod are recreated even on unschedulable node so draining doesn't make sense
					// we are aligning here with community as 'kubectl drain' also ignore them
					continue
				}
				if isEvictedPod(pod) {
					// we don't need to care about already evicted pods
					continue
				}

				if pod.GetNamespace() == "kube-system" {
					kubesystemPods = append(kubesystemPods, pod)
				} else {
					customPods = append(customPods, pod)
				}
			}
		}

		left := len(customPods) + len(kubesystemPods)
		if left == 0 {
			return nil
		}

		if len(customPods) > 0 {
			for _, pod := range customPods {
				err := d.evict(ctx, pod)
				if IsCannotEvictPod(err) {
					continue
				} else if err != nil {
					return microerror.Mask(err)
				}
			}
		}

		if len(kubesystemPods) > 0 && len(customPods) == 0 {
			for _, pod := range kubesystemPods {
				err := d.evict(ctx, pod)
				if IsCannotEvictPod(err) {
					continue
				} else if err != nil {
					return microerror.Mask(err)
				}
			}
		}

		return microerror.Maskf(evictionInProgressError, "%d pods still pending eviction, waiting", left)
	}

	err := backoff.RetryNotify(o, backoff.NewConstant(timeout, 10*time.Second), backoff.NewNotifier(d.logger, ctx))
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (d *Drainer) evict(ctx context.Context, pod corev1.Pod) error {
	eviction := &policyv1beta1.Eviction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.GetName(),
			Namespace: pod.GetNamespace(),
		},
		DeleteOptions: &metav1.DeleteOptions{
			GracePeriodSeconds: terminationGracePeriod(pod),
		},
	}

	err := d.wcClients.K8sClient().PolicyV1beta1().Evictions(eviction.GetNamespace()).Evict(ctx, eviction)
	if IsCannotEvictPod(err) {
		return microerror.Mask(cannotEvictPodError)
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func terminationGracePeriod(pod corev1.Pod) *int64 {
	var d int64 = 60

	if pod.Spec.TerminationGracePeriodSeconds != nil && *pod.Spec.TerminationGracePeriodSeconds > 0 {
		d = *pod.Spec.TerminationGracePeriodSeconds
	}

	return to.Int64P(d)
}
