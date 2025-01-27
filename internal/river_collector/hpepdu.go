// MIT License
//
// (C) Copyright [2021,2024-2025] Hewlett Packard Enterprise Development LP
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

package river_collector

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Cray-HPE/hms-hmcollector/internal/hmcollector"
	rf "github.com/Cray-HPE/hms-smd/v2/pkg/redfish"
)

func (collector HPEPDURiverCollector) ParseJSONPowerEvents(payloadBytes []byte,
	location string) (events []hmcollector.Event) {
	var payload map[string]interface{}

	decodeErr := json.Unmarshal(payloadBytes, &payload)
	if decodeErr != nil {
		return
	}

	if name, ok := payload["Name"]; ok {
		nameStr := name.(string)
		if strings.Contains(nameStr, "Outlet") {
			events = collector.parseOutletPayload(payloadBytes, location)
		} else if strings.Contains(nameStr, "Branch") {
			events = collector.parseBranchPayload(payloadBytes, location)
		} else if strings.Contains(nameStr, "Mains") {
			events = collector.parseMainPayload(payloadBytes, location)
		}
	}
	return

}

func (collector HPEPDURiverCollector) parseOutletPayload(payloadBytes []byte,
	location string) (events []hmcollector.Event) {
	timestamp := time.Now().Format(time.RFC3339)
	var outlet hmcollector.Outlet

	decodeErr := json.Unmarshal(payloadBytes, &outlet)
	if decodeErr != nil {
		return
	}

	// Use Oid to get the outlet number because "Id" isn't always populated
	f := strings.FieldsFunc(outlet.Links.Oid,
		func(c rune) bool { return !unicode.IsNumber(c) })
	indexU64, _ := strconv.ParseUint(f[len(f)-1], 10, 8)
	index := int16(indexU64)

	// Outlet Power
	powerEvent := hmcollector.Event{
		MessageId:      PowerMessageID,
		EventTimestamp: timestamp,
		Oem:            &hmcollector.Sensors{},
	}
	powerEvent.Oem.TelemetrySource = "River"

	payloadP := hmcollector.CrayJSONPayload{
		Index: new(int16),
	}
	payloadP.Timestamp = timestamp
	payloadP.Location = location + "p0"
	payloadP.ParentalContext = "PDU"
	payloadP.PhysicalContext = "Outlet"
	payloadP.PhysicalSubContext = "Output"
	*payloadP.Index = index
	payloadP.Value = strconv.FormatFloat(outlet.PowerWatts.Reading, 'f', -1, 64)

	powerEvent.Oem.Sensors = append(powerEvent.Oem.Sensors, payloadP)

	// Outlet Current
	currentEvent := hmcollector.Event{
		MessageId:      CurrentMessageID,
		EventTimestamp: timestamp,
		Oem:            &hmcollector.Sensors{},
	}
	currentEvent.Oem.TelemetrySource = "River"

	payloadC := hmcollector.CrayJSONPayload{
		Index: new(int16),
	}
	payloadC.Timestamp = timestamp
	payloadC.Location = location + "p0"
	payloadC.ParentalContext = "PDU"
	payloadC.PhysicalContext = "Outlet"
	payloadC.PhysicalSubContext = "Output"
	*payloadC.Index = index
	payloadC.Value = strconv.FormatFloat(outlet.CurrentAmps.Reading, 'f', -1, 64)

	currentEvent.Oem.Sensors = append(currentEvent.Oem.Sensors, payloadC)

	// Outlet Voltage
	voltageEvent := hmcollector.Event{
		MessageId:      VoltageMessageID,
		EventTimestamp: timestamp,
		Oem:            &hmcollector.Sensors{},
	}
	voltageEvent.Oem.TelemetrySource = "River"

	payloadV := hmcollector.CrayJSONPayload{
		Index: new(int16),
	}
	payloadV.Timestamp = timestamp
	payloadV.Location = location + "p0"
	payloadV.ParentalContext = "PDU"
	payloadV.PhysicalContext = "Outlet"
	payloadV.PhysicalSubContext = "Output"
	*payloadV.Index = index
	payloadV.Value = strconv.FormatFloat(outlet.Voltage.Reading, 'f', -1, 64)

	voltageEvent.Oem.Sensors = append(voltageEvent.Oem.Sensors, payloadV)

	// Outlet Energy
	energyEvent := hmcollector.Event{
		MessageId:      EnergyMessageID,
		EventTimestamp: timestamp,
		Oem:            &hmcollector.Sensors{},
	}
	energyEvent.Oem.TelemetrySource = "River"

	payloadE := hmcollector.CrayJSONPayload{
		Index: new(int16),
	}
	payloadE.Timestamp = timestamp
	payloadE.Location = location + "p0"
	payloadE.ParentalContext = "PDU"
	payloadE.PhysicalContext = "Outlet"
	payloadE.PhysicalSubContext = "Output"
	*payloadE.Index = index
	payloadE.Value = strconv.FormatFloat(outlet.EnergykWh.Reading, 'f', -1, 64)

	energyEvent.Oem.Sensors = append(energyEvent.Oem.Sensors, payloadE)

	events = append(events, powerEvent, currentEvent, voltageEvent, energyEvent)

	sensor, ok := collector.Sensors[outlet.Links.Oid]
	if !ok {
		return
	}
	if sensor.LastPowerState == outlet.PowerState {
		// No power state change. We're done parsing.
		return
	}

	sensor.LastPowerState = outlet.PowerState
	collector.Sensors[outlet.Links.Oid] = sensor

	// Outlet Power State Event
	message := fmt.Sprintf("The power state of resource %s has changed to type %s.", outlet.Links.Oid, outlet.PowerState)
	scEvent := hmcollector.Event{
		EventTimestamp:    timestamp,
		Severity:          "OK",
		Message:           message,
		MessageId:         ResourceMessageID,
		MessageArgs:       []string{outlet.Links.Oid, outlet.PowerState},
		Context:           location,
		OriginOfCondition: &hmcollector.ResourceID{Oid: outlet.Links.Oid},
	}

	events = append(events, scEvent)

	return
}

func (collector HPEPDURiverCollector) parseBranchPayload(payloadBytes []byte,
	location string) (events []hmcollector.Event) {
	timestamp := time.Now().Format(time.RFC3339)
	var branch hmcollector.Circuit

	decodeErr := json.Unmarshal(payloadBytes, &branch)
	if decodeErr != nil {
		return
	}

	// Branches use letters A-F for their name but numbers 1-6 for their ids.
	// However, the Id field isn't always populated so just convert the branch
	// letter into an index.
	f := strings.Fields(branch.Name)
	branchChar := f[len(f)-1]
	index := int16(branchChar[0]) - 64
	deviceSpecificContext := ""

	// Get the Line#ToLine# string for this branch for context.
	for name, _ := range branch.PolyPhasePowerWatts {
		if strings.Contains(name, "Line") {
			deviceSpecificContext = name + "-BR" + branchChar
		}
	}

	// Branch Current
	currentEvent := hmcollector.Event{
		MessageId:      CurrentMessageID,
		EventTimestamp: timestamp,
		Oem:            &hmcollector.Sensors{},
	}
	currentEvent.Oem.TelemetrySource = "River"

	payloadC := hmcollector.CrayJSONPayload{
		Index: new(int16),
	}
	payloadC.Timestamp = timestamp
	payloadC.Location = location + "p0"
	payloadC.ParentalContext = "PDU"
	payloadC.PhysicalContext = "Branch"
	payloadC.PhysicalSubContext = "Input"
	payloadC.DeviceSpecificContext = deviceSpecificContext
	*payloadC.Index = index
	payloadC.Value = strconv.FormatFloat(branch.CurrentAmps.Reading, 'f', -1, 64)

	currentEvent.Oem.Sensors = append(currentEvent.Oem.Sensors, payloadC)

	// Branch Energy
	energyEvent := hmcollector.Event{
		MessageId:      EnergyMessageID,
		EventTimestamp: timestamp,
		Oem:            &hmcollector.Sensors{},
	}
	energyEvent.Oem.TelemetrySource = "River"

	payloadE := hmcollector.CrayJSONPayload{
		Index: new(int16),
	}
	payloadE.Timestamp = timestamp
	payloadE.Location = location + "p0"
	payloadE.ParentalContext = "PDU"
	payloadE.PhysicalContext = "Branch"
	payloadE.PhysicalSubContext = "Input"
	payloadE.DeviceSpecificContext = deviceSpecificContext
	*payloadE.Index = index
	payloadE.Value = strconv.FormatFloat(branch.EnergykWh.Reading, 'f', -1, 64)

	energyEvent.Oem.Sensors = append(energyEvent.Oem.Sensors, payloadE)

	events = append(events, currentEvent, energyEvent)

	return
}

func (collector HPEPDURiverCollector) parseMainPayload(payloadBytes []byte,
	location string) (events []hmcollector.Event) {
	timestamp := time.Now().Format(time.RFC3339)
	var pduMain hmcollector.Circuit

	decodeErr := json.Unmarshal(payloadBytes, &pduMain)
	if decodeErr != nil {
		return
	}

	// Phase Power
	powerEvent := hmcollector.Event{
		MessageId:      PowerMessageID,
		EventTimestamp: timestamp,
		Oem:            &hmcollector.Sensors{},
	}
	powerEvent.Oem.TelemetrySource = "River"

	// Get the sensor index
	sensorListP := []string{}
	for name, _ := range pduMain.PolyPhasePowerWatts {
		sensorListP = append(sensorListP, name)
	}
	sort.Strings(sensorListP)

	for index, name := range sensorListP {
		sensor := pduMain.PolyPhasePowerWatts[name]
		payload := hmcollector.CrayJSONPayload{
			Index: new(int16),
		}
		payload.Timestamp = timestamp
		payload.Location = location + "p0"
		payload.ParentalContext = "PDU"
		payload.PhysicalContext = "Phase"
		payload.PhysicalSubContext = "Input"
		payload.DeviceSpecificContext = name
		*payload.Index = int16(index)
		payload.Value = strconv.FormatFloat(sensor.Reading, 'f', -1, 64)

		powerEvent.Oem.Sensors = append(powerEvent.Oem.Sensors, payload)
	}

	// Phase Voltage
	voltageEvent := hmcollector.Event{
		MessageId:      VoltageMessageID,
		EventTimestamp: timestamp,
		Oem:            &hmcollector.Sensors{},
	}
	voltageEvent.Oem.TelemetrySource = "River"

	// Get the sensor index
	sensorListV := []string{}
	for name, _ := range pduMain.PolyPhaseVoltage {
		sensorListV = append(sensorListV, name)
	}
	sort.Strings(sensorListV)

	for index, name := range sensorListV {
		sensor := pduMain.PolyPhaseVoltage[name]
		payload := hmcollector.CrayJSONPayload{
			Index: new(int16),
		}
		payload.Timestamp = timestamp
		payload.Location = location + "p0"
		payload.ParentalContext = "PDU"
		payload.PhysicalContext = "Phase"
		payload.PhysicalSubContext = "Input"
		payload.DeviceSpecificContext = name
		*payload.Index = int16(index)
		payload.Value = strconv.FormatFloat(sensor.Reading, 'f', -1, 64)

		voltageEvent.Oem.Sensors = append(voltageEvent.Oem.Sensors, payload)
	}

	// Line Current
	currentEvent := hmcollector.Event{
		MessageId:      CurrentMessageID,
		EventTimestamp: timestamp,
		Oem:            &hmcollector.Sensors{},
	}
	currentEvent.Oem.TelemetrySource = "River"

	// Get the sensor index
	sensorListC := []string{}
	for name, _ := range pduMain.PolyPhaseCurrentAmps {
		sensorListC = append(sensorListC, name)
	}
	sort.Strings(sensorListC)

	for index, name := range sensorListC {
		sensor := pduMain.PolyPhaseCurrentAmps[name]
		payload := hmcollector.CrayJSONPayload{
			Index: new(int16),
		}
		payload.Timestamp = timestamp
		payload.Location = location + "p0"
		payload.ParentalContext = "PDU"
		payload.PhysicalContext = "Line"
		payload.PhysicalSubContext = "Input"
		payload.DeviceSpecificContext = name
		*payload.Index = int16(index)
		payload.Value = strconv.FormatFloat(sensor.Reading, 'f', -1, 64)

		currentEvent.Oem.Sensors = append(currentEvent.Oem.Sensors, payload)
	}

	events = append(events, powerEvent, voltageEvent, currentEvent)

	return
}

func (collector HPEPDURiverCollector) ParseJSONThermalEvents(payloadBytes []byte,
	location string) (events []hmcollector.Event) {
	return
}

func (collector HPEPDURiverCollector) GetPayloadURLForTelemetryType(endpoint *rf.RedfishEPDescription,
	telemetryType TelemetryType) []string {
	if telemetryType != TelemetryTypePower {
		return []string{}
	}
	urls := make([]string, 0, len(collector.Sensors))
	for id, _ := range collector.Sensors {
		url := fmt.Sprintf("https://%s%s", endpoint.FQDN, id)
		urls = append(urls, url)
	}
	return urls
}
