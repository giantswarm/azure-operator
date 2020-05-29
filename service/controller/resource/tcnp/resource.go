package tcnp

import (
	"time"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/v4/service/controller/debugger"
)

const (
	// Name is the identifier of the resource.
	Name = "tcnp"
)

type Config struct {
	Debugger       *debugger.Debugger
	G8sClient      versioned.Interface
	K8sClient      kubernetes.Interface
	Location       string
	Logger         micrologger.Logger
	VMSSMSIEnabled bool
}

type Resource struct {
	debugger       *debugger.Debugger
	g8sClient      versioned.Interface
	k8sClient      kubernetes.Interface
	location       string
	logger         micrologger.Logger
	vmssMSIEnabled bool
}

func New(config Config) (*Resource, error) {
	if config.Debugger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Debugger must not be empty", config)
	}
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Location == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Location must not be empty", config)
	}

	r := &Resource{
		debugger:       config.Debugger,
		g8sClient:      config.G8sClient,
		location:       config.Location,
		logger:         config.Logger,
		vmssMSIEnabled: config.VMSSMSIEnabled,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getResourceStatus(customObject providerv1alpha1.AzureConfig, t string) (string, error) {
	{
		c, err := r.g8sClient.ProviderV1alpha1().AzureConfigs(customObject.Namespace).Get(customObject.Name, metav1.GetOptions{})
		if err != nil {
			return "", microerror.Mask(err)
		}

		customObject = *c
	}

	for _, r := range customObject.Status.Cluster.Resources {
		if r.Name != Name {
			continue
		}

		for _, c := range r.Conditions {
			if c.Type == t {
				return c.Status, nil
			}
		}
	}

	return "", nil
}

func (r *Resource) setResourceStatus(customObject providerv1alpha1.AzureConfig, t string, s string) error {
	// Get the newest CR version. Otherwise status update may fail because of:
	//
	//	 the object has been modified; please apply your changes to the
	//	 latest version and try again
	//
	{
		c, err := r.g8sClient.ProviderV1alpha1().AzureConfigs(customObject.Namespace).Get(customObject.Name, metav1.GetOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		customObject = *c
	}

	resourceStatus := providerv1alpha1.StatusClusterResource{
		Conditions: []providerv1alpha1.StatusClusterResourceCondition{
			{
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Status:             s,
				Type:               t,
			},
		},
		Name: Name,
	}

	var set bool
	for i, r := range customObject.Status.Cluster.Resources {
		if r.Name != Name {
			continue
		}

		for _, c := range r.Conditions {
			if c.Type == t {
				continue
			}
			resourceStatus.Conditions = append(resourceStatus.Conditions, c)
		}

		customObject.Status.Cluster.Resources[i] = resourceStatus
		set = true
	}

	if !set {
		customObject.Status.Cluster.Resources = append(customObject.Status.Cluster.Resources, resourceStatus)
	}

	{
		n := customObject.GetNamespace()
		_, err := r.g8sClient.ProviderV1alpha1().AzureConfigs(n).UpdateStatus(&customObject)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
