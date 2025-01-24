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

package main

import (
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"strconv"
)

func testConfluent() {
	fmt.Printf("Connecting to %s and gathering topics with Confluent library...\n", bootstrapServers)

	// 100000000
	// 1213486160
	// 1000000000

	configMap := kafka.ConfigMap{
		"bootstrap.servers":         *kafkaHost + ":" + strconv.Itoa(*kafkaPort),
		"group.id":                  "test",
	}

	fmt.Printf("Configuration: %+v\n", configMap)

	adminClient, err := kafka.NewAdminClient(&configMap)

	if err != nil {
		fmt.Println(err)
		return
	}

	metadata, err := adminClient.GetMetadata(nil, true, 5000);
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Found %d topics:\n", len(metadata.Topics))

	for _, topicMetadata := range metadata.Topics {
		fmt.Printf("\t%s - %d partition(s).\n", topicMetadata.Topic, len(topicMetadata.Partitions))
	}

	adminClient.Close()

	p, err := kafka.NewProducer(&configMap)
	if err != nil {
		panic(err)
	}

	defer p.Close()

	// Delivery report handler for produced messages
	go func() {
		for e := range p.Events() {
			fmt.Println(e)
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					fmt.Printf("Delivery failed: %v\n", ev.TopicPartition)
				} else {
					fmt.Printf("Delivered message to %v\n", ev.TopicPartition)
				}
			}
		}
	}()

	for i := 1; i <= iterationCount; i++ {
		payload := `
{
    "Context": "Confluent",
    "Events": [
        {
            "EventId": "` + strconv.Itoa(i) + `",
            "Severity": "OK",
            "Message": "The power state of resource /redfish/v1/Systems/Node0 has changed to type On.",
            "MessageId": "CrayAlerts.1.0.ResourcePowerStateChanged",
            "MessageArgs": [
                "/redfish/v1/Systems/Node0",
                "On"
            ],
            "OriginOfCondition": {
                "@odata.id": "/redfish/v1/Systems/Node0"
            }
        }
    ]
}
`
		// Produce messages to topic (asynchronously)
		topic := kafkaTopic
		p.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          []byte(payload),
		}, nil)
	}

	// Wait for message deliveries before shutting down
	p.Flush(60 * 1000)
}
