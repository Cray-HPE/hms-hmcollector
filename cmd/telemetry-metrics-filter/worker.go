package main

import (
	"context"
	"sync"

	"github.com/Cray-HPE/hms-hmcollector/internal/hmcollector"
	"go.uber.org/zap"
)

type UnparsedEventPayload struct {
	Topic   string
	Payload []byte
}

type Worker struct {
	id     int
	logger *zap.Logger

	brokerConfig BrokerConfig

	workQueue chan UnparsedEventPayload
	ctx       context.Context

	wg *sync.WaitGroup
}

func (w *Worker) Start() {
	defer w.wg.Done()
	logger := w.logger
	logger.Info("Starting worker")

	for {
		select {
		case workUnit := <-w.workQueue:
			logger.Debug("Received work unit", zap.Any("workUnit", workUnit))
			// Process the event

			// Unmarshal the event json
			events, err := hmcollector.UnmarshalEvents(logger, workUnit.Payload)
			if err != nil {
				logger.Error("Failed to unmarshal events", zap.String("topic", workUnit.Topic), zap.ByteString("payload", workUnit.Payload))
			}

			// TODO process the event here
			// TODO throttle or send messages
			_ = events

		case <-w.ctx.Done():
			logger.Info("Worker finished")
			return
		}
	}
}
