// Copyright 2020 Hewlett Packard Enterprise Development LP

package hmcollector

import (
	"fmt"
)

// Cray Specific

type CrayJSONPayload struct {
	Timestamp             string
	Location              string
	ParentalContext       string `json:",omitempty"`
	ParentalIndex         *uint8 `json:",omitempty"`
	PhysicalContext       string
	Index                 *uint8 `json:",omitempty"`
	PhysicalSubContext    string `json:",omitempty"`
	DeviceSpecificContext string `json:",omitempty"`
	SubIndex              *uint8 `json:",omitempty"`
	Value                 string
}

type Sensors struct {
	Sensors         []CrayJSONPayload
	TelemetrySource string
}

type ResourceID struct {
	Oid string `json:"@odata.id"`
}

type Event struct {
	EventType         string      `json:",omitempty"`
	EventId           string      `json:",omitempty"`
	EventTimestamp    string      `json:",omitempty"`
	Severity          string      `json:",omitempty"`
	Message           string      `json:",omitempty"`
	MessageId         string      `json:",omitempty"`
	MessageArgs       []string    `json:",omitempty"`
	Context           string      `json:",omitempty"` // Older versions
	OriginOfCondition *ResourceID `json:",omitempty"`
	Oem               *Sensors    `json:",omitempty"` // Used only on for Cray RF events
}

type Events struct {
	OContext     string  `json:"@odata.context,omitempty"`
	Oid          string  `json:"@odata.id,omitempty"`
	Otype        string  `json:"@odata.type,omitempty"`
	Id           string  `json:"Id,omitempty"`
	Name         string  `json:"Name,omitempty"`
	Context      string  `json:"Context,omitempty"` // Later versions
	Description  string  `json:"Description,omitempty"`
	Events       []Event `json:"Events,omitempty"`
	EventsOCount int     `json:"Events@odata.count,omitempty"`
}

// Redfish Power

type PowerMetrics struct {
	AverageConsumedWatts float64
	IntervalInMin        float64
	MaxConsumedWatts     float64
	MinConsumedWatts     float64
}

type PowerControl struct {
	PhysicalContext    string
	Name               string
	MemberId           string
	PowerConsumedWatts float64
	PowerMetrics       PowerMetrics
}

type PowerSupply struct {
	MemberId             string
	Name                 string
	LastPowerOutputWatts float64
	LineInputVoltage     float64
	PowerInputWatts      float64
	PowerOutputWatts     float64
}

type Voltage struct {
	Name            string
	MemberId        string
	PhysicalContext string
	ReadingVolts    float64
}

type Power struct {
	PowerControl  []PowerControl
	PowerSupplies []PowerSupply
	Voltages      []Voltage
}

// Redfish Thermal

type Temperature struct {
	Name            string
	MemberId        string
	PhysicalContext string
	ReadingCelsius  float64
}

type Fan struct {
	Name            string
	PhysicalContext string
	MemberId        string
	Reading         float64
}

type EnclosureThermal struct {
	Temperatures []Temperature
	Fans         []Fan
}

// Redfish Subscriptions

type EventSubscriptionOdataId struct {
	OId string `json:"@odata.id"`
}

type EventSubscriptionCollection struct {
	OType         string                     `json:"@odata.type"`
	Name          string                     `json:"Name"`
	MembersOCount int                        `json:"Members@odata.count"`
	Members       []EventSubscriptionOdataId `json:"Members"`
}

type RFOem struct {
	EventTransmitIntervalSeconds int
}

type EventSubscription struct {
	OContext         string `json:"@odata.context,omitempty"`
	Oid              string `json:"@odata.id,omitempty"`
	Otype            string `json:"@odata.type,omitempty"`
	Id               string `json:",omitempty"`
	Name             string `json:",omitempty"`
	Context          string `json:",omitempty"`
	Destination      string
	EventTypes       []string `json:",omitempty"`
	Protocol         string
	RegistryPrefixes []string `json:",omitempty"`
	Oem              *RFOem   `json:",omitempty"`
}

func (eventSubscription EventSubscription) String() string {
	return fmt.Sprintf(
		"Context: %s, "+
			"Destination: %s, "+
			"EventTypes: %s, "+
			"RegistryPrefixes: %s, "+
			"Oem: %+v",
		eventSubscription.Context,
		eventSubscription.Destination,
		eventSubscription.EventTypes,
		eventSubscription.RegistryPrefixes,
		eventSubscription.Oem)
}

type EventService struct {
	EventTypesForSubscription []string `json:"EventTypesForSubscription"`
}

// Redfish Systems - Used to identify hardware

type Systems struct {
	// Literally the only thing we care about from all this is the model number.
	Model string
}

type Chassis struct {
	Model string
}
