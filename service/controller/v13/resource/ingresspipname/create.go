package ingresspipname

import (
	"context"
	"fmt"

	"github.com/giantswarm/azure-operator/service/controller/v13/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	{
		if key.IngressPIPName(cr) != "" {
			r.logger.LogCtx(ctx, "level", "debug", "message", "found out that ingress PIP name is already set to status")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

			return nil
		}
	}

	var ingressPIPName string
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "finding if ingress PIP has been already created")

		groupName := key.ClusterID(cr)
		pipClient, err := r.getPIPClient(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		// First try to look for legacy PIP name (dummmy-pip).
		_, err = pipClient.Get(ctx, groupName, key.LegacyIngressLBPIPName, "")
		if IsPIPNotFound(err) {
			// This is ok. Maybe PIP is created with currently desired name?

			_, err = pipClient.Get(ctx, groupName, key.DefaultIngressPIPName(cr), "")
			if IsPIPNotFound(err) {
				// This is ok as well. The given Public IP is not created yet.
			} else if err != nil {
				return microerror.Mask(err)
			}

			ingressPIPName = key.DefaultIngressPIPName(cr)

		} else if err != nil {
			return microerror.Mask(err)
		} else {
			ingressPIPName = key.LegacyIngressLBPIPName
		}

		if ingressPIPName == "" {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find already created ingress PIP")
			ingressPIPName = key.DefaultIngressPIPName(cr)
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("defaulting to %q", ingressPIPName))
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found ingress PIP with name %q", ingressPIPName))
		}
	}

	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "updating CR status with ingress PIP name")

		cr.Status.Provider.Ingress.LoadBalancer.PublicIPName = ingressPIPName

		_, err := r.g8sClient.ProviderV1alpha1().AzureConfigs(cr.Namespace).UpdateStatus(&cr)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "updated CR status with ingress PIP name")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
	}

	return nil
}
