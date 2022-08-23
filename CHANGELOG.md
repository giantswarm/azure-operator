# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- Bump `k8scc` to fix syntax error in `k8s-addons` script.

### Changed

- Bump `k8scc` to v14 to support kubernetes 1.23.
- Change default storage classes in order to use out-of-tree CSI provisioner. 
- Improved storage class migration resource.
 
### Removed

- Remove --cloud-config flag from k8s components.

## [5.22.0] - 2022-07-04

### Changed

- Tighten pod and container security contexts for PSS restricted policies.

### Fixed

- Fix handling of `MachinePools'` status fields for empty node pools.

### Changed

- Bump `k8scc` to enable `auditd` monitoring for `execve` syscalls.

## [5.21.0] - 2022-06-22

### Changed

- Changes to EncryptionConfig in order to work with `encryption-provider-operator`.

### Fixed

- Add pause annotation before deleting old machinepool and azuremachinepool CRs during migration to non-exp.
- Update ownerReference UIDs during migration to non-exp.
- Avoid updating `AzureCluster` at every reconciliation loop in the `subnet` resource.
- Avoid saving `AzureCluster` status if there are no changes to avoid useless reconciliation loops.

## [5.20.0] - 2022-06-07

### Changed

- Bumped k8scc to latest version to fix `localhost` node name problem.

## [5.19.0] - 2022-06-07

### Added

- Added possibility to specify VNet CIDR in `AzureCluster`.
- Migrate MachinePool CRs from `exp.cluster.x-k8s.io/v1alpha3` to `cluster.x-k8s.io/v1beta1`
- Migrate AzureMachinePool CRs from `exp.infrastructure.cluster.x-k8s.io/v1alpha3` to `infrastructure.cluster.x-k8s.io/v1beta1`

### Changed

- Use systemd cgroup driver on masters and cgroups v2 worker nodes. 
- Update github.com/Azure/azure-sdk-for-go to v58.1.0+incompatible
- Update github.com/giantswarm/apiextensions to v6.0.0
- Update github.com/giantswarm/certs to v4.0.0
- Update github.com/giantswarm/conditions to v0.5.0
- Update github.com/giantswarm/conditions-handler to v0.3.0
- Update github.com/giantswarm/k8sclient to v7.0.1
- Update github.com/giantswarm/k8scloudconfig to v13.4.0
- Update github.com/giantswarm/operatorkit to v7.0.1
- Update github.com/giantswarm/release-operator to v3.2.0
- Update github.com/giantswarm/tenantcluster to v6.0.0
- Update k8s.io/api to v0.22.2
- Update k8s.io/apiextensions-apiserver to v0.22.2
- Update k8s.io/apimachinery to v0.22.2
- Update k8s.io/client-go to v0.22.2
- Update sigs.k8s.io/cluster-api to v1.0.5
- Update sigs.k8s.io/cluster-api-provider-azure to v1.0.2
- Update sigs.k8s.io/controller-runtime to v0.10.3
- Bump various other dependencies to address CVEs.

### Fixed

- Set `AzureMachine.Status.Ready` according to AzureMachine's Ready condition.

## [5.18.0] - 2022-03-21

### Added

- Add VerticalPodAutoscaler CR.

## [5.17.0] - 2022-03-15

### Fixed

- Fix panic while checking for cgroups version during upgrade.

### Added

- Add GiantSwarmCluster tag to Vnet.

## [5.16.0] - 2022-02-23

### Changed

- Make nodepool nodes roll in case the user switches between cgroups v1 and v2.

## [5.15.0] - 2022-02-16

### Changed

- Drop dependency on `giantswarm/apiextensions/v2`.
- Bump `k8scloudconfig` to disable `rpc-statd`.

## [5.14.0] - 2022-02-02

### Added

- Add support for feature that enables forcing cgroups v1 for Flatcar version `3033.2.0` and above.

### Changed

- Upgraded to giantswarm/exporterkit v1.0.0
- Upgraded to giantswarm/microendpoint v1.0.0
- Upgraded to giantswarm/microkit v1.0.0
- Upgraded to giantswarm/micrologger v0.6.0
- Upgraded to giantswarm/versionbundle v1.0.0
- Upgraded to spf13/viper v1.10.0

## [5.13.0] - 2022-01-14

### Changed

- Bumped k8scc to latest version to support Calico 3.21.

## [5.12.0] - 2021-12-14

### Added

- Deal with AzureClusterConfig CR to avoid cluster operator conflict.

## [5.11.0] - 2021-12-10

### Removed

- Remove creation of legacy `AzureClusterConfig` CR as they are unused.

## [5.10.2] - 2021-12-07

### Fixed

- Consider case when API is down when checking if Master node is upgrading during node pool reconciliation.

## [5.10.1] - 2021-12-02

### Changed

- When looking for the encryption secret, search on all namespaces (to support latest cluster-operator).

## [5.10.0] - 2021-11-08

### Changed

- Delegate Storage account type selection for master VM's disks to Azure API.
- Separate the drain and node deletion phases during node pool upgrades to avoid stuck disks.

### Fixed

- During an upgrade, fixed the detection of a master node being upgraded to wait before upgrading node pools.

## [5.9.0] - 2021-09-13

### Changed

- Use go embed in place of pkger.
- Rename API backend pool to comply with CAPZ.
- Rename API Load Balancing rule to comply with CAPZ.
- Rename API health probe to comply with CAPZ.
- Set `DisableOutputSnat` to true for API Load Balancer Load Balancing Rule to comply with CAPZ.
- Bumped `k8scloudconfig` to support Kubernetes 1.21

### Fixed

- Ensure Spark CR release version label is updated when upgrading a cluster.

### Removed

- Remove MSI extension from node pools.
- Remove VPN gateway cleanup code.

## [5.8.1] - 2021-07-22

### Fixed

- Fix namespace in secret reference of `AzureClusterIdentity`.

## [5.8.0] - 2021-07-13

### Added

- Allow using an existing public IP for the NAT gateway of worker nodes.

### Fixed

- Fix udev rules that caused `/boot` automount to fail

### Changed

- Upgrade `k8scloudconfig` to `v10.8.1` from `v10.5.0`.

## [5.7.2] - 2021-06-24

### Fixed

- Ensure the node pool deployment is applied when the node pool size is changed externally.

## [5.7.1] - 2021-06-21

### Changed

- Consider node pools out of date if flatcar image has changed.
- Consider node pools out of date if kubernetes version has changed.
- `AzureClusterIdentity`, and the secret it references are created in the `AzureCluster` namespace instead of `giantswarm`.
- Don't update `AzureClusterIdentity` CR's that are not managed by azure-operator.

### Fixed

- Don't get the node pool upgrade stuck if the current state of `AzureMachinePool` is invalid.

## [5.7.0] - 2021-05-13

### Changed

- Avoid creating too many worker nodes at the same time when upgrading node pools.
- Don't reimage master instances unless the masters VMSS has the right model.
- Don't wait for new workers to be up during spot instances node pools upgrades.
- Bumped `k8scloudconfig` to `10.5.0` to support kubernetes 1.20.

### Fixed

- Rely on k8s nodes instead of Azure instances when counting up-to-date nodes to decide if upgrade has finished.
- Fixed logic that decides whether or not to update an `AzureMachine` based on the `release.giantswarm.io/last-deployed-version` annotation.
- When deleting a node pool, also delete the VMSS role assignment.
- Check errors coming from k8s API using the wrapped error.

## [5.6.0] - 2021-04-21

### Changed

- Replace VPN Gateway with VNet Peering.
- Update OperatorKit to `v4.3.1` to drop usage of self-link which is not supported in k8s 1.20 anymore.

### Removed

- Support for single tenant BYOC credentials (warning: the operator will error at startup if any organization credentials is not multi tenant).

## [5.5.2] - 2021-03-18

### Changed

- Increase VMSS termination events timeout to 15 minutes.

### Fixed

- Avoid logging errors when trying to create the workload cluster k8s client and cluster is not ready yet.

## [5.5.1] - 2021-02-24

### Fixed

- Fix a race condition when upgrading node pools with 0 replicas.
- Fix Upgrading condition for node pools with autoscaler enabled.

## [5.5.0] - 2021-02-22

### Added

- Add new handler that creates `AzureClusterIdentity` CRs and the related `Secrets` out of Giant Swarm's credential secrets.
- Ensure `AzureCluster` CR has the `SubscriptionID` field set.
- Reference `Spark` CR as bootstrap reference from the `MachinePool` CR.
- Ensure node pools min size is applied immediately when changed.

### Fixed

- Avoid blocking the whole `AzureConfig` handler on cluster creation because we can't update the `StorageClasses`.
- Avoid overriding the NP size when the scaling is changed by autoscaler.

## [5.4.0] - 2021-02-05

### Changed

- Changed `StorageClasses` `volumeBindingMode` to `WaitForFirstConsumer`.
- When setting Cluster `release.giantswarm.io/last-deployed-version` annotation, Cluster `Ready` condition is not checked anymore, which effectively means that Cluster `Upgrading` condition does not depend on Cluster `Ready` condition.
- Use cluster-api-provider-azure v0.4.12-gsalpha1.
- Simplified the upgrade process by leveraging automated draining of nodes.

### Added

- Added spot instances support for node pools.
- Setting `release.giantswarm.io/last-deployed-version` on `AzureMachine` CR when the control plane creation or upgrade is done.
- Setting AzureMachine `Creating` and `Upgrading` conditions. Existing condition handlers `Creating` and `Upgrading` are used.

## [5.3.0] - 2021-02-01

### Changed

- Enable VMSS termination events.
- Bump `conditions-handler` to v0.2.1 to get `MachinePool` `ReplicasReady` fixes.

### Fixed

- When scaling up node pool VMSS during an upgrade, consider the real number of old workers running and not the value in the `MachinePool` CR to handle the case when the Autoscaler changed the size.
- Handle WC API not available error in `nodestatus` handler.
- Fix logging statements when using debug log level.

### Removed

- Remove check for `germanywestcentral` region and assume availability zone settings are correct in the CRs.

## [5.2.1] - 2021-01-20

### Fixed

- Ensure the management cluster's network space is never used for workload clusters.

## [5.2.0] - 2021-01-14

### Changed

- Bump `conditions-handler` to v0.2.0 to get `MachinePool` `ReplicasReady` condition.

### Fixed

- Ensure that availability zones are kept unchanged during migration from 12.x to 13.x.
- Don't set `MachinePool.Status.InfrastructureReady` in `nodestatus` handler.
- Ensure autoscaler annotations during migration from 12.x to 13.x.
- Improve handling errors when accessing Kubernetes API.

## [5.1.0] - 2020-12-14

### Changed

- Only submit Subnet ARM deployment when Subnet name or Subnet CIDR change.
- Use controller-runtime instead of typed clients.
- Move provider-independent conditions implementation to external `giantswarm/conditions` and `giantswarm/conditions-handlers` modules.
- Replaced Cluster `ProviderInfrastructureReady` with upstream `InfrastructureReady` condition.
- Fix incorrect (too early) `Upgrading` condition transition from `True` to `False`.

### Added

- Tenant cluster k8s client lookup is cached.
- Add `terminate-unhealthy-node` feature to automaticaly terminate bad and unhealthy nodes in a Cluster.
- Cluster `ControlPlaneReady` condition.
- AzureMachine `Ready`, `SubnetReady` and `VMSSReady` conditions.
- MachinePool `Creating` condition.

## [5.0.0] - 2020-12-01

### Fixed

- Use CP public IP's instead of TC public IP's to re-configure masters network security group.

## [5.0.0-beta7] - 2020-11-26

### Fixed

- Re-configure masters network security group to allow CP's public IPs to etcd LB ingress.

## [5.0.0-beta6] - 2020-11-26

### Fixed

- Avoid returning errors when still waiting for tenant cluster k8s API to be ready.
- Re-configure workers' network security group rules when upgrading from pre-NP cluster.
- Release allocated subnet when deleting node pool.
- Allow the control plane nodes to access the ETCD cluster for monitoring and backup purposes.

## [5.0.0-beta5] - 2020-11-18

### Fixed

- Don't set `Upgrading` condition `Reason` when it's `False` and already contains a `Reason`.

## [5.0.0-beta4] - 2020-11-17

## [5.0.0-beta2] - 2020-11-16

### Changed

- Roll nodes on release upgrade if major components involved in node creation changes (k8s, flatcar, etcd...).

## [5.0.0-beta1] - 2020-11-11

### Added

- Pass dockerhub token for kubelet authorized image pulling.
- Add missing registry mirrors in `spark` resource.
- Set `Cluster` and `AzureCluster` Ready status fields.

### Fixed

- Only try to save Azure VMSS IDs in Custom Resources if VMSS exists.
- Fix firewall rules to allow traffic between nodes in all node pools.

### Changed

- Use `AzureCluster.Spec.NetworkSpec.Vnet.CIDRBlocks` field instead of deprecated `AzureCluster.Spec.NetworkSpec.Vnet.CidrBlock`.
- Use `Subnet.CIDRBlocks` field instead of deprecated `Subnet.CidrBlock`.
- Simplified master node's upgrade process.
- Upgraded `apiextensions` to `3.7.0`.
- Upgraded `e2e-harness` to `3.0.0`.
- Upgraded `helmclient` to `3.0.1`.
- Upgraded `k8sclient` to `5.0.0`.
- Upgraded `k8scloudconfig` to `9.1.1`.
- Upgraded `operatorkit` to `4.0.0`.
- Upgraded `statusresource` to `3.0.0`.

## [5.0.0-alpha4] - 2020-10-27

### Changed

- Set cluster-autoscaler-enabled tag to false when min replicas and max replicas are the same for a node pool.

### Removed

- Removed instance watchdog to save on VMSS API calls.
- Removed 50% VMSS calls remaining check that stopped node operations to prevent 429 error from happening.

## [5.0.0-alpha3] - 2020-10-16

### Changed

- Do not use public SSH keys coming from the CRs.

### Fixed

- Try to send only one request to VMSS Azure API from `nodepool` handler.

## [5.0.0-alpha2] - 2020-10-14

### Fixed

- Fixed firewall rules to allow prometheus to scrape node-level exporters from all node pools.
- Encryption secret is now taken from the CR namespace rather than the organization namespace.

### Changed

- Get the storage account type to use for node pools' VMSS from the AzureMachinePool CR.

## [5.0.0-alpha1] - 2020-10-12

### Added

- Add monitoring label
- Add provider independent controllers to manage labeling and setting owner references in other provider dependent objects.
- Export container logs for e2e tests to azure analytics.
- Enable persistent volume `expansion` support in the default `Storage Classes`.
- Added to all VMSSes the tags needed by cluster autoscaler.

### Changed

- Decouple `Service` from controllers using an slice of controllers.
- Retry failed ARM deployments regardless of the checksum check.
- Master instances now use a dedicated NAT gateway for egress traffic.
- Updated backward incompatible Kubernetes dependencies to v1.18.5.
- Removed the ETCD Private Load Balancer, reusing the API public one for ETCD traffic (needed by HA masters).
- Updated CAPI to `v0.3.9` and CAPZ to `v0.4.7`, using GiantSwarm forks that contain k8s 1.18 changes.
- Use `DataDisks` field to define VM disks when creating node pools.
- Don't error if certificates are not present yet. Cancel reconciliation and wait until next loop instead.

## [4.2.0] - 2020-07-28

### Added

- Mapping from Cluster API & CAPZ CRs to AzureConfig. This change provides migration path towards Azure Cluster API implementation.
- State machine flowchart generation.
- Support to forward errors to Sentry.
- New `cloudconfig` handler for the `AzureCluster` controller that creates the required cloudconfig files in the Storage Account.
- Add --service.registry.mirrors flag for setting registry mirror domains.
- New `subnet` handler for the `AzureCluster` controller that creates the node pool subnet.

### Changed

- Allow tenant cluster to be created without built-in workers.
- Changed how the Azure authentication works when connecting to a different Subscription than the Control Plane's one.
- Restricted storage account access to the local VNET only.
- Removed the flatcar migration state machine transitions.
- Calculate CIDR for a new Tenant Cluster using a local resource rather than getting it from `kubernetesd`.
- Migrate the `vmsscheck` guards to use the Azure client factory.
- Use `0.1.0` tag for `k8s-api-heahtz` image.
- Use `0.2.0` tag for `k8s-setup-network-env` image.
- Use fixed value for registry domain (docker.io) and mirrors (giantswarm.azurecr.io).
- Replace --service.registrydomain with --service.registry.domain.

### Removed

- The Azure MSI extension for linux is not deployed anymore.
- The local calico kubernetes manifests are removed. We use the `k8scloudconfig` ones now.

## [4.1.0] - 2020-06-24

### Changed

- Use VNet gateway for egress traffic of worker VMSS instances

### Fixed

- Make the rate limit circuit breaker to only inspect response HTTP status code if there were no errors doing the request.

### Removed

- Migrate the `vmsscheck` guards to use the Azure client factory.
- Move NGINX IC LoadBalancer Service management from azure-operator to nginx-ingress-controller app.

## [4.0.1] 2020-05-20

## Fixed

- Avoid blocking all egress traffic from workers during flatcar migration.

## [4.0.0] 2020-05-05

### Added

- Add network policy.

### Changed

- Replace CoreOS VM image to Flatcar with manual migration.
- Move containerPort values from deployment to `values.yaml`.



### Changed

- Migrated to go modules.
- Use ARM nested templates instead of relying on Github when using linked templates.



## [3.0.7] 2020-04-28

### Added

- Add new instance types: Standard_E8a_v4 and Standard_E8as_v4.
- Some parameters have now defaults so it's easier to run the operator locally.

### Fixed

- Fix for outdated error matching that was preventing clusters from being bootstrapped.

### Changed

- Reduce number of Azure API calls when creating, updating and scaling clusters which lowers the risk of exceeding Azure API rate limits and hitting error 429.
- Collectors that expose Azure metrics have been migrated to its own repository.



## [3.0.6] 2020-04-09

### Fixed

- Removed usage of LastModelApplied field of the VMSS Instance type.



## [3.0.5] 2020-04-08

### Added

- Add azure-operator version to ARM parameters.
- Added `autorest` http decorator to hold back when Azure API responds with "HTTP 429 Too Many Requests".

### Changed

- Improved the discovery of new nodes.



## [3.0.1] 2020-04-02

### Added

- Added process to keep watching for failed instances on the VMSS.

### Fixed

- Fixed workers' over-provisioning during cluster creation.
- Fixed `wait-for-domains` cloud init script.

### Changed

- Upgraded the Azure SDK and Service API endpoints.
- Retrieve component versions from releases.
- Only roll nodes when they aren't in sync with provider operator.



[Unreleased]: https://github.com/giantswarm/azure-operator/compare/v5.22.0...HEAD
[5.22.0]: https://github.com/giantswarm/azure-operator/compare/v5.21.0...v5.22.0
[5.21.0]: https://github.com/giantswarm/azure-operator/compare/v5.20.0...v5.21.0
[5.20.0]: https://github.com/giantswarm/azure-operator/compare/v5.19.0...v5.20.0
[5.19.0]: https://github.com/giantswarm/azure-operator/compare/v5.18.0...v5.19.0
[5.18.0]: https://github.com/giantswarm/azure-operator/compare/v5.17.0...v5.18.0
[5.17.0]: https://github.com/giantswarm/azure-operator/compare/v5.16.0...v5.17.0
[5.16.0]: https://github.com/giantswarm/azure-operator/compare/v5.15.0...v5.16.0
[5.15.0]: https://github.com/giantswarm/azure-operator/compare/v5.14.0...v5.15.0
[5.14.0]: https://github.com/giantswarm/azure-operator/compare/v5.13.0...v5.14.0
[5.13.0]: https://github.com/giantswarm/azure-operator/compare/v5.12.0...v5.13.0
[5.12.0]: https://github.com/giantswarm/azure-operator/compare/v5.11.0...v5.12.0
[5.11.0]: https://github.com/giantswarm/azure-operator/compare/v5.10.2...v5.11.0
[5.10.2]: https://github.com/giantswarm/azure-operator/compare/v5.10.1...v5.10.2
[5.10.1]: https://github.com/giantswarm/azure-operator/compare/v5.10.0...v5.10.1
[5.10.0]: https://github.com/giantswarm/azure-operator/compare/v5.9.0...v5.10.0
[5.9.0]: https://github.com/giantswarm/azure-operator/compare/v5.8.1...v5.9.0
[5.8.1]: https://github.com/giantswarm/azure-operator/compare/v5.8.0...v5.8.1
[5.8.0]: https://github.com/giantswarm/azure-operator/compare/v5.7.2...v5.8.0
[5.7.2]: https://github.com/giantswarm/azure-operator/compare/v5.7.1...v5.7.2
[5.7.1]: https://github.com/giantswarm/azure-operator/compare/v5.7.0...v5.7.1
[5.7.0]: https://github.com/giantswarm/azure-operator/compare/v5.6.0...v5.7.0
[5.6.0]: https://github.com/giantswarm/azure-operator/compare/v5.5.2...v5.6.0
[5.5.2]: https://github.com/giantswarm/azure-operator/compare/v5.5.1...v5.5.2
[5.5.1]: https://github.com/giantswarm/azure-operator/compare/v5.5.0...v5.5.1
[5.5.0]: https://github.com/giantswarm/azure-operator/compare/v5.4.0...v5.5.0
[5.4.0]: https://github.com/giantswarm/azure-operator/compare/v5.3.0...v5.4.0
[5.3.0]: https://github.com/giantswarm/azure-operator/compare/v5.2.1...v5.3.0
[5.2.1]: https://github.com/giantswarm/azure-operator/compare/v5.2.0...v5.2.1
[5.2.0]: https://github.com/giantswarm/azure-operator/compare/v5.1.0...v5.2.0
[5.1.0]: https://github.com/giantswarm/azure-operator/compare/v5.0.0...v5.1.0
[5.0.0]: https://github.com/giantswarm/azure-operator/compare/v5.0.0-beta7...v5.0.0
[5.0.0-beta7]: https://github.com/giantswarm/azure-operator/compare/v5.0.0-beta6...v5.0.0-beta7
[5.0.0-beta6]: https://github.com/giantswarm/azure-operator/compare/v5.0.0-beta5...v5.0.0-beta6
[5.0.0-beta5]: https://github.com/giantswarm/azure-operator/compare/v5.0.0-beta4...v5.0.0-beta5
[5.0.0-beta4]: https://github.com/giantswarm/azure-operator/compare/v5.0.0-beta2...v5.0.0-beta4
[5.0.0-beta2]: https://github.com/giantswarm/azure-operator/compare/v5.0.0-beta1...v5.0.0-beta2
[5.0.0-beta1]: https://github.com/giantswarm/azure-operator/compare/v5.0.0-alpha4...v5.0.0-beta1
[5.0.0-alpha4]: https://github.com/giantswarm/azure-operator/compare/v5.0.0-alpha3...v5.0.0-alpha4
[5.0.0-alpha3]: https://github.com/giantswarm/azure-operator/compare/v5.0.0-alpha2...v5.0.0-alpha3
[5.0.0-alpha2]: https://github.com/giantswarm/azure-operator/compare/v5.0.0-alpha1...v5.0.0-alpha2
[5.0.0-alpha1]: https://github.com/giantswarm/azure-operator/compare/v4.2.0...v5.0.0-alpha1
[4.2.0]: https://github.com/giantswarm/azure-operator/compare/v4.1.0...v4.2.0
[4.1.0]: https://github.com/giantswarm/azure-operator/compare/v4.0.1...v4.1.0
[4.0.1]: https://github.com/giantswarm/azure-operator/compare/v4.0.0...v4.0.1
[4.0.0]: https://github.com/giantswarm/azure-operator/compare/v3.0.7...v4.0.0
[3.0.7]: https://github.com/giantswarm/azure-operator/compare/v3.0.6...v3.0.7
[3.0.6]: https://github.com/giantswarm/azure-operator/compare/v3.0.5...v3.0.6
[3.0.5]: https://github.com/giantswarm/azure-operator/compare/v3.0.1...v3.0.5
[3.0.1]: https://github.com/giantswarm/azure-operator/compare/v1.0.0...v3.0.1
[1.0.0]: https://github.com/giantswarm/azure-operator/releases/tag/v1.0.0
