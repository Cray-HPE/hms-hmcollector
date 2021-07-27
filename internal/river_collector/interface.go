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
	"github.com/Cray-HPE/hms-hmcollector/internal/hmcollector"
	rf "github.com/Cray-HPE/hms-smd/pkg/redfish"
)

type TelemetryType string

// Universal truths
const (
	MessageRegistryName  = "CrayTelemetry"
	PowerMessageID       = MessageRegistryName + "." + "Power"
	VoltageMessageID     = MessageRegistryName + "." + "Voltage"
	TemperatureMessageID = MessageRegistryName + "." + "Temperature"
	FanMessageID         = MessageRegistryName + "." + "Fan"

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
