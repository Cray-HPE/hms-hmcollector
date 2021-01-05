// Copyright 2020 Hewlett Packard Enterprise Development LP

package river_collector

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"stash.us.cray.com/HMS/hms-hmcollector/internal/hmcollector"
	rf "stash.us.cray.com/HMS/hms-smd/pkg/redfish"
)

func (collector HPERiverCollector) ParseJSONPowerEvents(payloadBytes []byte,
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
		payload := hmcollector.CrayJSONPayload{
			Index: new(uint8),
		}

		payload.Timestamp = timestamp
		payload.Location = location
		payload.PhysicalContext = "PowerSupplyBay"
		indexU64, _ := strconv.ParseUint(PowerControl.MemberId, 10, 8)
		*payload.Index = uint8(indexU64)
		payload.Value = strconv.FormatFloat(PowerControl.PowerMetrics.AverageConsumedWatts, 'f', -1,
			64)

		powerEvent.Oem.Sensors = append(powerEvent.Oem.Sensors, payload)
	}

	if len(power.PowerControl) > 0 {
		events = append(events, powerEvent)
	}

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
		indexU64, _ := strconv.ParseUint(PowerSupply.MemberId, 10, 8)
		*payload.Index = uint8(indexU64)
		payload.Value = strconv.FormatFloat(PowerSupply.LineInputVoltage, 'f', -1, 64)

		voltageEvent.Oem.Sensors = append(voltageEvent.Oem.Sensors, payload)
	}

	if len(power.Voltages) > 0 {
		events = append(events, voltageEvent)
	}

	return
}

func (collector HPERiverCollector) ParseJSONThermalEvents(payloadBytes []byte,
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

	// Fans
	for _, Fan := range thermal.Fans {
		payload := hmcollector.CrayJSONPayload{
			Index: new(uint8),
		}

		payload.Timestamp = timestamp
		payload.Location = location

		payload.DeviceSpecificContext = Fan.Name
		payload.PhysicalContext = "System"
		indexU64, _ := strconv.ParseUint(Fan.MemberId, 10, 8)
		*payload.Index = uint8(indexU64)
		payload.Value = strconv.FormatFloat(Fan.Reading, 'f', -1, 64)

		temperatureEvent.Oem.Sensors = append(temperatureEvent.Oem.Sensors, payload)
	}

	// Temperatures
	for _, Temperature := range thermal.Temperatures {
		payload := hmcollector.CrayJSONPayload{
			Index: new(uint8),
		}

		payload.Timestamp = timestamp
		payload.Location = location

		payload.DeviceSpecificContext = Temperature.Name
		payload.PhysicalContext = Temperature.PhysicalContext
		indexU64, _ := strconv.ParseUint(Temperature.MemberId, 10, 8)
		*payload.Index = uint8(indexU64)
		payload.Value = strconv.FormatFloat(Temperature.ReadingCelsius, 'f', -1, 64)

		temperatureEvent.Oem.Sensors = append(temperatureEvent.Oem.Sensors, payload)
	}

	if len(thermal.Fans) > 0 || len(thermal.Temperatures) > 0 {
		events = append(events, temperatureEvent)
	}

	return
}

func (collector HPERiverCollector) GetPayloadURLForTelemetryType(endpoint *rf.RedfishEPDescription,
	telemetryType TelemetryType) string {
	return fmt.Sprintf("https://%s/redfish/v1/Chassis/1/%s", endpoint.FQDN, telemetryType)
}
