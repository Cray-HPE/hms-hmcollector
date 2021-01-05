module stash.us.cray.com/HMS/hms-hmcollector

go 1.12

require (
	github.com/Shopify/sarama v1.23.1
	github.com/eapache/go-resiliency v1.2.0 // indirect
	github.com/hashicorp/go-retryablehttp v0.6.7
	github.com/namsral/flag v1.7.4-pre
	github.com/rcrowley/go-metrics v0.0.0-20190706150252-9beb055b7962 // indirect
	go.uber.org/zap v1.15.0
	gopkg.in/confluentinc/confluent-kafka-go.v1 v1.1.0
	gopkg.in/jcmturner/goidentity.v3 v3.0.0 // indirect
	stash.us.cray.com/HMS/hms-certs v1.0.6
	stash.us.cray.com/HMS/hms-compcredentials v1.7.0
	stash.us.cray.com/HMS/hms-go-http-lib v1.2.2
	stash.us.cray.com/HMS/hms-securestorage v1.8.0
	stash.us.cray.com/HMS/hms-smd v1.25.1
)
