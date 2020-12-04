// Copyright 2020 Hewlett Packard Enterprise Development LP

package main

import (
	"bytes"
	"fmt"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"time"
	rf "stash.us.cray.com/HMS/hms-smd/pkg/redfish"
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

	//This is not retryablehttp, so implement the retries here

	for ix := 1; ix <= 3; ix ++ {
		rfClientLock.RLock()
		resp, doErr = rfClient.Do(request)
		rfClientLock.RUnlock()
		if (doErr == nil) {
			break
		}
		logger.Error("client.Do() ",zap.Int("attempt",ix),zap.Error(doErr))
		time.Sleep(time.Duration(*httpTimeout + (ix*2)) * time.Second)
	}

	if resp != nil {
		defer resp.Body.Close()
	}
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
