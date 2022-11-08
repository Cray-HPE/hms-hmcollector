package main

import (
	"time"

	"github.com/paulbellamy/ratecounter"
)

type Metrics struct {
	OverallKafkaConsumerLag       int32
	InstantKafkaMessagesPerSecond *ratecounter.RateCounter
}

func NewMetrics() *Metrics {
	return &Metrics{
		InstantKafkaMessagesPerSecond: ratecounter.NewRateCounter(1 * time.Second),
	}
}
