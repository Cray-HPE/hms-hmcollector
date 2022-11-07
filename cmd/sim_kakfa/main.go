package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/namsral/flag"
	"github.com/paulbellamy/ratecounter"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

func main() {
	bootStrapServers := flag.String("bootstrap_servers", "localhost:9092", "Kafka bootstrap server")

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": *bootStrapServers,
		"client.id":         fmt.Sprintf("sim_kafka_%s", hostname),
		"acks":              "all"})

	if err != nil {
		fmt.Printf("Failed to create producer: %s\n", err)
		os.Exit(1)
	}

	rate := ratecounter.NewRateCounter(1 * time.Second)

	// Metrics output loop
	go func() {
		ticker := time.NewTicker(1000 * time.Millisecond)

		for {
			select {
			case <-sigchan:
				fmt.Println("Metrics loop is done")
				return
			case <-ticker.C:
				fmt.Println(rate.Rate(), "events/second")
			}
		}
	}()

	ticker := time.NewTicker(time.Millisecond)
	run := true
	for run {
		select {
		case sig := <-sigchan:
			fmt.Printf("Caught signal %v: terminating\n", sig)
			run = false
		case <-ticker.C:
			rate.Incr(1)
			// default:
			// fmt.Println("foo")

			// fmt.Println("Sending message")

			payload := `{"Context":"x1001c1b0","Events":[
			{"EventTimestamp":"2022-10-07T23:59:01Z","MessageId":"CrayTelemetry.Current","Oem":{"Sensors":[
			{"Timestamp":"2022-10-07T23:59:01.144707955Z","Location":"x1001c1","PhysicalContext":"Rectifier","Index":1,"PhysicalSubContext":"Input","DeviceSpecificContext":"Line3","Value":"12.07"},
			{"Timestamp":"2022-10-07T23:59:01.196647484Z","Location":"x1001c1","PhysicalContext":"Rectifier","Index":1,"PhysicalSubContext":"Input","DeviceSpecificContext":"Line2","Value":"11.99"},
			{"Timestamp":"2022-10-07T23:59:01.248623130Z","Location":"x1001c1","PhysicalContext":"Rectifier","Index":1,"PhysicalSubContext":"Input","DeviceSpecificContext":"Line1","Value":"12.05"},
			{"Timestamp":"2022-10-07T23:59:01.296768659Z","Location":"x1001c1","PhysicalContext":"Rectifier","Index":0,"PhysicalSubContext":"Input","DeviceSpecificContext":"Line3","Value":"12.12"},
			{"Timestamp":"2022-10-07T23:59:01.348856260Z","Location":"x1001c1","PhysicalContext":"Rectifier","Index":0,"PhysicalSubContext":"Input","DeviceSpecificContext":"Line2","Value":"12.05"},
			{"Timestamp":"2022-10-07T23:59:01.400991300Z","Location":"x1001c1","PhysicalContext":"Rectifier","Index":0,"PhysicalSubContext":"Input","DeviceSpecificContext":"Line1","Value":"12.22"}],"TelemetrySource":"cC"}}]}`

			topic := "cray-telemetry-power"

			msg := kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
				Value:          []byte(payload),
				Timestamp:      time.Now(),
			}
			if err := p.Produce(&msg, nil); err != nil {
				fmt.Println("Error failed to produce message! Error:", err)
			}

		}
		// time.Sleep(time.Second)
	}
}
