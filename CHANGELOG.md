# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## Fixed

- Make the rate limit circuit breaker to only inspect response HTTP status code if there were no errors doing the request.

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



[Unreleased]: https://github.com/giantswarm/aws-operator/compare/v4.0.0...HEAD

[4.0.0]: https://github.com/giantswarm/aws-operator/compare/v3.0.7...v4.0.0

[3.0.7]: https://github.com/giantswarm/aws-operator/compare/v3.0.6...v3.0.7
[3.0.6]: https://github.com/giantswarm/aws-operator/compare/v3.0.5...v3.0.6
[3.0.5]: https://github.com/giantswarm/aws-operator/compare/v3.0.1...v3.0.5
[3.0.1]: https://github.com/giantswarm/aws-operator/compare/v1.0.0...v3.0.1

[1.0.0]: https://github.com/giantswarm/aws-operator/releases/tag/v1.0.0
