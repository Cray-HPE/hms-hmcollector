// Copyright 2020 Hewlett Packard Enterprise Development LP

package main

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"stash.us.cray.com/HMS/hms-hmcollector/internal/hmcollector"
)

var smaOK = false
var smaOKPrev bool
var smaBroker string

func writeToKafka(topic string, payload string) {
	msg := kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          []byte(payload),
		Timestamp:      time.Now(),
	}

	// Loop on array index to avoid copy overhead of range
	for idx, _ := range kafkaBrokers {
		thisBroker := kafkaBrokers[idx]

		brokerLogger := logger.With(
			zap.String("broker", thisBroker.BrokerAddress),
			zap.String("topic", topic),
		)
		if atomicLevel.Level() == zap.DebugLevel {
			// If we're at debug level, then also include the message.
			brokerLogger = brokerLogger.With(zap.String("msg.Value", string(msg.Value)))
		}

		if _, hasTopic := thisBroker.TopicsToPublish[topic]; hasTopic {
			brokerLogger.Debug("Sent message.", zap.String("msg.Value", string(msg.Value)))

			produceErr := thisBroker.KafkaProducer.Produce(&msg, nil)
			if produceErr != nil {
				brokerLogger.Error("Failed to produce message!")
			}
		} else {
			brokerLogger.Debug("Not sending message to broker because topic not in list")
		}
	}

}

func processData(jsonPayloads <-chan jsonPayload) {
	for payload := range jsonPayloads {
		writeToKafka(payload.topic, payload.payload)
	}
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
				eventLogger.Error("Failed to produce message!")
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
		if (strings.Contains(thisBroker.BrokerAddress,"sma.svc")) {
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

	if (smaBroker == "") {
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

	for {
		cmd := fmt.Sprintf("ping -c 1 %s > /dev/null 2>&1 && echo pingtrue || echo pingfalse",smaBroker)
		outp,err := exec.Command("/bin/sh", "-c", cmd).Output()
		if (err != nil) {
			logger.Error("Error from SMA ping:",zap.Error(err))
		} else {
			logger.Debug("SMA Ping result:",zap.String("rslt",string(outp)))
			if (strings.Contains(string(outp),"pingtrue")) {
				smaOK = true
			} else {
				smaOK = false
			}

			if (first || (smaOK != smaOKPrev)) {
				if (smaOK) {
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

