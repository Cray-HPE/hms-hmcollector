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
	kafkaGroup := flag.String("kafka_group", "telemetry-metrics-filter", "Kafka group")
	workerCount := flag.Int("worker_count", 10, "Number of event workers")
	httpListenString := flag.String("http_listen", "0.0.0.0:9088", "HTTP Server listen string")

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

	// Retrieve the hostname of the pod
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	// Topics to listen on
	// TODO read in from filter config
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

	// Metrics output loop
	go func() {
		ticker := time.NewTicker(1000 * time.Millisecond)

		for {
			select {
			case <-workerCtx.Done():
				logger.Info("Metrics loop is done")
				return
			case <-ticker.C:
				logger.Info("Metrics",
					zap.Int64("InstantKafkaMessagesPerSecond", metrics.InstantKafkaMessagesPerSecond.Rate()),
					zap.Int32("OverallKafkaConsumerLag", metrics.OverallKafkaConsumerLag),
				)
			}
		}
	}()

	// Start consumers
	var consumerWg sync.WaitGroup
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	consumer := Consumer{
		id:               0,
		logger:           logger.With(zap.Int("ConsumerID", 0)),
		bootStrapServers: *bootStrapServers,
		kafkaGroup:       *kafkaGroup,
		hostname:         hostname,
		topics:           topics,
		metrics:          metrics,
		consumerCtx:      consumerCtx,
		workQueue:        workQueue,
		wg:               &consumerWg,
	}

	go consumer.Start()

	// Start API
	api := API{
		logger:       logger.With(zap.Int("ApiID", 0)), // I don't like this name
		listenString: *httpListenString,
	}
	go api.Start()

	sig := <-sigchan
	fmt.Printf("Caught signal %v: terminating\n", sig)

	// Stop the consumers
	logger.Info("Stopping consumers")
	consumerCancel()
	consumerWg.Wait()
	logger.Info("All consumers completed")

	// No more work, stop the workers
	logger.Info("Stopping workers")
	workerCancel()
	workerWg.Wait()
	logger.Info("All workers completed")

}
