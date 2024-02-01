// MIT License
//
// (C) Copyright [2024] Hewlett Packard Enterprise Development LP
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
	"net/http"
	"strings"

	rf "github.com/Cray-HPE/hms-smd/pkg/redfish"
	"github.com/Cray-HPE/hms-xname/xnametypes"
	"go.uber.org/zap"
)

const (
	CrayRedfishPath     = "/redfish/v1/Chassis/Enclosure"
	GigabyteRedfishPath = "/redfish/v1/Chassis/Self"
	HpeRedfishPath      = "/redfish/v1/Chassis/1"
	IntelRedfishPath    = "/redfish/v1/Chassis/RackMount"
	OpenBmcRedfishPath  = "/redfish/v1/Chassis/BMC_0"
)

type rfChassis struct {
	Members []rfChassisMembers
}

type rfChassisMembers struct {
	ID string `json:"@odata.id"`
}

func httpGetRedfishType(endpoint *rf.RedfishEPDescription) RfType {
	if xnametypes.GetHMSType(endpoint.ID) != xnametypes.NodeBMC {
		// Skip anything that is not a NodeBMC
		// GET /redfish/v1/Chassis will fail for other types of endpoints
		return UnknownRfType
	}

	URL := "https://" + endpoint.FQDN + "/redfish/v1/Chassis"
	payloadBytes, statusCode, err := doHTTPAction(endpoint, http.MethodGet, URL, nil)
	if err != nil {
		logger.Error("Could not determine the redfish implementation because get /redfish/v1/Chassis failed.",
			zap.Any("endpoint", endpoint),
			zap.Error(err))
		return UnsetRfType
	}
	if statusCode != 200 {
		logger.Error("Could not determine the redfish implementation because get /redfish/v1/Chassis failed.",
			zap.Any("endpoint", endpoint),
			zap.Int("statusCode", statusCode))
		return UnsetRfType

	}
	var chassis rfChassis
	decodeErr := json.Unmarshal(payloadBytes, &chassis)
	if decodeErr != nil {
		logger.Error("Could not determine the redfish implementation, failed to parse response.",
			zap.Any("endpoint", endpoint),
			zap.Error(err))
		return UnknownRfType
	}

	for _, member := range chassis.Members {
		switch {
		case strings.EqualFold(member.ID, CrayRedfishPath):
			return CrayRfType
		case strings.EqualFold(member.ID, GigabyteRedfishPath):
			return GigabyteRfType
		case strings.EqualFold(member.ID, HpeRedfishPath):
			return HpeRfType
		case strings.EqualFold(member.ID, IntelRedfishPath):
			return IntelRfType
		case strings.EqualFold(member.ID, OpenBmcRedfishPath):
			return OpenBmcRfType
		}
	}
	return UnknownRfType
}

func isOpenBmcModel(model string) bool {
	// Paradise has the model P4352/P4353
	return strings.Contains(model, "P4352") ||
		strings.Contains(model, "P4353")
}

func GetRedfishType(endpoint *rf.RedfishEPDescription) RfType {
	rfType := endpointsCache.ReadRfType(endpoint.ID, endpoint.DiscInfo.LastAttempt)
	if rfType == UnsetRfType {
		rfType = httpGetRedfishType(endpoint)
		endpointsCache.WriteRfType(endpoint.ID, endpoint.DiscInfo.LastAttempt, rfType)
	}
	return rfType
}
