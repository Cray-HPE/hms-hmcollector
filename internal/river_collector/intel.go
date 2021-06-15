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

package river_collector

import (
	"encoding/json"
	"fmt"
	"stash.us.cray.com/HMS/hms-hmcollector/internal/hmcollector"
	rf "stash.us.cray.com/HMS/hms-smd/pkg/redfish"
	"strconv"
	"time"
)

func (collector IntelRiverCollector) ParseJSONPowerEvents(payloadBytes []byte,
	location string) (events []hmcollector.Event) {
	timestamp := time.Now().Format(time.RFC3339)

	var power hmcollector.Power
	decodeErr := json.Unmarshal(payloadBytes, &power)
	if decodeErr != nil {
		return
	}

	powerEvent := hmcollector.Event{
		MessageId:      PowerMessageID,
		EventTimestamp: time.Now().Format(time.RFC3339),
		Oem:            &hmcollector.Sensors{},
	}
	powerEvent.Oem.TelemetrySource = "River"

	// PowerControl
	for _, PowerControl := range power.PowerControl {
		if PowerControl.Name == "Server Power Control" {
			payload := hmcollector.CrayJSONPayload{
				Index: new(uint8),
			}

			payload.Timestamp = timestamp
			payload.Location = location
			payload.PhysicalContext = "Chassis"
			payload.DeviceSpecificContext = "Chassis_PowerControl"
			indexU64, _ := strconv.ParseUint(PowerControl.MemberId, 10, 8)
			*payload.Index = uint8(indexU64)
			payload.Value = strconv.FormatFloat(PowerControl.PowerConsumedWatts, 'f', -1, 64)

			powerEvent.Oem.Sensors = append(powerEvent.Oem.Sensors, payload)
		}
	}

	events = append(events, powerEvent)

	voltageEvent := hmcollector.Event{
		MessageId:      VoltageMessageID,
		EventTimestamp: timestamp,
		Oem:            &hmcollector.Sensors{},
	}
	voltageEvent.Oem.TelemetrySource = "River"

	// PowerSupplies
	for _, PowerSupply := range power.PowerSupplies {
		payload := hmcollector.CrayJSONPayload{
			Index: new(uint8),
		}

		payload.Timestamp = timestamp
		payload.Location = location
		payload.PhysicalContext = "PowerSupplyBay"
		payload.DeviceSpecificContext = "PowerSupplyBay_" + PowerSupply.MemberId
		indexU64, _ := strconv.ParseUint(PowerSupply.MemberId, 10, 8)
		*payload.Index = uint8(indexU64)
		payload.Value = strconv.FormatFloat(PowerSupply.LineInputVoltage, 'f', -1, 64)

		voltageEvent.Oem.Sensors = append(voltageEvent.Oem.Sensors, payload)
	}

	// Voltages
	for _, Voltage := range power.Voltages {
		payload := hmcollector.CrayJSONPayload{
			Index: new(uint8),
		}

		payload.Timestamp = timestamp
		payload.Location = location
		payload.PhysicalContext = "SystemBoard"
		payload.DeviceSpecificContext = Voltage.Name[3:len(Voltage.Name)]
		payload.Value = strconv.FormatFloat(Voltage.ReadingVolts, 'f', -1, 64)

		voltageEvent.Oem.Sensors = append(voltageEvent.Oem.Sensors, payload)
	}

	events = append(events, voltageEvent)

	return
}

func (collector IntelRiverCollector) ParseJSONThermalEvents(payloadBytes []byte,
	location string) (events []hmcollector.Event) {
	timestamp := time.Now().Format(time.RFC3339)

	var thermal hmcollector.EnclosureThermal
	decodeErr := json.Unmarshal(payloadBytes, &thermal)
	if decodeErr != nil {
		return
	}

	temperatureEvent := hmcollector.Event{
		MessageId:      TemperatureMessageID,
		EventTimestamp: timestamp,
		Oem:            &hmcollector.Sensors{},
	}
	temperatureEvent.Oem.TelemetrySource = "River"

	for _, Temperature := range thermal.Temperatures {
		payload := hmcollector.CrayJSONPayload{
			Index:         new(uint8),
			ParentalIndex: new(uint8),
		}

		payload.Timestamp = timestamp
		payload.Location = location
		payload.PhysicalContext = "Baseboard"
		payload.Value = strconv.FormatFloat(Temperature.ReadingCelsius, 'f', -1, 64)
		payload.DeviceSpecificContext = Temperature.Name

		temperatureEvent.Oem.Sensors = append(temperatureEvent.Oem.Sensors, payload)
	}

	events = append(events, temperatureEvent)

	return
}

func (collector IntelRiverCollector) GetPayloadURLForTelemetryType(endpoint *rf.RedfishEPDescription,
	telemetryType TelemetryType) string {
	return fmt.Sprintf("https://%s/redfish/v1/Chassis/RackMount/Baseboard/%s", endpoint.FQDN, telemetryType)
}
