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

## [2.36.0] - 2025-01-29

### Security

- Update image and module dependencies

## [2.35.0] - 2025-01-08

### Added

- Added support for ppprof builds

## [2.34.0] - 2024-12-06

### Fixed

- Resolved some scaling/resource issues
- Updated version of Go to 1.23

## [2.33.0] - 2024-05-28

### Added

- CASMHMS-6208 - Handle Foxconn Paradise OpenBmc power events

## [2.32.0] - 2024-05-07

### Added

- CASMHMS-6202 - Added custom logging featues to log messages per xname

## [2.31.0] - 2024-04-19

### Changed

- CASMHMS-6127 - Fixed redfish subscription support by updating the Model for OpenBMC

## [2.30.0] - 2024-02-12

### Changed

- CASMHMS-6133 - Added support for polling telemetry data from OpenBMC Nodes

## [2.29.0] - 2024-01-24

### Changed

- CASMHMS-6127 - Added support for creating subscriptions on OpenBMC Nodes

## [2.28.0] - 2024-01-12

### Changed

- CASMHMS-6123 - In messages handle -1 for the fields: Index, ParentalIndex, and SubIndex.

## [2.27.0] - 2023-09-11

### Added

- CASMHMS-5831 - Handle new telemetry IDs from Olympus nCs and cCs

## [2.26.0] - 2023-04-19

### Changed

- CASMHMS-5685 - Changed to prune event subscriptions with stale destinations. Stale subscriptions happen when hardware is physically moved.

## [2.25.0] - 2023-01-19

### Changed

- CASMSMF-7120: Update for Cray Fabric Health Telemetry and Events to be sent to a common kafka topic cray-fabric-health instead of cray-fabric-health-events and cray-fabric-health-telemetry.

## [2.24.0] - 2023-01-10

### Added

- CASMSMF-7120: Added support for Cray Fabric Health Telemetry collection.

## [2.23.0] - 2022-11-07

### Changed

- CASMHMS-5824: Add a message key to kafka messages to ensure events are sent to the same Kafka partition. The message key is the BMC Xname concatenated with the Redfish Event Message ID. For example `x3000c0s11b4.EventLog.1.0.PowerStatusChange`. 

## [2.22.0] - 2022-10-05

### Changed 

- CASMHMS-5776: Switched over to use `ping` from the `iputils` package instead of from busybox to work around an issue
  were busybox `ping` does not have permission when running as non-root user.

## [2.21.0] - 2022-07-20

### Security

- Removed checked in certificate.
- Updated echo_server help information to include the steps to generate a certificate.

## [2.20.0] - 2022-07-06

### Changed

- Changed HSM v1 API references to v2

## [2.19.0] - 2022-05-16

### Changed

- added crayfabric topic.


## [2.18.0] - 2022-05-05

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
