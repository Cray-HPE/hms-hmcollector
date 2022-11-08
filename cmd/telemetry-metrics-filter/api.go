package main

import (
	"net/http"

	"go.uber.org/zap"
)

type API struct {
	logger *zap.Logger

	listenString string
}

func (api *API) Start() {
	logger := api.logger

	// Add routes
	http.HandleFunc("/liveness", api.LivenessHandler)
	http.HandleFunc("/readiness", api.ReadinessHandler)
	http.HandleFunc("/health", api.HealthHandler)

	// Start HTTP server
	logger.Info("Starting HTTP server", zap.String("listenAddress", api.listenString))
	http.ListenAndServe(api.listenString, nil)
}

// Kubernetes liveness probe - if this responds with anything other than success (code <400) it will cause the
// pod to be restarted (eventually).
func (api *API) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// Kubernetes liveness probe - if this responds with anything other than success (code <400) multiple times in
// a row it will cause the pod to be restarted.  Only fail this probe for issues that we expect a restart to fix.
func (api *API) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	// logger := api.logger

	ready := true

	// If the Kafka bus isn't good, then return not ready since any incoming data will be dropped.  A restart may not
	// fix this, but it will also keep any traffic from being routed here.
	// NOTE: typically if the Kafka bus is down, the brokers will be created but the producers will not be
	// instantiated yet.

	// TODO need to add a check for kafka health

	if ready {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

func (api *API) HealthHandler(w http.ResponseWriter, r *http.Request) {
	// logger := api.logger
	w.WriteHeader(http.StatusNotImplemented)
}
