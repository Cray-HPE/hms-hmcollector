// MIT License
//
// (C) Copyright [2020-2021,2024-2025] Hewlett Packard Enterprise Development LP
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
// THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
// OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
// ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/namsral/flag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	base "github.com/Cray-HPE/hms-base/v2"
	"github.com/Cray-HPE/hms-certs/pkg/hms_certs"
	"github.com/Cray-HPE/hms-hmcollector/internal/hmcollector"
	"github.com/Cray-HPE/hms-hmcollector/internal/river_collector"
	rf "github.com/Cray-HPE/hms-smd/v2/pkg/redfish"
)

const NumWorkers = 30
const EndpointRefreshInterval = 5

var telemetryTypes = [2]river_collector.TelemetryType{
	river_collector.TelemetryTypePower,
	river_collector.TelemetryTypeThermal,
}

var (
	//namsral flag parsing, also parses from env vars that are upper case
	pollingEnabled     = flag.Bool("polling_enabled", false, "Should polling be enabled?")
	rfSubscribeEnabled = flag.Bool("rf_subscribe_enabled", false,
		"Should redfish subscribing be enabled?")
	rfStreamingEnabled = flag.Bool("rf_streaming_enabled", true,
		"Should streaming telemetry subscriptions be created?")
	restEnabled = flag.Bool("rest_enabled", true, "Should a RESTful server be started?")

	VaultEnabled = flag.Bool("vault_enabled", true, "Should vault be used for credentials?")
	VaultAddr    = flag.String("vault_addr", "http://localhost:8200", "Address of Vault.")
	VaultKeypath = flag.String("vault_keypath", "secret/hms-creds",
		"Keypath for Vault credentials.")

	kafkaBrokersConfigFile = flag.String("kafka_brokers_config", "configs/kafka_brokers.json",
		"Path to the configuration file containing all of the Kafka brokers this collector should produce to.")

	pruneOldSubscriptions = flag.Bool("prune_old_subscriptions", true, "Should it prune old subscriptions that contain the wrong xname when compared to the endpoint.")

	pollingInterval    = flag.Int("polling_interval", 10, "The polling interval to use in seconds.")
	pduPollingInterval = flag.Int("pdu_polling_interval", 30, "The polling interval to use for redfish PDUs in seconds.")
	hsmRefreshInterval = flag.Int("hsm_refresh_interval", 30,
		"The interval to check HSM for new Redfish Endpoints in seconds.")

	smURL            = flag.String("sm_url", "", "Address of the State Manager.")
	restURL          = flag.String("rest_url", "", "Address for Redfish events to target.")
	restPort         = flag.Int("rest_port", 80, "The port the REST interface listens on.")
	caURI            = flag.String("hmcollector_ca_uri", "", "URI of the CA cert bundle.")
	logInsecFailover = flag.Bool("hmcollector_log_insecure_failover", true, "Log/don't log TLS insecure failovers.")
	httpTimeout      = flag.Int("http_timeout", 10, "Timeout in seconds for HTTP operations.")

	// vars for custom logging configuration
	logModesEnv        = ""
	logXnamesEnv       = ""
	shouldLogErrors    = false
	shouldLogForXnames = false
	logXnames          = make(map[string]struct{})
	logAllowedModes    = []string{"errors"}

	// This is really a hacky option that should only be used when incoming timestamps can't be trusted.
	// For example, if NTP isn't working and the controllers are reporting their time as from 1970.
	IgnoreProvidedTimestamp = flag.Bool("ignore_provided_timestamp", false,
		"Should the collector disregard any provided timestamps and instead use a local value of NOW?")

	serviceName  string
	kafkaBrokers []*hmcollector.KafkaBroker

	Running = true

	RestSRV   *http.Server = nil
	WaitGroup sync.WaitGroup

	ctx context.Context

	smdClient    *hms_certs.HTTPClientPair
	rfClient     *hms_certs.HTTPClientPair
	rfClientLock sync.RWMutex

	atomicLevel zap.AtomicLevel
	logger      *zap.Logger

	RFSubscribeShutdown chan bool
	PollingShutdown     chan bool

	hsmEndpointRefreshShutdown chan bool
	HSMEndpointsLock           sync.Mutex
	HSMEndpoints               map[string]*rf.RedfishEPDescription
)

type EndpointWithCollector struct {
	Endpoint       *rf.RedfishEPDescription
	RiverCollector river_collector.RiverCollector
	LastContacted  *time.Time
	Model          string
}

func doUpdateHSMEndpoints() {
	for Running {
		// Get Redfish endpoints from HSM
		newEndpoints, newEndpointsErr := hmcollector.GetEndpointList(smdClient, *smURL)
		if newEndpoints == nil || len(newEndpoints) == 0 || newEndpointsErr != nil {
			// Ignore and retry on next interval.
			logger.Warn("No endpoints retrieved from State Manager", zap.Error(newEndpointsErr))
		} else {
			for endpointIndex, _ := range newEndpoints {
				newEndpoint := newEndpoints[endpointIndex]

				// Make sure this is a new endpoint.
				HSMEndpointsLock.Lock()
				e, endpointIsKnown := HSMEndpoints[newEndpoint.ID]
				if endpointIsKnown {
					// The credentials are in the endpoint object, therefore,
					// only update the fields that need to stay up to date
					e.DiscInfo.LastAttempt = newEndpoint.DiscInfo.LastAttempt
					e.DiscInfo.LastStatus = newEndpoint.DiscInfo.LastStatus
					e.DiscInfo.RedfishVersion = newEndpoint.DiscInfo.RedfishVersion
				}
				HSMEndpointsLock.Unlock()

				if endpointIsKnown {
					continue
				}

				// No point in wasting our time trying to talk to endpoints HSM wasn't able to.
				if newEndpoint.DiscInfo.LastStatus != "DiscoverOK" {
					logger.Warn("Ignoring endpoint because HSM status not DiscoveredOK",
						zap.Any("newEndpoint", newEndpoint))
					continue
				}

				if *VaultEnabled {
					// Lookup the credentials if we have Vault enabled.
					updateErr := updateEndpointWithCredentials(&newEndpoint)

					if updateErr != nil {
						logger.Error("Unable to update credentials for endpoint with those retrieved from Vault",
							zap.Error(updateErr),
							zap.Any("newEndpoint", newEndpoint))

						// Ignore this endpoint for now, maybe the situation will improve the next time around.
						continue
					} else {
						logger.Debug("Updated endpoint credentials with those retrieved from Vault",
							zap.Any("newEndpoint", newEndpoint))
					}
				}

				HSMEndpointsLock.Lock()
				HSMEndpoints[newEndpoint.ID] = &newEndpoint
				HSMEndpointsLock.Unlock()
			}
		}

		// Use a channel in case we have long refresh intervals so we don't wait around for things to exit.
		select {
		case <-hsmEndpointRefreshShutdown:
			break
		case <-time.After(time.Duration(*hsmRefreshInterval) * time.Second):
			continue
		}
	}

	logger.Info("HSM endpoint monitoring routine shutdown.")
}

func fillMap(m map[string]struct{}, values string) {
	if values != "" {
		for _, v := range strings.Split(values, ",") {
			m[v] = struct{}{}
		}
	}

}

func SetupLogging() {
	logLevel := os.Getenv("LOG_LEVEL")
	logLevel = strings.ToUpper(logLevel)

	atomicLevel = zap.NewAtomicLevel()

	encoderCfg := zap.NewProductionEncoderConfig()
	logger = zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atomicLevel,
	))

	switch logLevel {
	case "DEBUG":
		atomicLevel.SetLevel(zap.DebugLevel)
	case "INFO":
		atomicLevel.SetLevel(zap.InfoLevel)
	case "WARN":
		atomicLevel.SetLevel(zap.WarnLevel)
	case "ERROR":
		atomicLevel.SetLevel(zap.ErrorLevel)
	case "FATAL":
		atomicLevel.SetLevel(zap.FatalLevel)
	case "PANIC":
		atomicLevel.SetLevel(zap.PanicLevel)
	default:
		atomicLevel.SetLevel(zap.InfoLevel)
	}

	// setup the custom logging config
	logModesEnv = os.Getenv("LOG_MODES")
	logModes := make(map[string]struct{})
	fillMap(logModes, logModesEnv)
	_, shouldLogErrors = logModes["errors"]
	logXnamesEnv = os.Getenv("LOG_XNAMES")
	fillMap(logXnames, logXnamesEnv)
	shouldLogForXnames = len(logXnames) > 0

	// log the custom logging config
	logger.Info("Extended logging config", zap.String("modes", logModesEnv), zap.String("xnames", logXnamesEnv), zap.Any("supported modes", logAllowedModes))
	for mode, _ := range logModes {
		validMode := false
		for _, allowedMode := range logAllowedModes {
			if mode == allowedMode {
				validMode = true
				break
			}
		}
		if !validMode {
			logger.Error("Invalid log mode in LOG_MODES environment variable",
				zap.String("invalid mode", mode),
				zap.Any("supported modes", logAllowedModes))
		}
	}
}

// This function is used to set up an HTTP validated/non-validated client
// pair for Redfish operations.  This is done at the start of things, and also
// whenever the CA chain bundle is "rolled".

func createRFClient() error {
	//Wait for all reader locks to release, prevent new reader locks.  Once
	//we acquire this lock, all RF operations are blocked until we unlock.

	//For testing/debug only.
	envstr := os.Getenv("HMCOLLECTOR_VAULT_CA_CHAIN_PATH")
	if envstr != "" {
		logger.Info("Replacing default Vault CA Chain with: ", zap.String("", envstr))
		hms_certs.ConfigParams.CAChainPath = envstr
	}
	envstr = os.Getenv("HMCOLLECTOR_VAULT_PKI_BASE")
	if envstr != "" {
		logger.Info("Replacing default Vault PKI Base with: ", zap.String("", envstr))
		hms_certs.ConfigParams.VaultPKIBase = envstr
	}
	envstr = os.Getenv("HMCOLLECTOR_VAULT_PKI_PATH")
	if envstr != "" {
		logger.Info("Replacing default Vault PKI Path with: ", zap.String("", envstr))
		hms_certs.ConfigParams.PKIPath = envstr
	}

	//Wait for all reader locks to release, prevent new reader locks.  Once
	//we acquire this lock, all RF operations are blocked until we unlock.

	rfClientLock.Lock()
	defer rfClientLock.Unlock()

	logger.Info("All RF threads paused.")
	if *caURI != "" {
		logger.Info("Creating Redfish HTTP client with CA trust bundle from",
			zap.String("", *caURI))
	} else {
		logger.Info("Creating Redfish HTTP client without CA trust bundle.")
	}
	rfc, err := hms_certs.CreateRetryableHTTPClientPair(*caURI, *httpTimeout, 4, 10)
	if err != nil {
		return fmt.Errorf("ERROR: Can't create Redfish HTTP client: %v", err)
	}

	rfClient = rfc
	return nil
}

func caChangeCB(caBundle string) {
	logger.Info("CA bundle rolled; waiting for all RF threads to pause...")
	err := createRFClient()
	if err != nil {
		logger.Error("Can't create TLS-verified HTTP client pair with cert roll, using previous one: ", zap.Error(err))
	} else {
		logger.Info("HTTP transports/clients now set up with new CA bundle.")
	}
}

func main() {
	var err error

	SetupLogging()

	serviceName, err = base.GetServiceInstanceName()
	if err != nil {
		serviceName = "HMCOLLECTOR"
		logger.Error("Can't get service instance name, using ",
			zap.String("", serviceName))
	}
	logger.Info("Service/Instance name:", zap.String("", serviceName))

	// Parse the arguments.
	flag.Parse()

	logger.Info("hmcollector starting...", zap.Any("logLevel", atomicLevel))

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(context.Background())

	hms_certs.ConfigParams.LogInsecureFailover = *logInsecFailover
	hms_certs.InitInstance(nil, serviceName)

	// For performance reasons we'll keep the client that was created for
	// this base request and reuse it later.

	smdClient, err = hms_certs.CreateRetryableHTTPClientPair("", 10, 4, 30)
	if err != nil {
		panic("Can't create insecure cert HTTP client!") //should never happen!
	}

	//Create a TLS-verified HTTP client for Redfish stuff.  Try for a while,
	//fail over if CA-enabled transports can't be created.

	ok := false

	for ix := 1; ix <= 10; ix++ {
		err := createRFClient()
		if err == nil {
			logger.Info("Redfish HTTP client pair creation succeeded.")
			ok = true
			break
		}

		logger.Error("TLS-verified client pair creation failure, ",
			zap.Int("attempt", ix), zap.Error(err))
		time.Sleep(2 * time.Second)
	}

	if !ok {
		logger.Error("Can't create secure HTTP client pair for Redfish, exhausted retries.")
		logger.Error("   Making insecure client; Please check CA bundle URI for correctness.")
		*caURI = ""
		err := createRFClient()
		if err != nil {
			//Should never happen!!
			panic("Can't create Redfish HTTP client!!!")
		}
	}

	if *caURI != "" {
		err := hms_certs.CAUpdateRegister(*caURI, caChangeCB)
		if err != nil {
			logger.Warn("Unable to register CA bundle watcher for ",
				zap.String("URI", *caURI), zap.Error(err))
			logger.Warn("   This means no updates when CA bundle is rolled.")
		}
	} else {
		logger.Warn("No CA bundle URI specified, not watching for CA changes.")
	}

	if *restEnabled {
		// Only enable handling of the root URL if REST is "enabled".
		http.HandleFunc("/", parseRequest)

		logger.Info("REST collection endpoint enabled.")
	}

	// Because we need our liveness/readiness probes to always work, we always setup a HTTP server.
	// NOTE: start the rest server here so we don't die before initialization happens
	WaitGroup.Add(1) // add the wait that main will sit on later
	logger.Info("Starting rest server.")
	doRest()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	hsmEndpointRefreshShutdown = make(chan bool)
	RFSubscribeShutdown = make(chan bool)
	PollingShutdown = make(chan bool)

	go func() {
		<-c
		Running = false

		// Cancel the context to cancel any in progress HTTP requests.
		cancel()

		if *pollingEnabled || *rfSubscribeEnabled {
			hsmEndpointRefreshShutdown <- true
		}

		if *rfSubscribeEnabled {
			RFSubscribeShutdown <- true
		}
		if *pollingEnabled {
			PollingShutdown <- true
		}

		if RestSRV != nil {
			if err := RestSRV.Shutdown(nil); err != nil {
				logger.Panic("Unable to stop REST collection server!", zap.Error(err))
			}
		}
	}()

	HSMEndpoints = make(map[string]*rf.RedfishEPDescription)

	// NOTE: will wait within this function forever if vault doesn't connect
	if *VaultEnabled {
		setupVault()
	}

	// NOTE: will wait within this function forever if kafka doesn't connect
	setupKafka()

	// Always need to keep an up-to-date list of Redfish endpoints assuming there is a mode enabled that can use it.
	if *pollingEnabled || *rfSubscribeEnabled {
		go doUpdateHSMEndpoints()
	}

	if *pollingEnabled {
		if *smURL == "" {
			logger.Panic("State Manager URL can NOT be empty")
		}
		WaitGroup.Add(1)

		logger.Info("Polling enabled.")

		gigabyteCollector = river_collector.GigabyteRiverCollector{}
		intelCollector = river_collector.IntelRiverCollector{}
		hpeCollector = river_collector.HPERiverCollector{}
		openBmcCollector = river_collector.OpenBMCRiverCollector{}

		go doPolling()
	}

	if *rfSubscribeEnabled {
		if *smURL == "" {
			logger.Panic("State Manager URL can NOT be empty!")
		}
		if *restURL == "" {
			logger.Panic("Redfish event target URL can NOT be empty!")
		}
		WaitGroup.Add(1)

		logger.Info("Redfish Event Subscribing enabled.")

		go doRFSubscribe()
	}

	// We'll spend pretty much the rest of life blocking on the next line.
	WaitGroup.Wait()

	// Close the connection to Kafka to make sure any buffered data gets flushed.
	defer func() {
		for idx := range kafkaBrokers {
			thisBroker := kafkaBrokers[idx]

			// This call to Flush is given a maximum timeout of 15 seconds (which is entirely arbitrary and should
			// never take that long). It's very likely this will return almost immediately in most cases.
			abandonedMessages := thisBroker.KafkaProducer.Flush(15 * 1000)
			logger.Info("Closed connection with Kafka broker.",
				zap.Any("broker", thisBroker),
				zap.Int("abandonedMessages", abandonedMessages))
		}

	}()

	// Cleanup any leftover connections...because Go.
	smdClient.CloseIdleConnections()
	rfClient.CloseIdleConnections()

	logger.Info("Exiting...")

	_ = logger.Sync()
}
