# Hardware Management Collector (hmcollector)

The hmcollector listens for power on and off events and collects telemetry
data.

## How it Works

There are two pods `cray-hms-hmcollector-ingress` and `cray-hms-hmcollector-poll`.

At start up `cray-hms-hmcollector-poll` creates redfish subscriptions on the
BMCs. It gets the the list of BMCs from HSM's redfish endpoints
(/hsm/v2/Inventory/RedfishEndpoints). After setting up the subscriptions it starts
polling telemetry data via redfish for the hardware that does not support
telemetry subscriptions.

The `cray-hms-hmcollector-ingress` acts as an HTTP server that the BMCs make
HTTP POST requests to when the BMC has some information to report. When this
call back happens hmcollector parses the request and puts the information on
the kafka bus for other CSM services to use. For example, the HSM services
uses this information to change the State field on its Components data.

## Configuration

To edit one of the configuration variables on a deployed hmcollector run the
following commands
```
kubectl -n services edit deployment/cray-hms-hmcollector-ingress
kubectl -n services edit deployment/cray-hms-hmcollector-poll
```

Then make changes to the given variable and save and exit. The given pods will
restart if changes have been made. These changes will last until the hmcollector
is upgraded.

| Environment Variable    | Description                                                                                                                            |
| --------------------    | -------------------------------------------------------------------------------------------------------------------------------------- |
| HMCOLLECTOR_CA_URI      | URI of the CA cert bundle                                                                                                              |
| HSM_REFRESH_INTERVAL    | The interval to check HSM for new Redfish Endpoints in seconds                                                                         |
| LOG_LEVEL               | Sets the log level: ERROR, INFO, DEBUG                                                                                                 |
| LOG_MODES               | Custom logging options. Supported values: errors                                                                                       |
| LOG_XNAMES              | Enable logging of ingress messages for the given BMC xnames. For example: "x9000c1s2b3,x9000c1s2b4" (Note that spaces are not allowed) |
| POLLING_ENABLED         | Enables or disables polling                                                                                                            |
| POLLING_INTERVAL        | The polling interval in seconds for redfish telemetry polling                                                                          |
| PRUNE_OLD_SUBSCRIPTIONS | Enable pruning old subscriptions that contain the wrong xname for the given BMC                                                        |
| REST_ENABLED            | Controls whether or not the HTTP server is started                                                                                     |
| REST_URL                | The Address for Redfish events to target                                                                                               |
| RF_STREAMING_ENABLED    | Enable creating subscriptions for telemetry                                                                                            |
| RF_SUBSCRIBE_ENABLED    | Enable creating redfish subscriptions                                                                                                  |
| SM_URL                  | The address of the State Manager (aka hsm, aka smd)                                                                                    |
| VAULT_ADDR              | The address of the vault server                                                                                                        |
| VAULT_ENABLED           | Enable using vault for credentials                                                                                                     |
| VAULT_KEYPATH           | Keypath for Vault credentials                                                                                                          |

## Redfish Examples

Get a list of the subscriptions
```
curl -ks -X GET -u root:${BMC_PASSWD} https://${XNAME}/redfish/v1/EventService/Subscriptions | jq
```

Get subscription details (example from Mountain hardware)
```
curl -ks -X GET -u root:${BMC_PASSWD} https://${XNAME}/redfish/v1/EventService/Subscriptions/1 | jq
```

Delete a subscription
```
curl -ksi -X DELETE -u root:${BMC_PASSWD} https://${XNAME}/redfish/v1/EventService/Subscriptions/1
```

Create a subscription for power events
```
curl -ksi -X POST -H "Content-Type: application/json" \
    -u root:${BMC_PASSWD} \
    -d '{"Destination":"http://10.94.200.71/'${XNAME}'", "Protocol": "Redfish", "EventTypes": ["Alert", "StatusChange", "ResourceUpdated", "ResourceAdded", "ResourceRemoved"], "Context":"'${XNAME}'" }' \
    https://${XNAME}/redfish/v1/EventService/Subscriptions
```

Some hardware only support the Alert event type.
```
curl -ksi -X POST -H "Content-Type: application/json" \
    -u root:${BMC_PASSWD} \
    -d '{"Destination":"http://10.94.200.71/'${XNAME}'", "Protocol": "Redfish", "EventTypes": ["Alert"], "Context":"'${XNAME}'" }' \
    https://${XNAME}/redfish/v1/EventService/Subscriptions
```

Create a subscription for telemetry information
```
curl -ksi -X POST -H "Content-Type: application/json" \
    -u root:${BMC_PASSWD} \
    -d '{"Destination":"http://10.94.200.71/'${XNAME}'", "Protocol": "Redfish", "EventTypes": ["Alert", "StatusChange", "ResourceUpdated", "ResourceAdded", "ResourceRemoved"], "Context":"'${XNAME}'", "RegistryPrefixes": ["CrayTelemetry"] }' \
    https://${XNAME}/redfish/v1/EventService/Subscriptions
```
```
curl -ksi -X POST -H "Content-Type: application/json" \
    -u root:${BMC_PASSWD} \
    -d '{"Destination":"http://10.94.200.71/'${XNAME}'", "Protocol": "Redfish", "Context":"'${XNAME}'", "RegistryPrefixes": ["CrayTelemetry"] }' \
    https://${XNAME}/redfish/v1/EventService/Subscriptions
```

## kubernetes actions

Restart the hmcollector
```
kubectl -n services rollout restart deployment cray-hms-hmcollector-ingress
kubectl -n services rollout restart deployment cray-hms-hmcollector-poll
```

Scale out the hmcollector ingress service from 3 pods to 6 pods. There needs to be one worker available for each pod.
```
kubectl -n services scale deployment cray-hms-hmcollector-ingress --current-replicas=3 --replicas=6
```

## CT Testing

CT tests have not yet been written for hmcollector.

