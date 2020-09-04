# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Add monitoring label
- Add provider independent controllers to manage labeling and setting owner references in other provider dependent objects.
- Export container logs for e2e tests to azure analytics.

### Changed

- Decouple `Service` from controllers using an slice of controllers.
- Retry failed ARM deployments regardless of the checksum check.
- Master instances now use a dedicated NAT gateway for egress traffic.
- Removed the ETCD Private Load Balancer, reusing the API public one for ETCD traffic (needed by HA masters).
- Updated CAPI to `v0.3.8` and CAPZ to `v0.4.7`.

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



[Unreleased]: https://github.com/giantswarm/azure-operator/compare/v4.2.0...HEAD
[4.2.0]: https://github.com/giantswarm/azure-operator/compare/v4.1.0...v4.2.0
[4.1.0]: https://github.com/giantswarm/azure-operator/compare/v4.0.1...v4.1.0
[4.0.1]: https://github.com/giantswarm/azure-operator/compare/v4.0.0...v4.0.1
[4.0.0]: https://github.com/giantswarm/azure-operator/compare/v3.0.7...v4.0.0
[3.0.7]: https://github.com/giantswarm/azure-operator/compare/v3.0.6...v3.0.7
[3.0.6]: https://github.com/giantswarm/azure-operator/compare/v3.0.5...v3.0.6
[3.0.5]: https://github.com/giantswarm/azure-operator/compare/v3.0.1...v3.0.5
[3.0.1]: https://github.com/giantswarm/azure-operator/compare/v1.0.0...v3.0.1
[1.0.0]: https://github.com/giantswarm/azure-operator/releases/tag/v1.0.0
