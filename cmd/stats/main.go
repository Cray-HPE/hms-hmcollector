package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
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
	workerCount := flag.Int("worker_count", 10, "Number of event workers")

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
	go runConsumer(0, *bootStrapServers, *kakfaGroup, hostname, topics, metrics, consumerCtx, workQueue, &consumerWg)
	// go runConsumer(1, *bootStrapServers, *kakfaGroup, hostname, topics, metrics, consumerCtx, workQueue, &consumerWg)
	// go runConsumer(2, *bootStrapServers, *kakfaGroup, hostname, topics, metrics, consumerCtx, workQueue, &consumerWg)
	// go runConsumer(3, *bootStrapServers, *kakfaGroup, hostname, topics, metrics, consumerCtx, workQueue, &consumerWg)

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

func runConsumer(id int, bootStrapServers, kakfaGroup, hostname string, topics []string, metrics *Metrics, workerCtx context.Context, workQueue chan UnparsedEventPayload, wg *sync.WaitGroup) {
	wg.Add(1)

	logger := logger.With(zap.Int("ConsumerID", id))

	fmt.Printf("Connecting to kafka at %s...\n", bootStrapServers)
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":      bootStrapServers,
		"group.id":               kakfaGroup,
		"client.id":              fmt.Sprintf("hmcollector_stats_%s", hostname),
		"session.timeout.ms":     6000,
		"statistics.interval.ms": 1000,
		"auto.offset.reset":      "latest",
	})

	if err != nil {
		logger.Fatal("Failed to create consumer", zap.Error(err))
	}

	// Subscribe to the topics...
	if err := c.SubscribeTopics(topics, nil); err != nil {
		logger.Fatal("Failed to subscribe to topics", zap.Error(err))
	}

	// Main loop to pull events out of kafka
	for {
		select {
		case <-workerCtx.Done():
			logger.Info("Closing consumer")
			c.Close()

			wg.Done()

			logger.Info("Consumer finished")

			return
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

				if _, err := c.Commit(); err != nil {
					logger.Error("Failed to commit offsets", zap.Error(err))
				}

			case kafka.Error:
				// Errors should generally be considered
				// informational, the client will try to
				// automatically recover.
				// But in this example we choose to terminate
				// the application if all brokers are down.
				logger.Error("consumer error", zap.String("error", e.String()))

				// fmt.Fprintf(os.Stderr, "%% Error: %v: %v\n", e.Code(), e)
				// if e.Code() == kafka.ErrAllBrokersDown {
				// 	run = false
				// }
			case kafka.OffsetsCommitted:
				logger.Debug("Offsets committed", zap.Any("msg", e))
			case *kafka.Stats:
				// Stats events are emitted as JSON (as string).
				// Either directly forward the JSON to your
				// statistics collector, or convert it to a
				// map to extract fields of interest.
				// The definition of the statistics JSON
				// object can be found here:
				// https://github.com/edenhill/librdkafka/blob/master/STATISTICS.md

				type kafkaPartitionStats struct {
					ConsumerLag int `json:"consumer_lag"`
				}

				type KafkaTopicStats struct {
					Partitions map[string]kafkaPartitionStats `json:"partitions"`
				}

				type kafkaStats struct {
					Topics map[string]KafkaTopicStats `json:"topics"`
				}

				var stats kafkaStats
				json.Unmarshal([]byte(e.String()), &stats)

				var overallLag int32
				for _, topic := range stats.Topics {
					for _, parition := range topic.Partitions {
						overallLag += int32(parition.ConsumerLag)
					}
				}

				atomic.StoreInt32(&metrics.OverallKafkaConsumerLag, overallLag)

				// o, _ := json.Marshal(stats)
				// fmt.Println(string(o))
				// // fmt.Printf("Stats: %v messages (%v bytes) messages consumed\n",
				// // 	stats["rxmsgs"], stats["rxmsg_bytes"])
				// consumerLag, err := strconv.Atoi(stats["consumer_lag"].(string))
				// if err != nil {
				// 	logger.Error("Failed to convert consumer_lag to int from string")
				// }

				// atomic.StoreInt32(&metrics.KafkaConsumerLag, int32(consumerLag))

			default:
				fmt.Printf("Ignored %v\n", e)
			}
		}
	}
}
