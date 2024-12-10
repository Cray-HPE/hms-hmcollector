// MIT License
//
// (C) Copyright [2020-2024] Hewlett Packard Enterprise Development LP
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
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Cray-HPE/hms-hmcollector/internal/hmcollector"
)

// Unmarshalling events is complicated because of the fact that some redfish
// implementations do not return events based on the redfish standard.
func unmarshalEvents(bodyBytes []byte) (events hmcollector.Events, err error) {
	var jsonObj map[string]interface{}
	marshalErr := func(bb []byte, e error) {
		err = e
		logger.Error("Unable to unmarshal JSON payload",
			zap.ByteString("bodyString", bb),
			zap.Error(e),
		)
	}
	if e := json.Unmarshal(bodyBytes, &jsonObj); e != nil {
		marshalErr(bodyBytes, e)
		return
	}
	// We know we got some sort of JSON object.
	v, ok := jsonObj["Events"]
	if !ok {
		marshalErr(bodyBytes, fmt.Errorf("JSON payload missing Events"))
		return
	}
	delete(jsonObj, "Events")

	if newBodyBytes, e := json.Marshal(jsonObj); e != nil {
		marshalErr(bodyBytes, e)
		return
	} else if e = json.Unmarshal(newBodyBytes, &events); e != nil {
		marshalErr(newBodyBytes, e)
		return
	}

	// We now have the base events object, but without the Events array.
	// The variable v is holding this info right now. We need to process each
	// entry of this array individually, since the OriginOfCondition field may
	// be a string rather than a JSON object.

	if evBytes, e := json.Marshal(v); e != nil {
		marshalErr(bodyBytes, e)
		return
	} else {
		var evObjs []map[string]interface{}
		if e = json.Unmarshal(evBytes, &evObjs); e != nil {
			// Try parsing the bytes as a single event object
			// Paradise BMCs return an object instead of an array of objects for the Events field
			var singleEvObj map[string]interface{}
			if e2 := json.Unmarshal(evBytes, &singleEvObj); e2 != nil {
				// return the original parse error
				marshalErr(evBytes, e)
				return
			} else {
				// convert the result to an array
				evObjs = make([]map[string]interface{}, 1)
				evObjs[0] = singleEvObj
			}
		}
		for _, ev := range evObjs {
			s, ok := ev["OriginOfCondition"].(string)
			if ok {
				delete(ev, "OriginOfCondition")
			}
			tmp, e := json.Marshal(ev)
			var tmpEvent hmcollector.Event
			if e = json.Unmarshal(tmp, &tmpEvent); e != nil {
				marshalErr(tmp, e)
				return
			}
			if ok {
				// if OriginOfCondition was a string, we need
				// to put it into the unmarshalled event.
				tmpEvent.OriginOfCondition = &hmcollector.ResourceID{s}
			}
			events.Events = append(events.Events, tmpEvent)
		}
	}
	return
}

func parseRequest(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	DrainAndCloseRequestBody(r)
	if err != nil {
		logger.Error("Unable to read body!", zap.Error(err))
		return
	}

	if ce := logger.Check(zap.DebugLevel, "Got request."); ce != nil {
		// If debug-level log output isn't enabled or if zap's sampling would have
		// dropped this log entry, we don't allocate the slice that holds these
		// fields.
		ce.Write(
			zap.String("URL request path", r.URL.Path),
			zap.String("string(bodyBytes)", string(bodyBytes)),
		)
	}

	// Need to interrogate this payload to figure out what topic it needs to go to.
	// MessageId field is always Registry.Entry, for HMS the Entry is what determines the table.
	events, marshalErr := unmarshalEvents(bodyBytes)
	if marshalErr != nil {
		// Best to let the client know they dun goofed.
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte("JSON payload malformed!"))
		if err != nil {
			logger.Error("Unable to write error message back to client!", zap.Error(err))
		}
		return
	}

	// Need to verify the context and perhaps fix it if it is missing.
	if events.Context == "" {
		events.Context = filepath.Base(r.URL.Path)
	}

	// At this point we have every Event parsed into structures. Here's the thing: while at the time of this comment
	// we didn't have payloads where the Events array contained Event entries with different MessageIds, it's of course
	// possible that they could be. Therefore, we have to check each Event to determine what topic it needs to go on.
	// This can be done more efficiently, but for now I'm just going to construct a new Events array for each Event.
	for _, event := range events.Events {
		// Check to see if we need to override the timestamp.
		if *IgnoreProvidedTimestamp {
			for index, _ := range event.Oem.Sensors {
				event.Oem.Sensors[index].Timestamp = time.Now().Format(time.RFC3339)
			}
		}

		// Check each of the sensors for a valid PhysicalContext. If it's not mark it as such.
		// TODO: Move this logic into the PMDB C++ code.
		if event.Oem != nil {
			for index, _ := range event.Oem.Sensors {
				if event.Oem.Sensors[index].PhysicalContext == "" {
					event.Oem.Sensors[index].PhysicalContext = "INVALID"
				}
			}
		}

		// Start with the old Events structure and just remove the event entries to preserve things like Context.
		newEvents := events
		newEvents.Events = []hmcollector.Event{event}

		eventsBytes, marshalErr := json.Marshal(newEvents)
		if marshalErr != nil {
			logger.Error("Unable to marshal event into JSON!", zap.Any("event", event), zap.Error(err))
			return
		}
		eventsString := string(eventsBytes)

		kafkaMessageKey := fmt.Sprintf("%s.%s", events.Context, event.MessageId)

		// Figure out where she goes and SEND IT!
		if strings.HasPrefix(event.MessageId, "CrayTelemetry") {
			if strings.HasSuffix(event.MessageId, "Temperature") {
				writeToKafka("cray-telemetry-temperature", eventsString, &kafkaMessageKey)
			} else if strings.HasSuffix(event.MessageId, "Voltage") {
				writeToKafka("cray-telemetry-voltage", eventsString, &kafkaMessageKey)
			} else if strings.HasSuffix(event.MessageId, "Power") ||
				strings.HasSuffix(event.MessageId, "Current") {
				writeToKafka("cray-telemetry-power", eventsString, &kafkaMessageKey)
			} else if strings.HasSuffix(event.MessageId, "Energy") {
				writeToKafka("cray-telemetry-energy", eventsString, &kafkaMessageKey)
			} else if strings.HasSuffix(event.MessageId, "Fan") ||
				strings.HasSuffix(event.MessageId, "Rotational") {
				writeToKafka("cray-telemetry-fan", eventsString, &kafkaMessageKey)
			} else if strings.HasSuffix(event.MessageId, "Pressure") {
				writeToKafka("cray-telemetry-pressure", eventsString, &kafkaMessageKey)
			} else if strings.HasSuffix(event.MessageId, "LiquidFlow") {
				writeToKafka("cray-telemetry-liquidflow", eventsString, &kafkaMessageKey)
			} else if strings.HasSuffix(event.MessageId, "Humidity") {
				writeToKafka("cray-telemetry-humidity", eventsString, &kafkaMessageKey)
			} else if strings.HasSuffix(event.MessageId, "Metric") {
				writeToKafka("cray-telemetry-metric", eventsString, &kafkaMessageKey)
			} else if strings.HasSuffix(event.MessageId, "PowerFactor") {
				writeToKafka("cray-telemetry-powerfactor", eventsString, &kafkaMessageKey)
			} else if strings.HasSuffix(event.MessageId, "Frequency") {
				writeToKafka("cray-telemetry-frequency", eventsString, &kafkaMessageKey)
			} else if strings.HasSuffix(event.MessageId, "Percent") {
				writeToKafka("cray-telemetry-percent", eventsString, &kafkaMessageKey)
			} else {
				logger.Error("Event MessageId has registry prefix CrayTelemetry, but the SensorType is unknown!",
					zap.Any("event", event))
			}
		} else if strings.HasPrefix(event.MessageId, "CrayFabricTelemetry") {
			writeToKafka("cray-fabric-telemetry", eventsString, &kafkaMessageKey)
		} else if strings.HasPrefix(event.MessageId, "CrayFabricPerfTelemetry") {
			writeToKafka("cray-fabric-perf-telemetry", eventsString, &kafkaMessageKey)
		} else if strings.HasPrefix(event.MessageId, "CrayFabricCritTelemetry") ||
			strings.HasPrefix(event.MessageId, "CrayFabricCriticalTelemetry") {
			writeToKafka("cray-fabric-crit-telemetry", eventsString, &kafkaMessageKey)
		} else if strings.HasPrefix(event.MessageId, "CrayFabricHealth") {
			writeToKafka("cray-fabric-health", eventsString, &kafkaMessageKey)
		} else {
			// If we get to this point then we don't have a specific topic this should go on,
			// dump it on the generic one.
			// TODO should normal redfish events have the message key? What about HSM services
			writeToKafka("cray-dmtf-resource-event", eventsString, &kafkaMessageKey)
		}
	}
}

// Kubernetes liveness probe - if this responds with anything other than success (code <400) it will cause the
// pod to be restarted (eventually).
func doLiveness(w http.ResponseWriter, r *http.Request) {
	defer DrainAndCloseRequestBody(r)

	w.WriteHeader(http.StatusNoContent)
}

// Kubernetes liveness probe - if this responds with anything other than success (code <400) multiple times in
// a row it will cause the pod to be restarted.  Only fail this probe for issues that we expect a restart to fix.
func doReadiness(w http.ResponseWriter, r *http.Request) {
	defer DrainAndCloseRequestBody(r)

	ready := true

	// If the Kafka bus isn't good, then return not ready since any incoming data will be dropped.  A restart may not
	// fix this, but it will also keep any traffic from being routed here.
	// NOTE: typically if the Kafka bus is down, the brokers will be created but the producers will not be
	// instantiated yet.
	numBrokers := 0
	numInit := 0
	for _, thisBroker := range kafkaBrokers {
		numBrokers++
		if thisBroker.KafkaProducer != nil {
			numInit++
		}
	}
	ready = (numBrokers == numInit) && (numBrokers != 0)

	if ready {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

// HealthResponse - used to report service health stats
type HealthResponse struct {
	Vault                 string
	Kafka                 string
	PollingEndpointStatus []EndpointStatus
}

type EndpointStatus struct {
	Xname         string
	Model         string
	LastContacted time.Time
}

func doHealth(w http.ResponseWriter, r *http.Request) {
	defer DrainAndCloseRequestBody(r)

	// Only allow 'GET' calls.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Actively check the dependencies and report back the results.
	var stats HealthResponse
	// see if the connection to the vault is established
	if *VaultEnabled {
		if compCredStore == nil {
			stats.Vault = "Vault enabled but not initialized"
		} else {
			stats.Vault = "Vault enabled and initialized"
		}
	} else {
		stats.Vault = "Vault not enabled"
	}

	// See what the kafka connection looks like.
	numBrokers := 0
	numInit := 0
	for _, thisBroker := range kafkaBrokers {
		numBrokers++
		if thisBroker.KafkaProducer != nil {
			numInit++
		}
	}
	stats.Kafka = fmt.Sprintf("Brokers: %d, Initialized:%d", numBrokers, numInit)

	// Build up the endpoint status.
	for _, endpoint := range endpoints {
		endpointStatus := EndpointStatus{
			Xname:         endpoint.Endpoint.ID,
			Model:         endpoint.Model,
			LastContacted: *endpoint.LastContacted,
		}

		stats.PollingEndpointStatus = append(stats.PollingEndpointStatus, endpointStatus)
	}

	statusJSON, _ := json.Marshal(stats)

	// Write the output.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(statusJSON)
}

func doRest() {
	portStr := strconv.Itoa(*restPort)
	srv := &http.Server{Addr: ":" + portStr}

	// Setup health resource for readiness/liveliness probes.
	http.HandleFunc("/readiness", doReadiness)
	http.HandleFunc("/liveness", doLiveness)
	// Health function for Kubernetes liveness/readiness.
	http.HandleFunc("/health", doHealth)

	go func() {
		defer WaitGroup.Done()
		err := srv.ListenAndServe()

		if err != nil {
			if err.Error() == "http: Server closed" {
				logger.Info("REST HTTP server shutdown")
			} else {
				logger.Panic("Unable to start REST HTTP server!", zap.Error(err))
			}
		}
	}()

	logger.Info("REST server started.", zap.String("portStr", portStr))

	RestSRV = srv
}
