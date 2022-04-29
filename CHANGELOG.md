# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

<!--
Guiding Principles:
* Changelogs are for humans, not machines.
* There should be an entry for every single version.
* The same types of changes should be grouped.
* Versions and sections should be linkable.
* The latest version comes first.
* The release date of each version is displayed.
* Mention whether you follow Semantic Versioning.

Types of changes:
Added - for new features
Changed - for changes in existing functionality
Deprecated - for soon-to-be removed features
Removed - for now removed features
Fixed - for any bug fixes
Security - in case of vulnerabilities
-->

## [2.18.0] - 2022-04-28

### Changed

- Updated hms-hmcollector to build using GitHub Actions instead of Jenkins.
- Pull images from artifactory.algol60.net instead of arti.dev.cray.com.

## [2.17.0] - 2022-04-04

### Changed

- CASMHMS-5479 - Look for 'CrayFabricCriticalTelemetry' instead of 'CrayFabricCritTelemetry' prefixes for the fabric critical telemetry records.

## [2.16.0] - 2021-12-15

### Changed

- CASMHMS-5296 - Rebuild container image to resolve CVE-2021-3520.

## [2.15.0] - 2021-11-30

### Added

- CASMHMS-5268 - Added telemetry support for HPE PDUs.

## [2.14.0] - 2021-10-27

### Changed

- Updated CT test RPM dependency.

## [2.13.0] - 2021-10-19

### Changed

- CASMHMS-5165 - Split the deployment of redfish event and streaming telemetry processing functionality from the deployment of subscription management and telemetry polling so they can be independently scaled.

## [2.12.8] - 2021-10-11

### Added

- CASMHMS-5055 - Added hmcollector CT test RPM.

## [2.12.7] - 2021-09-20

### Changed

- Changed the docker image to run as the user nobody

## [2.12.6] - 2021-09-17

### Fixed

- CASMHMS-5156 - Added locking around `HSMEndpointsLock` map to prevent the collector from panicing after concurrent map iteration and write.
- Expose resource limits and requests in the collector Helm chart, so they can be overridable via customizations.yaml.

## [2.12.5] - 2021-08-10

### Changed

- Added GitHub configuration files.

## [2.12.4] - 2021-07-27

### Changed

- Github transition phase 3. Remove stash references.

## [2.12.3] - 2021-07-21

### Changed

- Add support for building within the CSM Jenkins.

## [2.12.2] - 2021-07-12

### Security

- CASMHMS-4933 - Updated base container images for security updates.

## [2.12.1] - 2021-06-09

### Changed

- Fixed issues with fan and power telemetry.

## [2.12.0] - 2021-06-09

### Changed

- Bump minor version for CSM 1.2 release branch

## [2.11.0] - 2021-06-07

### Changed

- Bump minor version for CSM 1.1 release branch

## [2.10.6] - 2021-05-17

### Changed

- Added support for humidity and liquidflow telemetry.

## [2.10.5] - 2021-05-12

### Changed

- Update subscriptions and RF event processing to deal with iLO issues.

## [2.10.4] - 2021-05-04

### Changed

- Updated docker-compose files to pull images from Artifactory instead of DTR.

## [2.10.3] - 2021-04-21

### Changed

- Updated Dockerfiles to pull base images from Artifactory instead of DTR.

## [2.10.2] - 2021-04-21

### Changed

- Fixed HTTP response leaks which can lead to istio OOM.

## [2.10.1] - 2021-02-05

### Changed

- Added User-Agent headers to outbound HTTP requests.

## [2.10.0] - 2021-02-03

### Changed

- Update Copyright/License and re-vendor go packages

## [2.9.0] - 2021-01-14

### Changed

- Updated license file.

## [2.8.9] - 2020-12-16

### Changed

- CASMHMS-4241 - Stop collecting/polling River env telemetry if SMA is not available.

## [2.8.8] - 2020-11-13

### Added

- CASMHMS-4215 - Added final CA bundle configmap handling to Helm chart.

# [2.8.7] - 2020-10-20

### Security

- CASMHMS-4105 - Updated base Golang Alpine image to resolve libcrypto vulnerability.

# [2.8.6] - 2020-10-16

### Added

- CASMHMS-3763 - Support for TLS certs for Redfish operations.

# [2.8.5] - 2020-09-25

### Added

- CASMHMS-3947 - Support for HPE DL325

# [2.8.4] - 2020-09-10

### Security

- CASMHMS-3993 - Updated hms-hmcollector to use trusted baseOS images.

# [2.8.3] - 2020-09-08

### Fixed

- CASMHMS-3615 - made credentials refresh if the request ever comes back unauthorized.

# [2.8.2] - 2020-07-20

### Changed

- CASMHMS-3783 - Re-enabling by default Telemetry Polling for all River nodes
- Whether polling is enabled can now configured via a Helm chart value override
- Specifying the value `hmcollector_enable_polling=false` as an override in your Loftsman manifest will disable polling in the collector.

# [2.8.1] - 2020-07-17

### Changed

- CASMHMS-3772 - Disabling Telemetry Polling for all River nodes

# [2.8.0] - 2020-06-29

### Added

- CASMHMS-3607 - Added CT smoke test for hmcollector.

# [2.7.6] - 2020-06-05

### Changed

- CASMHMS-3260 - Now is online installable, upgradable, downgradable.

# [2.7.5] - 2020-05-26

### Changed

- CASMHMS-3433 - bumped resource limits.

# [2.7.4] - 2020-04-27

### Changed

- CASMHMS-2955 - use trusted baseOS images.

# [2.7.3] - 2020-03-30

### Changed

- CASMHMS-2818 - check that subscriptions are still present on controllers.
- CASMHMS-3200 - verify that subscription context remains valid.

# [2.7.2] - 2020-03-13

### Fixed

- Prevented inclusion of empty telemetrey payloads from Gigabyte.

# [2.7.1] - 2020-03-05

### Fixed

- Updated collection of Gigabyte model numbers to include everything known as of now.

# [2.7.0] - 2020-03-04

### Fixed

- Removed Sarama completely in favor of Confluent library built on librdkafka. It was observed that producing large numbers of messages simultaneously would result in only a fraction of them actually making it onto the bus.

# [2.6.1] - 2020-02-07

### Added

- CASMHMS-2642 - Updated liveness/readiness probes.

# [2.6.0] - 2020-01-09

### Added

- Initial version of changelog.
