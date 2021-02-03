// MIT License
//
// (C) Copyright [2020-2021] Hewlett Packard Enterprise Development LP
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
// THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
// OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
// ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

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
