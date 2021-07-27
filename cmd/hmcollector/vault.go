// MIT License
//
// (C) Copyright [2020-2021] Hewlett Packard Enterprise Development LP
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
	"fmt"
	"go.uber.org/zap"
	"time"

	compcredentials "github.com/Cray-HPE/hms-compcredentials"
	securestorage "github.com/Cray-HPE/hms-securestorage"
	rf "github.com/Cray-HPE/hms-smd/pkg/redfish"
)

var compCredStore *compcredentials.CompCredStore

func setupVault() {
	vaultLogger := logger.With(zap.String("vaultAddr", *VaultAddr))

	vaultLogger.Info("Connecting to Vault...")

	for Running {
		// Start a connection to Vault
		if secureStroage, err := securestorage.NewVaultAdapter(""); err != nil {
			vaultLogger.Warn("Unable to connect to Vault! Trying again in 1 second...", zap.Error(err))
			time.Sleep(1 * time.Second)
		} else {
			vaultLogger.Info("Connected to Vault.")

			compCredStore = compcredentials.NewCompCredStore(*VaultKeypath, secureStroage)
			break
		}
	}
}

func updateEndpointWithCredentials(endpoint *rf.RedfishEPDescription) (err error) {
	// make sure vault is up to date
	if compCredStore == nil {
		err = fmt.Errorf("vault not set up for credentials storage")
		return
	}

	credentials, credErr := compCredStore.GetCompCred(endpoint.ID)
	if credErr != nil {
		err = fmt.Errorf("unable to get credentials for endpoint %s: %s", endpoint.ID, credErr)
		return
	}

	endpoint.User = credentials.Username
	endpoint.Password = credentials.Password

	return
}
