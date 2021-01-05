// Copyright 2020 Hewlett Packard Enterprise Development LP

package main

import (
	"fmt"
	"github.com/Shopify/sarama"
	"strconv"
	"time"
)

var kafkaProducer *sarama.AsyncProducer

func handleSaramaKafkaErrors() {
	for kafkaError := range (*kafkaProducer).Errors() {
		fmt.Printf("Failed to produce message async (%s): %+v\n", kafkaError.Err, kafkaError.Msg)
	}
	fmt.Println("Done consuming errors.")
}

func testSarama() {
	fmt.Printf("Connecting to %s and gathering topics with Sarama library...\n", bootstrapServers)

	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Producer.Return.Errors = true
	config.Producer.Return.Successes = true
	config.Consumer.Fetch.Max = 2147483647

	client, err := sarama.NewClient([]string{bootstrapServers}, config)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Get all topics/brokers from client
	topics, _ := client.Topics()

	fmt.Printf("Found %d topics:\n", len(topics))

	for _, topic := range topics {
		fmt.Printf("\t%s\n", topic)
	}

	var producer sarama.AsyncProducer

	producer, err = sarama.NewAsyncProducerFromClient(client)
	if err != nil {
		fmt.Println("Unable to connect to Kafka!")
		return
	}

	kafkaProducer = &producer

	go handleSaramaKafkaErrors()

	for i := 1; i <= iterationCount; i++ {
		payload := `
{
    "Context": "Sarama",
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
		msg := &sarama.ProducerMessage{
			Topic: kafkaTopic,
			Value: sarama.StringEncoder(payload),
		}

		producer.Input() <- msg

		fmt.Printf("Sent message async: %+v\n", msg)
	}

	time.Sleep(time.Second * 3)

	producer.AsyncClose()
}
