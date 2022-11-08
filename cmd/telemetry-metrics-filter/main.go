package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	// Parse CLI flag configuration
	brokerConfigFile := flag.String("broker_config_file", "./configs/telemetry-filter-broker-config.json", "Broker configuration file")
	workerCount := flag.Int("worker_count", 10, "Number of event workers")
	httpListenString := flag.String("http_listen", "0.0.0.0:9088", "HTTP Server listen string")

	flag.Parse()

	// Setup logging
	setupLogging()

	// Parse broker configuration
	logger.Info("Parsing Broker configuration", zap.String("brokerConfigFile", *brokerConfigFile))
	brokerConfigRaw, err := ioutil.ReadFile(*brokerConfigFile)
	if err != nil {
		logger.Fatal("Failed to read broker config file", zap.Error(err))
	}

	var brokerConfig BrokerConfig
	if err := json.Unmarshal(brokerConfigRaw, &brokerConfig); err != nil {
		logger.Fatal("Failed to unmarshal broker config file to json", zap.Error(err))
	}

	// Setup signal handler
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	//
	// Create Workers
	//
	workers := []Worker{}

	var workerWg sync.WaitGroup
	workerCtx, workerCancel := context.WithCancel(context.Background())
	for id := 0; id < *workerCount; id++ {
		workerWg.Add(1)

		worker := Worker{
			id:           id,
			logger:       logger.With(zap.Int("WorkerID", id)),
			brokerConfig: brokerConfig,
			workQueue:    make(chan UnparsedEventPayload),
			ctx:          workerCtx,
			wg:           &workerWg,
		}
		workers = append(workers, worker)

		go worker.Start()
	}

	// Retrieve the hostname of the pod
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	// Start consumers
	var consumerWg sync.WaitGroup
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	consumer := Consumer{
		id:           0,
		logger:       logger.With(zap.Int("ConsumerID", 0)),
		hostname:     hostname,
		brokerConfig: brokerConfig,
		consumerCtx:  consumerCtx,
		workers:      workers,
		wg:           &consumerWg,
	}

	go consumer.Start()

	// Start REST API
	api := API{
		logger:       logger.With(zap.Int("ApiID", 0)), // I don't like this name
		consumer:     &consumer,
		listenString: *httpListenString,
	}
	go api.Start()

	// Metrics output loop
	go func() {
		ticker := time.NewTicker(5 * time.Second)

		for {
			select {
			case <-workerCtx.Done():
				logger.Info("Metrics loop is done")
				return
			case <-ticker.C:
				logger.Info("Metrics",
					zap.Int64("InstantKafkaMessagesPerSecond", consumer.metrics.InstantKafkaMessagesPerSecond.Rate()),
					zap.Int32("OverallKafkaConsumerLag", consumer.metrics.OverallKafkaConsumerLag),
				)
			}
		}
	}()

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
