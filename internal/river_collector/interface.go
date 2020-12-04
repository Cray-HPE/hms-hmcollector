// Copyright 2020 Hewlett Packard Enterprise Development LP

package river_collector

import (
	"stash.us.cray.com/HMS/hms-hmcollector/internal/hmcollector"
	rf "stash.us.cray.com/HMS/hms-smd/pkg/redfish"
)

type TelemetryType string

// Universal truths
const (
	MessageRegistryName  = "CrayTelemetry"
	PowerMessageID       = MessageRegistryName + "." + "Power"
	VoltageMessageID     = MessageRegistryName + "." + "Voltage"
	TemperatureMessageID = MessageRegistryName + "." + "Temperature"

	TelemetryTypePower   TelemetryType = "Power"
	TelemetryTypeThermal TelemetryType = "Thermal"
)

// This is just a generic no-op interface
type MockRiverCollector struct{}

// Vendor specific
type IntelRiverCollector struct{}
type GigabyteRiverCollector struct{}
type HPERiverCollector struct{}

type RiverCollector interface {
	GetPayloadURLForTelemetryType(endpoint *rf.RedfishEPDescription, telemetryType TelemetryType) string

	ParseJSONPowerEvents(payloadBytes []byte, location string) []hmcollector.Event
	ParseJSONThermalEvents(payloadBytes []byte, location string) []hmcollector.Event
}

func GetEventsForPayload(collector RiverCollector, payloadBytes []byte, endpoint *rf.RedfishEPDescription,
	telemetryType TelemetryType) []hmcollector.Event {
	var newEvents []hmcollector.Event
	switch telemetryType {
	case TelemetryTypePower:
		newEvents = collector.ParseJSONPowerEvents(payloadBytes, endpoint.ID)
	case TelemetryTypeThermal:
		newEvents = collector.ParseJSONThermalEvents(payloadBytes, endpoint.ID)
	}

	return newEvents
}
