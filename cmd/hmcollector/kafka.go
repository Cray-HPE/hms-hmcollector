// MIT License
//
// (C) Copyright [2020-2021,2024-2025] Hewlett Packard Enterprise Development LP
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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"github.com/Cray-HPE/hms-hmcollector/internal/hmcollector"
)

var smaOK = false
var smaOKPrev bool
var smaBroker string

func writeToKafka(topic, payload string, messageID *string) {
	msg := kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          []byte(payload),
		Timestamp:      time.Now(),
	}

	mId := ""
	if messageID != nil {
		msg.Key = []byte(*messageID)
		mId = *messageID
	}

	// Loop on array index to avoid copy overhead of range
	for idx, _ := range kafkaBrokers {
		thisBroker := kafkaBrokers[idx]

		brokerLogger := logger.With(
			zap.String("broker", thisBroker.BrokerAddress),
			zap.String("topic", topic),
			zap.String("id", mId),
		)
		if atomicLevel.Level() == zap.DebugLevel {
			// If we're at debug level, then also include the message.
			brokerLogger = brokerLogger.With(zap.String("msg.Value", string(msg.Value)))
		}

		if _, hasTopic := thisBroker.TopicsToPublish[topic]; hasTopic {
			brokerLogger.Debug("Sent message.", zap.String("msg.Value", string(msg.Value)))

			if shouldLogMessage(mId) {
				brokerLogger.Info("message", zap.String("msg.Value", string(msg.Value)))
			}
			produceErr := thisBroker.KafkaProducer.Produce(&msg, nil)
			if produceErr != nil {
				if shouldLogErrors {
					brokerLogger.Error("Failed to produce message!", zap.Error(produceErr))
				} else {
					brokerLogger.Error("Failed to produce message!")
				}
			}
		} else {
			brokerLogger.Debug("Not sending message to broker because topic not in list")
		}
	}

}

func shouldLogMessage(id string) bool {
	if shouldLogForXnames {
		ids := strings.Split(id, ".")
		xname := ids[0]
		_, matches := logXnames[xname]
		return matches
	}
	return false
}

func handleKafkaEvents(broker *hmcollector.KafkaBroker) {
	for event := range broker.KafkaProducer.Events() {
		switch ev := event.(type) {
		case *kafka.Message:
			eventLogger := logger.With(
				zap.String("broker", broker.BrokerAddress),
				zap.Any("Message.TopicPartition", ev.TopicPartition),
			)

			if ev.TopicPartition.Error != nil {
				if shouldLogErrors {
					eventLogger.Error("Failed to produce message!", zap.Error(ev.TopicPartition.Error))
				} else {
					eventLogger.Error("Failed to produce message!")
				}
			} else {
				eventLogger.Debug("Produced message.")
			}
		}
	}
}

func setupKafka() {
	// Now setup a connection to each of the provided Kafka brokers.
	jsonFile, err := os.Open(*kafkaBrokersConfigFile)
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()

	jsonBytes, _ := ioutil.ReadAll(jsonFile)
	err = json.Unmarshal(jsonBytes, &kafkaBrokers)
	if err != nil {
		panic(err)
	}

	// If there are no brokers defined panic.
	if len(kafkaBrokers) == 0 {
		logger.Panic("No Kafka brokers defined!",
			zap.String("kafkaBrokersConfigFile", *kafkaBrokersConfigFile))
	}

	for idx, _ := range kafkaBrokers {
		thisBroker := kafkaBrokers[idx]
		if strings.Contains(thisBroker.BrokerAddress, "sma.svc") {
			smaBroker = thisBroker.BrokerAddress
		}

		logger.Info("Connecting to Kafka broker...", zap.String("BrokerAddress", thisBroker.BrokerAddress))

		// Setup the config for this broker.
		thisBroker.KafkaConfig = &kafka.ConfigMap{
			"bootstrap.servers": thisBroker.BrokerAddress,
		}

		connected := false

		// We will retry forever to connect to Kafka...unless we're killed.
		for !connected {
			thisBroker.KafkaProducer, err = kafka.NewProducer(thisBroker.KafkaConfig)
			if err != nil {
				if !Running {
					os.Exit(0)
				}

				logger.Warn("Unable to connect to Kafka broker! Trying again in 1 second...",
					zap.String("BrokerAddress", thisBroker.BrokerAddress))
				time.Sleep(1 * time.Second)
			} else {
				connected = true
			}
		}
		logger.Info("Connected to Kafka broker.", zap.Any("broker", thisBroker))

		// Handle any events (good or bad, unfortunately can't just pick bad if we want) for all messages.
		go handleKafkaEvents(thisBroker)
	}

	time.Sleep(1 * time.Second)

	if smaBroker == "" {
		logger.Error("No brokers to ping...")
	} else {
		logger.Info("Monitoring SMA connectivity...")
		go monitorSMA()
	}
}

//Monitor the connectivity state of SMA Kafka.  If it goes away,  we
//can stop polling River telemetry.

func monitorSMA() {
	first := true

	// Extract the hostname of the SMA kafka broker
	smaHost := smaBroker
	if strings.Contains(smaBroker, ":") {
		// hostname:port was specified, need to retrieve just the hostname
		var err error
		smaHost, _, err = net.SplitHostPort(smaBroker)
		if err != nil {
			logger.Fatal("Unable to parse SMA broker host", zap.String("smaBroker", smaBroker), zap.Error(err))
		}
	}

	for {
		cmd := fmt.Sprintf("ping -c 1 %s > /dev/null && echo pingtrue || echo pingfalse", smaHost)

		result := exec.Command("/bin/sh", "-c", cmd)
		outp, err := result.CombinedOutput()
		if err != nil {
			logger.Error("Error from SMA ping:", zap.Error(err))
		} else {
			logger.Debug("SMA Ping result:", zap.String("rslt", string(outp)), zap.String("cmd", cmd))
			if strings.Contains(string(outp), "pingtrue") {
				smaOK = true
			} else {
				smaOK = false
			}

			if first || (smaOK != smaOKPrev) {
				if smaOK {
					logger.Warn("SMA check: Connectivity OK.  Telemetry polling is running.")
				} else {
					logger.Warn("SMA check: Connectivity is inactive.  Telemetry polling is paused.")
				}
			}
		}
		first = false
		smaOKPrev = smaOK

		time.Sleep(30 * time.Second)
	}
}
