// Copyright 2020 Hewlett Packard Enterprise Development LP

package main

import (
	"fmt"
	"go.uber.org/zap"
	"time"

	compcredentials "stash.us.cray.com/HMS/hms-compcredentials"
	securestorage "stash.us.cray.com/HMS/hms-securestorage"
	rf "stash.us.cray.com/HMS/hms-smd/pkg/redfish"
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
