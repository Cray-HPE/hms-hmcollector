// Copyright 2020 Hewlett Packard Enterprise Development LP

package main

import (
	"fmt"
	"github.com/namsral/flag"
	"strconv"
	"time"
)

var (
	kafkaHost = flag.String("kafka_host", "localhost",
		"The hostname of the Kafka server (likely cluster-kafka-bootstrap.sma.svc.cluster.local).")
	kafkaPort = flag.Int("kafka_port", 9092, "The port Kafka listens on.")

	bootstrapServers string
)

const kafkaTopic = "cray-dmtf-resource-event"
const iterationCount = 1000

func main() {
	// Parse the arguments.
	flag.Parse()

	bootstrapServers = *kafkaHost + ":" + strconv.Itoa(*kafkaPort)

	fmt.Printf("---CONFLUENT---\n")
	testConfluent()

	time.Sleep(30 * time.Second)

	fmt.Printf("\n\n---SARAMA---\n")
	testSarama()
}
