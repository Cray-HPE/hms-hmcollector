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
