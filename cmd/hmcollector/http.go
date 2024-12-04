// MIT License
//
// (C) Copyright [2020-2021,2024] Hewlett Packard Enterprise Development LP
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

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	rf "github.com/Cray-HPE/hms-smd/pkg/redfish"
	"go.uber.org/zap"
)

func doHTTPAction(endpoint *rf.RedfishEPDescription, method string,
                  fullURL string, body *[]byte) (payloadBytes []byte, statusCode int, err error) {

	var request *http.Request

	endpointLogger := logger.With(zap.String("xname", endpoint.ID))

	if (body == nil) {
		request, _ = http.NewRequest(method, fullURL, nil)
	} else {
		request, _ = http.NewRequest(method, fullURL, bytes.NewBuffer(*body))
	}

	request = request.WithContext(ctx)

	if !(endpoint.User == "" && endpoint.Password == "") {
		request.SetBasicAuth(endpoint.User, endpoint.Password)
	}

	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	var resp *http.Response
	var doErr error

	rfClientLock.RLock()	// TODO: Are locks really necessary here?
	resp, doErr = rfClient.Do(request)
	rfClientLock.RUnlock()
	defer DrainAndCloseResponseBody(resp)
	if doErr != nil {
		endpointLogger.Error("Unable to do request!", 
			zap.Error(doErr), zap.String("fullURL", fullURL))
		err = fmt.Errorf("unable to do request: %w", doErr)
		return
	}
	statusCode = resp.StatusCode

	// Now we need to check to see if we got a 401 (Unauthorized) or 403 (Forbidden).
	// If so, refresh the credentials in Vault.
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		endpointLogger.Warn("Got Unauthorized response from endpoint, refreshing credentials...")

		// Keep track of the previous credentials to see if they change.
		previousUsername := endpoint.User
		previousPassword := endpoint.Password

		updateErr := updateEndpointWithCredentials(endpoint)
		if updateErr != nil {
			endpointLogger.Error("Attempted to update credentials for endpoint but could not!",
				zap.Error(updateErr))
			err = fmt.Errorf("unable to update credentials for endpoint: %w", updateErr)
			return
		} else {
			if endpoint.User != previousUsername || endpoint.Password != previousPassword {
				endpointLogger.Info("Successfully updated credentials for endpoint. Re-attempting action...")

				// Recursively call this function again.
				return doHTTPAction(endpoint, method, fullURL, body)
			} else {
				endpointLogger.Warn("Refreshed credentials but they did not change.")
				err = fmt.Errorf("refreshed credentials did not change from original")
				return
			}
		}
	}

	// Get the payload.
	payloadBytes, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		endpointLogger.Error("Unable to read response body!",
			zap.Error(readErr), zap.String("fullURL", fullURL))
		err = fmt.Errorf("unable to read response body: %w", readErr)
		return
	}

	return
}
