# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Increase VMSS termination events timeout to 15 minutes.

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



[Unreleased]: https://github.com/giantswarm/azure-operator/compare/v5.3.0...HEAD
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
