module stash.us.cray.com/HMS/hms-hmcollector

go 1.16

require (
	github.com/Shopify/sarama v1.23.1
	github.com/eapache/go-resiliency v1.2.0 // indirect
	github.com/namsral/flag v1.7.4-pre
	github.com/rcrowley/go-metrics v0.0.0-20190706150252-9beb055b7962 // indirect
	go.uber.org/zap v1.15.0
	gopkg.in/confluentinc/confluent-kafka-go.v1 v1.1.0
	gopkg.in/jcmturner/goidentity.v3 v3.0.0 // indirect
	stash.us.cray.com/HMS/hms-base v1.13.0
	stash.us.cray.com/HMS/hms-certs v1.3.0
	stash.us.cray.com/HMS/hms-compcredentials v1.11.0
	stash.us.cray.com/HMS/hms-securestorage v1.12.0
	stash.us.cray.com/HMS/hms-smd v1.30.4
)
