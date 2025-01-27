// MIT License
//
// (C) Copyright [2020-2021,2025] Hewlett Packard Enterprise Development LP
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

package hmcollector

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	rf "github.com/Cray-HPE/hms-smd/v2/pkg/redfish"
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
