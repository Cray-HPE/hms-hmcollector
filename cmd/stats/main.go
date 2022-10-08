package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/namsral/flag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

var (
	logger      *zap.Logger
	atomicLevel zap.AtomicLevel
)

func setupLogging() {
	logLevel := os.Getenv("LOG_LEVEL")
	logLevel = strings.ToUpper(logLevel)

	atomicLevel = zap.NewAtomicLevel()

	encoderCfg := zap.NewProductionEncoderConfig()
	logger = zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atomicLevel,
	))

	switch logLevel {
	case "DEBUG":
		atomicLevel.SetLevel(zap.DebugLevel)
	case "INFO":
		atomicLevel.SetLevel(zap.InfoLevel)
	case "WARN":
		atomicLevel.SetLevel(zap.WarnLevel)
	case "ERROR":
		atomicLevel.SetLevel(zap.ErrorLevel)
	case "FATAL":
		atomicLevel.SetLevel(zap.FatalLevel)
	case "PANIC":
		atomicLevel.SetLevel(zap.PanicLevel)
	default:
		atomicLevel.SetLevel(zap.InfoLevel)
	}
}

func main() {
	bootStrapServers := flag.String("bootstrap_servers", "localhost:9092", "Kafka bootstrap server")
	kakfaGroup := flag.String("kakfa_group", "hmcollector_stats", "Kafka group")

	flag.Parse()

	setupLogging()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	topics := []string{
		"cray-telemetry-temperature",
		"cray-telemetry-voltage",
		"cray-telemetry-power",
		"cray-telemetry-energy",
		"cray-telemetry-fan",
		"cray-telemetry-pressure",
		"cray-telemetry-humidity",
		"cray-telemetry-liquidflow",
		"cray-fabric-telemetry",
		"cray-fabric-perf-telemetry",
		"cray-fabric-crit-telemetry",
		"cray-dmtf-resource-event",
	}

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Connecting to kafka at %s...\n", *bootStrapServers)
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  *bootStrapServers,
		"group.id":           *kakfaGroup,
		"client.id":          fmt.Sprintf("hmcollector_stats_%s", hostname),
		"session.timeout.ms": 6000,
		"auto.offset.reset":  "latest",
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create consumer: %s\n", err)
		os.Exit(1)
	}

	if err := c.SubscribeTopics(topics, nil); err != nil {
		panic(err)
	}

	run := true
	for run {
		select {
		case sig := <-sigchan:
			fmt.Printf("Caught signal %v: terminating\n", sig)
			run = false
		default:
			ev := c.Poll(100)
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				// fmt.Printf("%% Message on %s:\n%s\n",
				// e.TopicPartition, string(e.Value))
				if e.Headers != nil {
					fmt.Printf("%% Headers: %v\n", e.Headers)
				}
				// _, err := c.StoreMessage(e)
				// if err != nil {
				// 	fmt.Fprintf(os.Stderr, "%% Error storing offset after message %s:\n",
				// 		e.TopicPartition)
				// }
			case kafka.Error:
				// Errors should generally be considered
				// informational, the client will try to
				// automatically recover.
				// But in this example we choose to terminate
				// the application if all brokers are down.
				fmt.Fprintf(os.Stderr, "%% Error: %v: %v\n", e.Code(), e)
				if e.Code() == kafka.ErrAllBrokersDown {
					run = false
				}
			default:
				fmt.Printf("Ignored %v\n", e)
			}
		}
	}

	fmt.Printf("Closing consumer\n")
	c.Close()
}
