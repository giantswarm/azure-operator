package instance

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) deploymentUninitializedTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	azureMachinePool := &v1alpha3.AzureMachinePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       cr.Name,
			Namespace:                  cr.Namespace,
			Labels:                     cr.Labels,
			Annotations:                cr.Annotations,
			OwnerReferences:            nil,
		},
		Spec:       v1alpha3.AzureMachinePoolSpec{
			Location:       "",
			Template:       v1alpha3.AzureMachineTemplate{},
			AdditionalTags: nil,
			ProviderID:     "",
			ProviderIDList: nil,
		},
	}
	err = r.CtrlClient.Create(ctx, azureMachinePool)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
	reconciliationcanceledcontext.SetCanceled(ctx)

	return "", nil
}
