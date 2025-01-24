// MIT License
//
// (C) Copyright [2020-2021,2023,2025] Hewlett Packard Enterprise Development LP
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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	base "github.com/Cray-HPE/hms-base/v2"
	"github.com/Cray-HPE/hms-hmcollector/internal/hmcollector"
	rf "github.com/Cray-HPE/hms-smd/pkg/redfish"
	"github.com/Cray-HPE/hms-xname/xnametypes"
	"go.uber.org/zap"
)

func checkILO(endpoint *rf.RedfishEPDescription) bool {
	// We want to know if the endpoint is iLO or not.  We will read out
	// /redfish/v1/Registries/iLO to make this determination. If the read fails,
	// we assume that this is not iLO. We don't care about the return payload.
	URL := "https://" + endpoint.FQDN + "/redfish/v1/Registries/iLO"
	_, statusCode, err := doHTTPAction(endpoint, http.MethodGet, URL, nil)
	return err == nil && statusCode < 300
}

func checkOpenBmc(endpoint *rf.RedfishEPDescription) bool {
	rfType := GetRedfishType(endpoint)
	return rfType == OpenBmcRfType
}

func getDestination(endpoint *rf.RedfishEPDescription) string {
	destination, err := url.Parse(*restURL)
	if err != nil {
		logger.Error("RF event destination URL invalid",
			zap.String("restURL", *restURL),
			zap.Error(err))
	}
	if destination.Scheme == "http" && checkILO(endpoint) {
		// iLO requires https
		destination.Scheme = "https"
	} else if destination.Scheme == "http" && checkOpenBmc(endpoint) {
		// Open BMC requires https
		destination.Scheme = "https"
	}
	if destination.Port() == "" && *restPort != 80 && *restPort != 443 {
		destination.Host += ":" + strconv.Itoa(*restPort)
	}
	// We are adding the endpoint name to the destination so we have a way to
	// verify/fix the Context for redfish implementations which don't want to
	// properly return the context (iLO).
	destination.Path = "/" + endpoint.ID
	return destination.String()
}

func postRFSubscription(endpoint *rf.RedfishEPDescription, evTypes []string, registryPrefixes []string) (bool, error) {
	// Specify a port if we are not using the default http or https ports.
	// NOTE: This will NOT work with Intel BMCs
	subscribed := false
	sub := hmcollector.EventSubscription{
		Context:          endpoint.ID,
		Destination:      getDestination(endpoint),
		EventTypes:       evTypes,
		Protocol:         "Redfish",
		RegistryPrefixes: registryPrefixes,
	}

	payloadBytes, err := json.Marshal(sub)
	if err != nil {
		return subscribed, err
	}

	fullURL := fmt.Sprintf("https://%s/redfish/v1/EventService/Subscriptions", endpoint.FQDN)
	responsePayloadBytes, statusCode, postErr := doHTTPAction(endpoint, http.MethodPost, fullURL, &payloadBytes)

	postLogger := logger.With(
		zap.String("fullURL", fullURL),
		zap.String("payloadBytes", string(payloadBytes)),
		zap.String("responsePayloadBytes", string(responsePayloadBytes)),
		zap.Int("statusCode", statusCode),
	)
	if postErr != nil {
		postLogger.Error("HTTP request failed for POST attempt!",
			zap.Error(postErr))
	} else if statusCode == 400 && len(registryPrefixes) > 0 {
		postLogger.Info("Endpoint does not appear to support streaming telemetry.", zap.Error(postErr))
	} else if statusCode != http.StatusCreated &&
		statusCode != http.StatusNoContent &&
		statusCode != http.StatusOK {
		postLogger.Error("Got unexpected status code posting Redfish subscription!",
			zap.Int("statusCode", statusCode))
	} else {
		postLogger.Info("Created subscription with endpoint.",
			zap.Any("endpoint", endpoint),
			zap.Any("sub", sub),
			zap.String("string(payloadBytes)", string(payloadBytes)))
		subscribed = true
	}

	return subscribed, err
}

func isSubscriptionForWrongXname(xname string, eventSub *hmcollector.EventSubscription) bool {
	if !*pruneOldSubscriptions {
		return false
	}

	if xnametypes.GetHMSType(xname) == xnametypes.HMSTypeInvalid {
		logger.Debug("The endpoint ID is unexpectedly not an xname when checking for stale subscriptions.",
			zap.Any("xname", xname))
		// This function can only check for bad subscriptions when the xname passed to it is valid
		return false
	}

	xnameOfSub := eventSub.Context
	if xnametypes.GetHMSType(xnameOfSub) == xnametypes.HMSTypeInvalid {
		destinationParts := strings.Split(eventSub.Destination, "/")
		xnameOfSub = destinationParts[len(destinationParts)-1]
		if xnametypes.GetHMSType(xnameOfSub) == xnametypes.HMSTypeInvalid {
			// Neither the Destination nor the Context contain a valid xname
			return false
		}
	}

	xnameOfSubNormalized := xnametypes.NormalizeHMSCompID(xnameOfSub)
	xnameOfEndpointNormalized := xnametypes.NormalizeHMSCompID(xname)
	if xnameOfSubNormalized != xnameOfEndpointNormalized {
		// The destination ends in a valid xname, or the context is a valid xname.
		// The xname does not match the hostname/xname of the redfish endpoint.
		// Conclusion:
		//   The subscription was created by hms-hmcollector and not slingshot or something else.
		//   The subscription is for the wrong endpoint
		//   The hardware was likely physically moved and this old subscription should be deleted.
		logger.Info("Found mismatched subscription",
			zap.String("endpoint", xname),
			zap.String("destination", eventSub.Destination),
			zap.String("context", eventSub.Context),
			zap.String("normalized endpoint xname", xnameOfEndpointNormalized),
			zap.String("normalized subscription xname", xnameOfSubNormalized),
		)
		return true
	}
	return false
}

func deleteSubscriptionForWrongXname(endpoint *rf.RedfishEPDescription, subUri string, eventSub *hmcollector.EventSubscription) {
	// This function should only be used to delete subscriptions where isSubscriptionForWrongXname returns true.
	baseEndpointURL := "https://" + endpoint.FQDN
	fullURL := baseEndpointURL + subUri

	endpointLogger := logger.With(
		zap.Any("xname", endpoint.ID),
		zap.Any("destination", eventSub.Destination),
		zap.Any("context", eventSub.Context),
	)

	endpointLogger.Warn("Subscription does not match endpoint. Deleting the subscription.")

	deleteRetBytes, _, deleteErr := doHTTPAction(endpoint, http.MethodDelete, fullURL, nil)
	if deleteErr != nil {
		endpointLogger.Error("Failed to delete subscription!", zap.Error(deleteErr))
	} else {
		endpointLogger.Info("Deleted subscription.",
			zap.String("string(deleteRetBytes)", string(deleteRetBytes)))
	}
}

func getSubscriptions(endpoint *rf.RedfishEPDescription) (*hmcollector.EventSubscriptionCollection, error) {
	baseEndpointURL := "https://" + endpoint.FQDN
	fullURL := fmt.Sprintf("%s/redfish/v1/EventService/Subscriptions", baseEndpointURL)
	payloadBytes, _, err := doHTTPAction(endpoint, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}

	var subList hmcollector.EventSubscriptionCollection
	err = json.Unmarshal(payloadBytes, &subList)
	if err != nil {
		return nil, err
	}
	return &subList, nil
}

func getSubscription(endpoint *rf.RedfishEPDescription, subUri string) (*hmcollector.EventSubscription, error) {
	baseEndpointURL := "https://" + endpoint.FQDN
	fullURL := baseEndpointURL + subUri

	payloadBytes, _, err := doHTTPAction(endpoint, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}

	var eventSub hmcollector.EventSubscription
	err = json.Unmarshal(payloadBytes, &eventSub)
	if err != nil {
		return nil, err
	}
	return &eventSub, nil
}

func isDupRFSubscription(endpoint *rf.RedfishEPDescription, registryPrefixes []string) (bool, error) {
	baseEndpointURL := "https://" + endpoint.FQDN

	fullURL := fmt.Sprintf("%s/redfish/v1/EventService/Subscriptions", baseEndpointURL)
	payloadBytes, _, err := doHTTPAction(endpoint, http.MethodGet, fullURL, nil)
	if err != nil {
		return false, err
	}

	var subList hmcollector.EventSubscriptionCollection
	err = json.Unmarshal(payloadBytes, &subList)
	if err != nil {
		return false, err
	}

	for _, sub := range subList.Members {
		fullURL = baseEndpointURL + sub.OId

		payloadBytes, _, err := doHTTPAction(endpoint, http.MethodGet, fullURL, nil)
		if err != nil {
			return false, err
		}

		var eventSub hmcollector.EventSubscription
		err = json.Unmarshal(payloadBytes, &eventSub)
		if err != nil {
			return false, err
		}
		if eventSub.Destination == getDestination(endpoint) {
			// Matches this destination, make sure the registry prefix is one we created.
			match := false
			if (registryPrefixes == nil || len(registryPrefixes) == 0) &&
				(eventSub.RegistryPrefixes == nil || len(eventSub.RegistryPrefixes) == 0) {
				match = true
			} else if registryPrefixes != nil {
				hasAllRegistryPrefixes := true
				for _, necessaryRegistryPrefix := range registryPrefixes {
					hasThisRegistryPrefix := false

					for _, presentRegistryPrefix := range eventSub.RegistryPrefixes {
						if necessaryRegistryPrefix == presentRegistryPrefix {
							hasThisRegistryPrefix = true
							break
						}
					}

					if !hasThisRegistryPrefix {
						hasAllRegistryPrefixes = false
						break
					}
				}

				if hasAllRegistryPrefixes {
					match = true
				}
			}

			// If this is a match for destination and registry prefix, make sure context matches and return true.
			if match {
				// Fix context if it does not match.
				if eventSub.Context != endpoint.ID {
					logger.Warn("Existing endpoint subscription has context mismatch, attempting to fix.",
						zap.String("xname", endpoint.ID),
						zap.Strings("registryPrefixes", registryPrefixes),
						zap.Any("incorrectContext", eventSub.Context))

					// Set the correct context.
					match = fixSubscriptionMismatch(endpoint, sub, registryPrefixes)
				}

				// If the context didn't match, we attempted to update to the correct
				// context (can happen when a node changes xname CASMHMS-3200).  If
				// successfully updated return as a valid match.  If the update was unsuccessful
				// it attempted to delete the subscription, so return no match.
				return match, nil
			}
		} else {
			if *pruneOldSubscriptions &&
				isSubscriptionForWrongXname(endpoint.ID, &eventSub) {
				deleteSubscriptionForWrongXname(endpoint, sub.OId, &eventSub)
			}
		}
	}
	return false, nil
}

func fixSubscriptionMismatch(endpoint *rf.RedfishEPDescription, sub hmcollector.EventSubscriptionOdataId,
	registryPrefixes []string) bool {
	// Found a subscription that matches our subscription but has the wrong context.  Attempt to update the context
	// or if that fails delete if and let the correct one be created.
	patchSucceeded := true
	var contextPatch = struct {
		Context string `json:",omitempty"`
	}{endpoint.ID}
	payload, _ := json.Marshal(contextPatch)

	endpointLogger := logger.With(
		zap.Any("xname", endpoint.ID),
		zap.Strings("registryPrefixes", registryPrefixes),
	)

	fullURL := fmt.Sprintf("https://%s", path.Join(endpoint.FQDN, sub.OId))
	patchRetBytes, _, patchErr := doHTTPAction(endpoint, http.MethodPatch, fullURL, &payload)
	if patchErr != nil {
		patchSucceeded = false

		endpointLogger.Warn("Subscription context PATCH failed...attempting to delete subscription.",
			zap.Error(patchErr))

		deleteRetBytes, _, deleteErr := doHTTPAction(endpoint, http.MethodDelete, fullURL, nil)
		if deleteErr != nil {
			endpointLogger.Error("Failed to delete subscription!", zap.Error(deleteErr))
		} else {
			endpointLogger.Info("Deleted subscription.",
				zap.String("string(deleteRetBytes)", string(deleteRetBytes)))
		}
	} else {
		logger.Info("Successfully PATCHed subscription context.",
			zap.String("string(patchRetBytes)", string(patchRetBytes)))
	}

	return patchSucceeded
}

func doGetEventTypes(endpoint *rf.RedfishEPDescription) ([]string, error) {
	fullURL := fmt.Sprintf("https://%s/redfish/v1/EventService", endpoint.FQDN)
	payloadBytes, _, err := doHTTPAction(endpoint, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}

	var evService hmcollector.EventService
	err = json.Unmarshal(payloadBytes, &evService)
	if err != nil {
		return nil, err
	}

	logger.Debug("Got event types from endpoint.",
		zap.Any("xname", endpoint.ID),
		zap.Strings("EventTypesForSubscription", evService.EventTypesForSubscription))

	return evService.EventTypesForSubscription, err
}

// Verify the subscriptions are still in place
func rfVerifySub(verifyRFSubscriptions <-chan hmcollector.RFSub) {
	// This endpoint should already have the subscription present, but verify that it is still there.
	var verifyWaitGroup sync.WaitGroup

	for sub := range verifyRFSubscriptions {
		// If the subscription is still in the process of being set up or is already in an error state then don't
		// check it yet.
		if *sub.Status != hmcollector.RFSUBSTATUS_COMPLETE {
			continue
		}

		// Check the subscriptions in parallel since some of these endpoints can be extremely slow and make the whole
		// process take way longer than it needs to.
		verifyWaitGroup.Add(1)
		go func(inSub hmcollector.RFSub) {
			defer verifyWaitGroup.Done()

			// Check if the subscription is present.
			for _, registryPrefixGroup := range *inSub.PrefixGroups {
				// Check the endpoint to see if we are already subscribed.
				isDup, err := isDupRFSubscription(inSub.Endpoint, registryPrefixGroup)
				if err != nil {
					logger.Error("Unable to check if duplicate subscription!", zap.Error(err))
					*inSub.Status = hmcollector.RFSUBSTATUS_ERROR
					continue
				}
				if !isDup {
					// The subscription should be present but isn't - reset the status so the next time through it
					// will be reset through the normal mechanism.
					logger.Warn("Endpoint missing subscription...resetting status to re-attempt add.",
						zap.String("xname", inSub.Endpoint.ID),
						zap.Strings("registryPrefixGroup", registryPrefixGroup))
					*inSub.Status = hmcollector.RFSUBSTATUS_ERROR
					continue
				}
			}
		}(sub)
	}
	verifyWaitGroup.Wait()
}

func appendUniqueRegPrefix(inPrefix []string, pg *[][]string) {
	// Brute force, but most times this will only have up to two entries in this array.  May need to get smarter if
	// there are ever a lot of subscriptions added.
	if *pg != nil {
		for _, registryPrefixGroup := range *pg {
			// need to see if the two arrays contain the same elements
			matches := len(inPrefix) == len(registryPrefixGroup)
			if matches {
				for _, elem := range inPrefix {
					found := false
					for _, subElem := range registryPrefixGroup {
						if subElem == elem {
							found = true
							break
						}
					}
					if !found {
						matches = false
						break
					}
				}
			}
			if matches {
				// If we found a matching array no need to add anything.
				return
			}
		}
	}

	// nothing found, so add it
	*pg = append(*pg, inPrefix)
}

func rfSubscribe(pendingRFSubscriptions <-chan hmcollector.RFSub) {
	var subscriptionWaitGroup sync.WaitGroup

	for sub := range pendingRFSubscriptions {
		subscriptionWaitGroup.Add(1)

		// Create the subscriptions in parallel since some of these endpoints can be extremely slow and make the
		// whole process take way longer than it needs to.
		go func(sub hmcollector.RFSub) {
			defer subscriptionWaitGroup.Done()

			// Set up the registry prefix groups
			registryPrefixGroups := [][]string{nil}
			if *rfStreamingEnabled {
				// Only create the streaming subscription if enabled.
				rfType := GetRedfishType(sub.Endpoint)
				if rfType != OpenBmcRfType {
					registryPrefixGroups = append(registryPrefixGroups, []string{"CrayTelemetry"})
				}
			}

			if *pruneOldSubscriptions {
				subscriptions, err := getSubscriptions(sub.Endpoint)
				if err != nil {
					logger.Error("Unable to get subscriptions!", zap.String("xname", sub.Endpoint.ID), zap.Error(err))
				} else {
					for _, member := range subscriptions.Members {
						eventSub, err := getSubscription(sub.Endpoint, member.OId)
						if err != nil {
							logger.Error("Unable to get subscription!",
								zap.String("xname", sub.Endpoint.ID),
								zap.String("id", member.OId),
								zap.Error(err))
							continue
						}
						if isSubscriptionForWrongXname(sub.Endpoint.ID, eventSub) {
							deleteSubscriptionForWrongXname(sub.Endpoint, member.OId, eventSub)
						}
					}
				}
			}

			// Set up a subscription for the required registry prefix groups.
			for _, registryPrefixGroup := range registryPrefixGroups {
				// Check the endpoint to see if we are already subscribed.
				isDup, err := isDupRFSubscription(sub.Endpoint, registryPrefixGroup)
				if err != nil {
					logger.Error("Unable to check if duplicate subscription!", zap.Error(err))
					*sub.Status = hmcollector.RFSUBSTATUS_ERROR
					continue
				}
				if isDup {
					// Already present so don't add a second one but log it.
					logger.Debug("Endpoint already contains subscription.",
						zap.Any("endpoint", sub.Endpoint),
						zap.Strings("registryPrefixGroup", registryPrefixGroup))
					*sub.Status = hmcollector.RFSUBSTATUS_COMPLETE
					appendUniqueRegPrefix(registryPrefixGroup, sub.PrefixGroups)
					continue
				}
				// Get the event types that are available.
				evTypes, err := doGetEventTypes(sub.Endpoint)
				if err != nil {
					logger.Error("Unable to check if event types are available!", zap.Error(err))
					*sub.Status = hmcollector.RFSUBSTATUS_ERROR
					continue
				}
				subscribed, err := postRFSubscription(sub.Endpoint, evTypes, registryPrefixGroup)
				if err != nil {
					logger.Error("Unable to post Redfish subscription!", zap.Error(err))
					*sub.Status = hmcollector.RFSUBSTATUS_ERROR
					continue
				}

				if subscribed {
					logger.Info("Redfish subscription created", zap.String("ID", sub.Endpoint.ID), zap.Any("registryPrefix", registryPrefixGroup))
					// Make sure the list of all unique registry prefixes is kept up to date.
					appendUniqueRegPrefix(registryPrefixGroup, sub.PrefixGroups)
				}
			}

			*sub.Status = hmcollector.RFSUBSTATUS_COMPLETE
		}(sub)
	}

	subscriptionWaitGroup.Wait()
}

func doRFSubscribe() {
	endpoints := make(map[string]hmcollector.RFSub)

	// Setup channels for pendingSubscriptions
	pendingRFSubscriptions := make(chan hmcollector.RFSub, 1000)
	verifyRFSubscriptions := make(chan hmcollector.RFSub, 1000)

	defer close(pendingRFSubscriptions)
	defer close(verifyRFSubscriptions)
	defer WaitGroup.Done()

	go rfSubscribe(pendingRFSubscriptions)
	go rfVerifySub(verifyRFSubscriptions)

	// Keep a counter for checking subscriptions frequency.
	subCheckCnt := 0
	const subCheckFreq = 20

	logger.Info("Checking for new Redfish endpoints.", zap.Int("hsmRefreshInterval", *hsmRefreshInterval))
	for Running {
		subCheckCnt++
		logger.Debug("Running new Redfish endpoint scan.", zap.Int("subCheckCnt", subCheckCnt))

		// Determine if any new endpoints have been added.
		hsmEndpointsCache := map[string]*rf.RedfishEPDescription{}
		HSMEndpointsLock.Lock()
		for id, ep := range HSMEndpoints {
			hsmEndpointsCache[id] = ep
		}
		HSMEndpointsLock.Unlock()

		for _, newEndpoint := range hsmEndpointsCache {
			// HPE PDUs don't support subscriptions properly. To prevent tipping it over,
			// don't try subscribe to them.
			if base.GetHMSType(newEndpoint.ID) == base.CabinetPDUController &&
				!strings.Contains(newEndpoint.FQDN, "rts") {
				continue
			}
			if endpoint, ok := endpoints[newEndpoint.ID]; !ok || *endpoint.Status == hmcollector.RFSUBSTATUS_ERROR {
				// New endpoint found. Add it to our list and queue it for subscribing.
				logger.Info("Found new endpoint.", zap.Any("xname", newEndpoint.ID))

				endpointStatus := new(hmcollector.RFSubStatus)
				*endpointStatus = hmcollector.RFSUBSTATUS_PENDING
				endpoints[newEndpoint.ID] = hmcollector.RFSub{
					Endpoint:     newEndpoint,
					Status:       endpointStatus,
					PrefixGroups: &[][]string{},
				}
				pendingRFSubscriptions <- endpoints[newEndpoint.ID]
			} else if subCheckCnt%subCheckFreq == 0 {
				currentEndpoint := endpoints[newEndpoint.ID]
				currentEndpoint.Endpoint = newEndpoint
				endpoints[newEndpoint.ID] = currentEndpoint
				// Endpoint has a subscription, check that sub is still there and correct.
				// NOTE: Don't need to do this at the same frequency as picking up new additions so as to not
				// hammer the endpoint.
				logger.Debug("Verifying subscriptions for endpoint.", zap.String("xname", newEndpoint.ID))
				verifyRFSubscriptions <- endpoints[newEndpoint.ID]
			}
		}

		// Use a channel in case we have long refresh intervals so we don't wait around for things to exit.
		select {
		case <-RFSubscribeShutdown:
			break
		case <-time.After(EndpointRefreshInterval * time.Second):
			continue
		}
	}

	logger.Info("RF subscription routine shutdown.")
}
