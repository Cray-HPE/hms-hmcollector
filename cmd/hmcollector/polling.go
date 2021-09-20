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

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"github.com/Cray-HPE/hms-hmcollector/internal/hmcollector"
	"github.com/Cray-HPE/hms-hmcollector/internal/river_collector"
	rf "github.com/Cray-HPE/hms-smd/pkg/redfish"
)

var (
	gigabyteCollector river_collector.GigabyteRiverCollector
	intelCollector    river_collector.IntelRiverCollector
	hpeCollector      river_collector.HPERiverCollector

	endpointMutex      sync.Mutex
	endpoints          []EndpointWithCollector
	nonPolledEndpoints []*rf.RedfishEPDescription
)

func collectData(pendingEndpoints <-chan EndpointWithCollector, jsonPayloads chan<- jsonPayload) {
	for endpoint := range pendingEndpoints {
		endpointLogger := logger.With(zap.String("xname", endpoint.Endpoint.ID))

		// If SMA is not contact-able, there's no point grabbing any of this.
		// We'll consume the chan data but we won't act on it.

		if (!smaOK) {
			logger.Debug("SMA off, Skipping poll of ",zap.String("endpoint",endpoint.Endpoint.ID))
			continue
		}

		for _, telemetryType := range telemetryTypes {
			fullURL := endpoint.RiverCollector.GetPayloadURLForTelemetryType(endpoint.Endpoint, telemetryType)
			logger.Debug("collectData(): ",zap.String("Endpoint URL",fullURL))
			payloadBytes, statusCode, err := doHTTPAction(endpoint.Endpoint, http.MethodGet, fullURL, nil)

			if err != nil {
				// Error already logged, just stop progressing.
				continue
			}

			// Keep the LastContacted time up to date.
			*endpoint.LastContacted = time.Now()

			if statusCode != http.StatusOK {
				// Just log when this happens, it might be nothing.
				endpointLogger.Warn("Got unexpected status code from endpoint.",
					zap.String("fullURL", fullURL),
					zap.Int("statusCode", statusCode))
			}

			newEvents := river_collector.GetEventsForPayload(endpoint.RiverCollector, payloadBytes, endpoint.Endpoint,
				telemetryType)
			logger.Debug("collectData(): ",zap.Int("num events",len(newEvents)))

			for _, event := range newEvents {
				var finalEvents hmcollector.Events
				finalEvents.Context = endpoint.Endpoint.ID
				finalEvents.Events = append(finalEvents.Events, event)

				finalEventsJSON, _ := json.Marshal(finalEvents)
				finalEventsString := string(finalEventsJSON)

				if finalEventsString != "" {
					payload := jsonPayload{
						payload: finalEventsString,
					}

					switch event.MessageId {
					case river_collector.PowerMessageID:
						payload.topic = "cray-telemetry-power"
					case river_collector.VoltageMessageID:
						payload.topic = "cray-telemetry-voltage"
					case river_collector.TemperatureMessageID:
						payload.topic = "cray-telemetry-temperature"
					case river_collector.FanMessageID:
						payload.topic = "cray-telemetry-fan"
					default:
						logger.Error("Encountered message with unknown MessageId!", zap.Any("event", event))
						continue
					}

					jsonPayloads <- payload
				}
			}
		}
	}
}

func monitorPollingEndpoints() {
	var endpointWaitGroup sync.WaitGroup

	for Running {
		hsmEndpointsCache := map[string]*rf.RedfishEPDescription{}
		HSMEndpointsLock.Lock()
		for id, ep := range HSMEndpoints {
			hsmEndpointsCache[id] = ep
		}
		HSMEndpointsLock.Unlock()
		

		for _, endpoint := range hsmEndpointsCache {
			// Check to make sure we don't already know about this endpoint.
			endpointIsKnown := false
			for _, knownEndpoint := range endpoints {
				if endpoint.ID == knownEndpoint.Endpoint.ID {
					endpointIsKnown = true
					break
				}
			}

			if endpointIsKnown {
				continue
			}

			// Check to make sure we aren't ignoring this endpoint.
			endpointIsIgnored := false
			for _, ignoredEndpoint := range nonPolledEndpoints {
				if endpoint.ID == ignoredEndpoint.ID {
					endpointIsIgnored = true
					break
				}
			}

			if endpointIsIgnored {
				continue
			}

			endpointWaitGroup.Add(1)

			// Now we have to determine which of these endpoints are River and if they are what type of River.
			// Do as much of this in parallel as possible.
			go func(endpoint *rf.RedfishEPDescription) {
				defer endpointWaitGroup.Done()

				var newEndpoint *EndpointWithCollector
				var model string

				// Gigabyte
				fullURL := fmt.Sprintf("https://%s/redfish/v1/Systems/Self", endpoint.FQDN)
				payloadBytes, statusCode, err := doHTTPAction(endpoint, http.MethodGet, fullURL, nil)
				if err == nil && statusCode == http.StatusOK {
					var systems hmcollector.Systems
					decodeErr := json.Unmarshal(payloadBytes, &systems)
					if decodeErr != nil {
						logger.Error("Failed to decode model information, will not poll endpoint.",
							zap.Any("endpoint", endpoint),
							zap.Error(err))

						return
					}

					model = systems.Model
				}

				// Intel
				fullURL = "https://" + endpoint.FQDN + "/redfish/v1/Chassis/RackMount/Baseboard"
				payloadBytes, statusCode, err = doHTTPAction(endpoint, http.MethodGet, fullURL, nil)
				if err == nil && statusCode == http.StatusOK {
					var chassis hmcollector.Chassis
					decodeErr := json.Unmarshal(payloadBytes, &chassis)
					if decodeErr != nil {
						logger.Error("Failed to decode model information, will not poll endpoint.",
							zap.Any("endpoint", endpoint),
							zap.Error(err))

						return
					}

					model = chassis.Model
				}

				// HPE
				fullURL = "https://" + endpoint.FQDN + "/redfish/v1/Chassis/1"
				payloadBytes, statusCode, err = doHTTPAction(endpoint, http.MethodGet, fullURL, nil)
				if err == nil && statusCode == http.StatusOK {
					var chassis hmcollector.Chassis
					decodeErr := json.Unmarshal(payloadBytes, &chassis)
					if decodeErr != nil {
						logger.Error("Failed to decode model information, will not poll endpoint.",
							zap.Any("endpoint", endpoint),
							zap.Error(err))

						return
					}

					model = chassis.Model
				}

				// At this point we have what we need, use process of elimination.
				if strings.HasPrefix(model, "R272") ||
					strings.HasPrefix(model, "R282") ||
					strings.HasPrefix(model, "H262") {
					logger.Info("Found Gigabyte endpoint eligible for polling.",
						zap.Any("endpoint", endpoint))
					newEndpoint = &EndpointWithCollector{
						Endpoint:       endpoint,
						RiverCollector: gigabyteCollector,
					}
				} else if model == "S2600BPB" || model == "S2600WFT" {
					logger.Info("Found Intel endpoint eligible for polling.", zap.Any("endpoint", endpoint))
					newEndpoint = &EndpointWithCollector{
						Endpoint:       endpoint,
						RiverCollector: intelCollector,
					}
				} else if strings.HasPrefix(model, "ProLiant") {
					logger.Info("Found HPE endpoint eligible for polling.",
						zap.Any("endpoint", endpoint))
					newEndpoint = &EndpointWithCollector{
						Endpoint:       endpoint,
						RiverCollector: hpeCollector,
					}
				} else {
					// We have to ignore it if we can't determine what it is.
					logger.Warn("Unable to determine model number from endpoint, "+
						"which means this endpoint is either Mountain or an unknown River type",
						zap.Any("endpoint", endpoint),
						zap.String("model", model))
				}

				if newEndpoint != nil {
					newEndpoint.Model = model

					newEndpoint.LastContacted = &time.Time{}
					*newEndpoint.LastContacted = time.Now()

					endpointMutex.Lock()
					endpoints = append(endpoints, *newEndpoint)
					endpointMutex.Unlock()
				} else {
					logger.Warn("Failed to collect model information, will not poll endpoint.",
						zap.Any("endpoint", endpoint),
						zap.Error(err))

					// Add this endpoint to a list of endpoints what won't be polled.
					endpointMutex.Lock()
					nonPolledEndpoints = append(nonPolledEndpoints, endpoint)
					endpointMutex.Unlock()
				}
			}(endpoint)
		}

		endpointWaitGroup.Wait()

		// Use a channel in case we have long refresh intervals so we don't wait around for things to exit.
		select {
		case <-PollingShutdown:
			break
		case <-time.After(EndpointRefreshInterval * time.Second):
			continue
		}
	}

	logger.Info("Polling endpoint monitor routine shutdown.")
}

func doPolling() {
	// Setup a background goroutine to monitor for the comings (and goings) of endpoints.
	go monitorPollingEndpoints()

	logger.Info("Collecting data from endpoints at interval.", zap.Int("pollingInterval", *pollingInterval))

	// Setup channels for pendingEndpoints and JSON payloads.
	pendingEndpoints := make(chan EndpointWithCollector, len(endpoints))
	jsonPayloads := make(chan jsonPayload, 10000)

	defer close(pendingEndpoints)
	defer close(jsonPayloads)
	defer WaitGroup.Done()

	// Start up the pool of workers
	for worker := 1; worker <= NumWorkers; worker++ {
		go collectData(pendingEndpoints, jsonPayloads)
		go processData(jsonPayloads)
	}

	for Running {
		for _, endpoint := range endpoints {
			pendingEndpoints <- endpoint
		}
		time.Sleep(time.Duration(*pollingInterval) * time.Second)
	}

	logger.Info("Polling routine shutdown.")
}
