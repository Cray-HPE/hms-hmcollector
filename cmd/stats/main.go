package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

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
	workerCount := flag.Int("worker_count", 2, "Number of event workers")

	flag.Parse()

	setupLogging()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	//
	// Create Workers
	//
	workQueue := make(chan UnparsedEventPayload)

	var workerWg sync.WaitGroup
	workerCtx, workerCancel := context.WithCancel(context.Background())
	for id := 0; id < *workerCount; id++ {
		workerWg.Add(1)

		worker := Worker{
			id:        id,
			logger:    logger.With(zap.Int("WorkerID", id)),
			workQueue: workQueue,
			ctx:       workerCtx,
			wg:        &workerWg,
		}

		go worker.Start()
	}

	// Setup Metrics
	metrics := NewMetrics()

	// Setup the kafka consumer
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

	// Topics to listen on
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

	// Subscribe to the topics...
	if err := c.SubscribeTopics(topics, nil); err != nil {
		panic(err)
	}

	// Offset commit loop
	go func() {
		ticker := time.NewTicker(1000 * time.Millisecond)

		for {
			select {
			case <-workerCtx.Done():
				logger.Info("Offset commit loop is done")
				return
			case <-ticker.C:
				// logger.Info("Committing offsets")
				if _, err := c.Commit(); err != nil {
					logger.Error("Failed to commit offsets to kafka", zap.Error(err))
				}
			}
		}
	}()

	// Metrics output loop
	go func() {
		ticker := time.NewTicker(1000 * time.Millisecond)

		for {
			select {
			case <-workerCtx.Done():
				logger.Info("Metrics loop is done")
				return
			case <-ticker.C:
				logger.Info("Metrics", zap.Int64("InstantKafkaMessagesPerSecond", metrics.InstantKafkaMessagesPerSecond.Rate()))
			}
		}
	}()

	// Main loop to pull events out of kafka
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
				metrics.InstantKafkaMessagesPerSecond.Incr(1)

				if e.TopicPartition.Topic == nil {
					logger.Warn("Received message without a topic", zap.Any("msg", e))
					continue
				}

				workQueue <- UnparsedEventPayload{
					Topic:   *e.TopicPartition.Topic,
					Payload: e.Value,
				}

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
			case kafka.OffsetsCommitted:
				logger.Debug("Offsets committed", zap.Any("msg", e))
			default:
				fmt.Printf("Ignored %v\n", e)
			}
		}
	}

	fmt.Printf("Closing consumer\n")
	c.Close()

	// No more work, stop the workers
	workerCancel()
	workerWg.Wait()
	logger.Info("All workers completed")

}
