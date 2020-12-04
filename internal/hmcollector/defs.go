// Copyright 2020 Hewlett Packard Enterprise Development LP

package hmcollector

import (
	"fmt"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	rf "stash.us.cray.com/HMS/hms-smd/pkg/redfish"
)

type RedfishEndpoints struct {
	Endpoints []rf.RedfishEPDescription `json:"RedfishEndpoints"`
}

type RFSub struct {
	Endpoint     *rf.RedfishEPDescription
	Status       *RFSubStatus
	PrefixGroups *[][]string
}

type RFSubStatus int

const (
	RFSUBSTATUS_ERROR RFSubStatus = iota
	RFSUBSTATUS_PENDING
	RFSUBSTATUS_COMPLETE
)

// Struct to support dealing with multiple Kafka instances.
type KafkaBroker struct {
	BrokerAddress string
	// This is going to seem like kind of a dumb way to do this, but we're going to need to lookup which topics a given
	// Kafka broker should get published A LOT. Like, every single message a lot. So we want this to be O(1) and a
	// map gives us that.
	TopicsToPublish map[string]interface{}

	KafkaConfig   *kafka.ConfigMap `json:"-"`
	KafkaProducer *kafka.Producer  `json:"-"`
}

func (broker KafkaBroker) String() string {
	var topics []string
	for topic, _ := range broker.TopicsToPublish {
		topics = append(topics, topic)
	}

	return fmt.Sprintf("Broker Address: %s, Topics: %+v", broker.BrokerAddress, topics)
}
