package machinepoolmigration

import (
	"context"

	oldcapzexpv1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/capzexp/v1alpha3"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CheckerConfig struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

type Checker struct {
	ctrlClient client.Client
	logger     micrologger.Logger
}

func NewChecker(c CheckerConfig) (*Checker, error) {
	if c.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", c)
	}
	if c.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", c)
	}

	r := &Checker{
		ctrlClient: c.CtrlClient,
		logger:     c.Logger,
	}

	return r, nil
}

func (c *Checker) NewAzureMachinePoolPendingCreation(ctx context.Context, namespacedName types.NamespacedName) (bool, error) {
	migratedAzureMachinePoolExists, err := c.CheckIfMigratedAzureMachinePoolExists(ctx, namespacedName)
	if err != nil {
		c.logger.Debugf(ctx, "Failed to fetch migrated AzureMachinePool %s/%s, assuming migration has not been completed, should cancel reconciliation", namespacedName.Namespace, namespacedName.Name)
		return false, microerror.Mask(err)
	}
	oldAzureMachinePoolExists, err := c.CheckIfOldAzureMachinePoolExists(ctx, namespacedName)
	if err != nil {
		c.logger.Debugf(ctx, "Failed to fetch old AzureMachinePool %s/%s, assuming migration has not been completed, should cancel reconciliation", namespacedName.Namespace, namespacedName.Name)
		return false, microerror.Mask(err)
	}

	if oldAzureMachinePoolExists && !migratedAzureMachinePoolExists {
		// Old experimental AzureMachinePool exists, but new one is not found yet.
		c.logger.Debugf(ctx, "Found old AzureMachinePool %s/%s, but migrated AzureMachinePool not found, assuming migration has not been completed, should cancel reconciliation", namespacedName.Namespace, namespacedName.Name)
		return true, nil
	}

	return false, nil
}

func (c *Checker) CheckIfOldAzureMachinePoolExists(ctx context.Context, namespacedName types.NamespacedName) (bool, error) {
	c.logger.Debugf(ctx, "Checking if old exp AzureMachinePool %s/%s still exists", namespacedName.Namespace, namespacedName.Name)
	oldAzureMachinePool := &oldcapzexpv1alpha3.AzureMachinePool{}
	err := c.ctrlClient.Get(ctx, namespacedName, oldAzureMachinePool)
	if apierrors.IsNotFound(err) {
		c.logger.Debugf(ctx, "Old exp AzureMachinePool %s/%s not found", namespacedName.Namespace, namespacedName.Name)
		return false, nil
	} else if err != nil {
		c.logger.Debugf(ctx, "Failed to fetch old exp AzureMachinePool %s/%s", namespacedName.Namespace, namespacedName.Name)
		return false, microerror.Mask(err)
	}

	c.logger.Debugf(ctx, "Old exp AzureMachinePool %s/%s found", namespacedName.Namespace, namespacedName.Name)
	return true, nil
}

func (c *Checker) CheckIfMigratedAzureMachinePoolExists(ctx context.Context, namespacedName types.NamespacedName) (bool, error) {
	c.logger.Debugf(ctx, "Checking if migrated AzureMachinePool %s/%s has been created", namespacedName.Namespace, namespacedName.Name)
	azureMachinePool := &capzexp.AzureMachinePool{}
	err := c.ctrlClient.Get(ctx, namespacedName, azureMachinePool)
	if apierrors.IsNotFound(err) {
		c.logger.Debugf(ctx, "Migrated AzureMachinePool %s/%s not found", namespacedName.Namespace, namespacedName.Name)
		return false, nil
	} else if err != nil {
		c.logger.Debugf(ctx, "Failed to fetch migrated AzureMachinePool %s/%s", namespacedName.Namespace, namespacedName.Name)
		return false, microerror.Mask(err)
	}

	if !AreAzureMachinePoolReferencesUpdated(*azureMachinePool) {
		c.logger.Debugf(ctx, "Migrated AzureMachinePool %s/%s found, but its references are not updated", namespacedName.Namespace, namespacedName.Name)
		return false, nil
	}

	c.logger.Debugf(ctx, "Migrated AzureMachinePool %s/%s found", namespacedName.Namespace, namespacedName.Name)
	return true, nil
}
