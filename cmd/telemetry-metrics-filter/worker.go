package main

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

type UnparsedEventPayload struct {
	Topic   string
	Payload []byte
}

type Worker struct {
	id     int
	logger *zap.Logger

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

			events, err := unmarshalEvents(workUnit.Payload)
			if err != nil {
				logger.Error("Failed to unmarshal events", zap.String("topic", workUnit.Topic), zap.ByteString("payload", workUnit.Payload))
			}

			_ = events
		case <-w.ctx.Done():
			logger.Info("Worker finished")
			return
		}
	}
}
