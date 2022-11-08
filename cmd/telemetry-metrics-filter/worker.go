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
	metrics      *WorkerMetrics
}

func (w *Worker) Start() {
	defer w.wg.Done()
	logger := w.logger
	logger.Info("Starting worker")

	w.metrics = &WorkerMetrics{}

	// Setup topic filter
	topicFilters := map[string]*TopicFilter{}
	for topic, filterConfig := range w.brokerConfig.TopicsToFilter {
		topicFilters[topic] = NewTopicFilter(logger, time.Second*time.Duration(filterConfig.ThrottlePeriodSeconds))
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

				// TODO determine filtered topic name
				// TODO send event to kafka
			}

		case <-w.ctx.Done():
			logger.Info("Worker finished")
			return
		}
	}
}
