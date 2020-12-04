// Copyright 2020 Hewlett Packard Enterprise Development LP

package hmcollector

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	hmshttp "stash.us.cray.com/HMS/hms-go-http-lib"
	"stash.us.cray.com/HMS/hms-smd/pkg/redfish"
)

func GetEndpointList(httpClient *retryablehttp.Client, gatewayUrl string) ([]rf.RedfishEPDescription, error) {
	var RedfishEndpoints RedfishEndpoints

	request := hmshttp.NewHTTPRequest(gatewayUrl + "/hsm/v1/Inventory/RedfishEndpoints")
	request.Client = httpClient

	payloadBytes, _, err := request.DoHTTPAction()
	if err != nil {
		return nil, fmt.Errorf("unable to do HTTP GET for RF endpoints: %s", err)
	}

	stringPayloadBytes := string(payloadBytes)
	if stringPayloadBytes != "" {
		// If we've made it to here we have all we need, unmarshal.
		jsonErr := json.Unmarshal(payloadBytes, &RedfishEndpoints)
		if jsonErr != nil {
			err = fmt.Errorf("unable to unmarshal payload: %s", jsonErr)
			return nil, err
		}
	}

	return RedfishEndpoints.Endpoints, nil
}
