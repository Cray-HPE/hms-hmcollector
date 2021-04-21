module stash.us.cray.com/HMS/hms-hmcollector

go 1.12

replace (
	stash.us.cray.com/HMS/hms-certs => stash.us.cray.com/HMS/hms-certs v1.2.2
	stash.us.cray.com/HMS/hms-compcredentials => stash.us.cray.com/HMS/hms-compcredentials v1.10.0
	stash.us.cray.com/HMS/hms-go-http-lib => stash.us.cray.com/HMS/hms-go-http-lib v1.4.0
	stash.us.cray.com/HMS/hms-securestorage => stash.us.cray.com/HMS/hms-securestorage v1.11.0
	stash.us.cray.com/HMS/hms-smd => stash.us.cray.com/HMS/hms-smd v1.28.7
)

require (
	github.com/Shopify/sarama v1.23.1
	github.com/eapache/go-resiliency v1.2.0 // indirect
	github.com/golang/snappy v0.0.1
	github.com/hashicorp/errwrap v1.0.0
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/go-retryablehttp v0.6.7
	github.com/hashicorp/go-sockaddr v1.0.2
	github.com/hashicorp/go-uuid v1.0.1
	github.com/hashicorp/hcl v1.0.0
	github.com/hashicorp/vault/sdk v0.1.13
	github.com/jcmturner/gofork v0.0.0-20190328161633-dc7c13fece03
	github.com/mitchellh/mapstructure v1.3.0
	github.com/namsral/flag v1.7.4-pre
	github.com/pierrec/lz4 v2.4.1+incompatible
	github.com/rcrowley/go-metrics v0.0.0-20190706150252-9beb055b7962 // indirect
	github.com/ryanuber/go-glob v1.0.0
	github.com/sirupsen/logrus v1.6.0
	go.uber.org/zap v1.15.0
	golang.org/x/crypto v0.0.0-20200709230013-948cd5f35899
	golang.org/x/net v0.0.0-20200813134508-3edf25e44fcc
	golang.org/x/sys v0.0.0-20200817155316-9781c653f443
	golang.org/x/text v0.3.3
	gopkg.in/confluentinc/confluent-kafka-go.v1 v1.1.0
	gopkg.in/jcmturner/aescts.v1 v1.0.1
	gopkg.in/jcmturner/dnsutils.v1 v1.0.1
	gopkg.in/jcmturner/goidentity.v3 v3.0.0 // indirect
	gopkg.in/jcmturner/gokrb5.v7 v7.2.3
	gopkg.in/jcmturner/rpc.v1 v1.1.0
	gopkg.in/square/go-jose.v2 v2.3.1
	stash.us.cray.com/HMS/hms-base v1.12.0
	stash.us.cray.com/HMS/hms-certs v1.2.0
	stash.us.cray.com/HMS/hms-compcredentials v1.10.0
	stash.us.cray.com/HMS/hms-go-http-lib v1.4.0
	stash.us.cray.com/HMS/hms-securestorage v1.11.0
	stash.us.cray.com/HMS/hms-smd v1.25.1
)
