package main

type BrokerConfig struct {
	BrokerAddress  string                 `json:"BrokerAddress"`
	ConsumerGroup  string                 `json:"ConsumerGroup"`
	TopicsToFilter map[string]TopicFilter `json:"TopicsToFilter"`
}

type TopicFilter struct {
	ThrottlePeriodSeconds int     `json:"ThrottlePeriodSeconds"`
	DestinationTopicName  *string `json:"DestinationTopicName"`
}
