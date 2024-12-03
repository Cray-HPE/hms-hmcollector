// MIT License
//
// (C) Copyright [2020-2022] Hewlett Packard Enterprise Development LP
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
	"io"
	"io/ioutil"
	"net/http"

	"github.com/Cray-HPE/hms-certs/pkg/hms_certs"
	rf "github.com/Cray-HPE/hms-smd/pkg/redfish"
)

func GetEndpointList(httpClient *hms_certs.HTTPClientPair, gatewayUrl string) ([]rf.RedfishEPDescription, error) {
	var RedfishEndpoints RedfishEndpoints

	request,qerr := http.NewRequest("GET",gatewayUrl + "/hsm/v2/Inventory/RedfishEndpoints",nil)
	if (qerr != nil) {
		return nil,fmt.Errorf("Unable to create HTTP request: %v",qerr)
	}
	rsp,serr := httpClient.Do(request)
	defer func () {
		if rsp != nil && rsp.Body != nil {
				_, _ = io.Copy(io.Discard, rsp.Body)
				rsp.Body.Close()
		}
	}()

	if (serr != nil) {
		return nil, fmt.Errorf("Error in HTTP request: %v",serr)
	}
	payloadBytes, perr := ioutil.ReadAll(rsp.Body)
	if (perr != nil) {
		return nil, fmt.Errorf("unable to do HTTP GET for RF endpoints: %s", perr)
	}

	stringPayloadBytes := string(payloadBytes)
	if stringPayloadBytes != "" {
		// If we've made it to here we have all we need, unmarshal.
		jsonErr := json.Unmarshal(payloadBytes, &RedfishEndpoints)
		if jsonErr != nil {
			return nil, fmt.Errorf("unable to unmarshal payload: %s", jsonErr)
		}
	}

	return RedfishEndpoints.Endpoints, nil
}
