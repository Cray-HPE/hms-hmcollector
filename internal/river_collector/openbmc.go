// MIT License
//
// (C) Copyright [2020-2021,2024-2025] Hewlett Packard Enterprise Development LP
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
	"strconv"
	"time"

	"github.com/Cray-HPE/hms-hmcollector/internal/hmcollector"
	rf "github.com/Cray-HPE/hms-smd/v2/pkg/redfish"
)

func (collector OpenBMCRiverCollector) ParseJSONPowerEvents(payloadBytes []byte,
	location string) (events []hmcollector.Event) {
	timestamp := time.Now().Format(time.RFC3339)

	var power hmcollector.Power
	decodeErr := json.Unmarshal(payloadBytes, &power)
	if decodeErr != nil {
		return
	}

	// PowerControl
	powerControlEvent := hmcollector.Event{
		MessageId:      PowerMessageID,
		EventTimestamp: timestamp,
		Oem:            &hmcollector.Sensors{},
	}
	powerControlEvent.Oem.TelemetrySource = "River"
	hasPowerControlData := false

	for _, PowerControl := range power.PowerControl {
		if PowerControl.Name == "Chassis Power Control" {
			payload := hmcollector.CrayJSONPayload{
				Index: new(int16),
			}

			payload.Timestamp = timestamp
			payload.Location = location
			payload.PhysicalContext = "Chassis"
			payload.DeviceSpecificContext = PowerControl.Name
			indexU64, _ := strconv.ParseUint(PowerControl.MemberId, 10, 8)
			*payload.Index = int16(indexU64)
			payload.Value = strconv.FormatFloat(PowerControl.PowerConsumedWatts, 'f', -1, 64)

			powerControlEvent.Oem.Sensors = append(powerControlEvent.Oem.Sensors, payload)

			hasPowerControlData = true
		}
	}

	if hasPowerControlData {
		events = append(events, powerControlEvent)
	}

	// PowerSupplies
	powerSupplyEvent := hmcollector.Event{
		MessageId:      PowerMessageID,
		EventTimestamp: timestamp,
		Oem:            &hmcollector.Sensors{},
	}
	powerSupplyEvent.Oem.TelemetrySource = "River"

	for _, PowerSupply := range power.PowerSupplies {
		payload := hmcollector.CrayJSONPayload{
			Index: new(int16),
		}

		payload.Timestamp = timestamp
		payload.Location = location
		payload.PhysicalContext = "PowerSupplyBay"
		payload.DeviceSpecificContext = PowerSupply.Name
		indexU64, _ := strconv.ParseUint(PowerSupply.MemberId, 10, 8)
		*payload.Index = int16(indexU64)
		payload.Value = strconv.FormatFloat(PowerSupply.PowerOutputWatts, 'f', -1, 64)

		powerSupplyEvent.Oem.Sensors = append(powerSupplyEvent.Oem.Sensors, payload)
	}

	if len(power.PowerSupplies) > 0 {
		events = append(events, powerSupplyEvent)
	}

	// Voltages
	voltageEvent := hmcollector.Event{
		MessageId:      VoltageMessageID,
		EventTimestamp: timestamp,
		Oem:            &hmcollector.Sensors{},
	}
	voltageEvent.Oem.TelemetrySource = "River"

	for _, Voltage := range power.Voltages {
		payload := hmcollector.CrayJSONPayload{
			Index: new(int16),
		}

		payload.Timestamp = timestamp
		payload.Location = location
		payload.PhysicalContext = "SystemBoard"
		payload.DeviceSpecificContext = Voltage.Name
		payload.Value = strconv.FormatFloat(Voltage.ReadingVolts, 'f', -1, 64)

		voltageEvent.Oem.Sensors = append(voltageEvent.Oem.Sensors, payload)
	}

	if len(power.Voltages) > 0 {
		events = append(events, voltageEvent)
	}

	return
}

func (collector OpenBMCRiverCollector) ParseJSONThermalEvents(payloadBytes []byte,
	location string) (events []hmcollector.Event) {
	timestamp := time.Now().Format(time.RFC3339)

	var thermal hmcollector.EnclosureThermal
	decodeErr := json.Unmarshal(payloadBytes, &thermal)
	if decodeErr != nil {
		return
	}

	// Fans
	fanEvent := hmcollector.Event{
		MessageId:      FanMessageID,
		EventTimestamp: timestamp,
		Oem:            &hmcollector.Sensors{},
	}
	fanEvent.Oem.TelemetrySource = "River"

	for _, Fan := range thermal.Fans {
		payload := hmcollector.CrayJSONPayload{
			Index: new(int16),
		}

		payload.Timestamp = timestamp
		payload.Location = location

		payload.DeviceSpecificContext = Fan.Name
		payload.PhysicalContext = "System"
		indexU64, _ := strconv.ParseUint(Fan.MemberId, 10, 8)
		*payload.Index = int16(indexU64)
		payload.Value = strconv.FormatFloat(Fan.Reading, 'f', -1, 64)

		fanEvent.Oem.Sensors = append(fanEvent.Oem.Sensors, payload)
	}

	if len(thermal.Fans) > 0 {
		events = append(events, fanEvent)
	}

	// Temperatures

	temperatureEvent := hmcollector.Event{
		MessageId:      TemperatureMessageID,
		EventTimestamp: timestamp,
		Oem:            &hmcollector.Sensors{},
	}
	temperatureEvent.Oem.TelemetrySource = "River"

	for _, Temperature := range thermal.Temperatures {
		payload := hmcollector.CrayJSONPayload{
			Index: new(int16),
		}

		payload.Timestamp = timestamp
		payload.Location = location

		payload.DeviceSpecificContext = Temperature.Name
		payload.PhysicalContext = Temperature.PhysicalContext
		indexU64, _ := strconv.ParseUint(Temperature.MemberId, 10, 8)
		*payload.Index = int16(indexU64)
		payload.Value = strconv.FormatFloat(Temperature.ReadingCelsius, 'f', -1, 64)

		temperatureEvent.Oem.Sensors = append(temperatureEvent.Oem.Sensors, payload)
	}

	if len(thermal.Temperatures) > 0 {
		events = append(events, temperatureEvent)
	}

	return
}

func (collector OpenBMCRiverCollector) GetPayloadURLForTelemetryType(endpoint *rf.RedfishEPDescription,
	telemetryType TelemetryType) []string {
	url := fmt.Sprintf("https://%s/redfish/v1/Chassis/Baseboard_0/%s", endpoint.FQDN, telemetryType)
	return []string{url}
}
