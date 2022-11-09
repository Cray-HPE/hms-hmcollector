package main

import (
	"context"
	"sync"
	"time"

	"github.com/Cray-HPE/hms-hmcollector/internal/hmcollector"
	"go.uber.org/zap"
)

type UnparsedEventPayload struct {
	MessageKey []byte
	Topic      string
	PayloadRaw []byte
}

type WorkerMetrics struct {
}

type Worker struct {
	id        int
	logger    *zap.Logger
	workQueue chan UnparsedEventPayload
	ctx       context.Context
	wg        *sync.WaitGroup

	brokerConfig BrokerConfig

	metrics  *WorkerMetrics
	producer *Producer
}

func (w *Worker) Start() {
	w.wg.Add(1)
	defer w.wg.Done()

	logger := w.logger
	logger.Info("Starting worker")

	w.metrics = &WorkerMetrics{}

	// Setup topic filter
	topicFilters := map[string]*TopicFilter{}
	destinationTopics := map[string]string{}
	for topic, filterConfig := range w.brokerConfig.TopicsToFilter {
		topicFilters[topic] = NewTopicFilter(logger, time.Second*time.Duration(filterConfig.ThrottlePeriodSeconds))
		destinationTopic := topic + w.brokerConfig.FilteredTopicSuffix
		if filterConfig.DestinationTopicName != nil {
			destinationTopic = *filterConfig.DestinationTopicName
		}

		destinationTopics[topic] = destinationTopic
	}

	// TODO add some metrics to see when the last time a BMC sent a telemetry type

	// Process events!
	for {
		select {
		case workUnit := <-w.workQueue:
			logger.Debug("Received work unit", zap.ByteString("messageKey", workUnit.MessageKey), zap.String("topic", workUnit.Topic))
			// Process the event

			// Unmarshal the event json
			events, err := hmcollector.UnmarshalEvents(logger, workUnit.PayloadRaw)
			if err != nil {
				logger.Error("Failed to unmarshal events", zap.String("topic", workUnit.Topic), zap.ByteString("payload", workUnit.PayloadRaw))
				continue
			}

			// Process the event here
			topicFilter, ok := topicFilters[workUnit.Topic]
			if !ok {
				// Somehow we got a event for a topic that wasn't subscripted for. This should never happen
				logger.Error("No topic filter for topic", zap.String("topic", workUnit.Topic))
				continue
			}

			// Decided to throttle or send an event
			if topicFilter.ShouldThrottle(events) {
				logger.Debug("Throttling message", zap.ByteString("messageKey", workUnit.MessageKey), zap.String("topic", workUnit.Topic))
			} else {
				logger.Debug("Sending message", zap.ByteString("messageKey", workUnit.MessageKey), zap.String("topic", workUnit.Topic))

				// Determine filtered topic name
				destinationTopic, ok := destinationTopics[workUnit.Topic]
				if !ok {
					// Somehow we got a event for a topic that wasn't subscripted for. This should never happen
					logger.Error("No destination filter for topic", zap.String("topic", workUnit.Topic))
					continue
				}

				// Send event to kafka
				err = w.producer.Produce(destinationTopic, workUnit.PayloadRaw)
				if err != nil {
					// TODO log producer errors?? How?
					logger.Error("Failed to produce message", zap.Error(err))
				}
			}

		case <-w.ctx.Done():
			logger.Info("Worker finished")
			return
		}
	}
}
