package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type Consumer struct {
	id               int
	logger           *zap.Logger
	bootStrapServers string
	kafkaGroup       string
	hostname         string
	topics           []string
	metrics          *Metrics
	consumerCtx      context.Context
	workQueue        chan UnparsedEventPayload
	wg               *sync.WaitGroup
}

func (c *Consumer) Start() {
	logger := c.logger

	c.wg.Add(1)

	fmt.Printf("Connecting to kafka at %s...\n", c.bootStrapServers)
	kc, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":      c.bootStrapServers,
		"group.id":               c.kafkaGroup,
		"client.id":              fmt.Sprintf("telemetry-metrics-filter_%s", c.hostname),
		"session.timeout.ms":     6000,
		"statistics.interval.ms": 1000,
		"auto.offset.reset":      "latest",
	})

	if err != nil {
		logger.Fatal("Failed to create consumer", zap.Error(err))
	}

	// Subscribe to the topics...
	if err := kc.SubscribeTopics(c.topics, nil); err != nil {
		logger.Fatal("Failed to subscribe to topics", zap.Error(err))
	}

	// Main loop to pull events out of kafka
	for {
		select {
		case <-c.consumerCtx.Done():
			logger.Info("Closing consumer")
			kc.Close()

			logger.Info("Consumer finished")
			c.wg.Done()

			return
		default:
			ev := kc.Poll(100)
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				c.metrics.InstantKafkaMessagesPerSecond.Incr(1)

				if e.TopicPartition.Topic == nil {
					logger.Warn("Received message without a topic", zap.Any("msg", e))
					continue
				}

				// TODO there should be multiple work queues, and we route messages to the same worker.
				c.workQueue <- UnparsedEventPayload{
					Topic:   *e.TopicPartition.Topic,
					Payload: e.Value,
				}

				// The kafka consumer should auto commit
				// if _, err := kc.Commit(); err != nil {
				// 	logger.Error("Failed to commit offsets", zap.Error(err))
				// }

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

				atomic.StoreInt32(&c.metrics.OverallKafkaConsumerLag, overallLag)

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
