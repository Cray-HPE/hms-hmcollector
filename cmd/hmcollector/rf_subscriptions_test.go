// MIT License
//
// (C) Copyright [2023] Hewlett Packard Enterprise Development LP
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
	"os"
	"testing"

	"github.com/Cray-HPE/hms-hmcollector/internal/hmcollector"
)

func TestMain(m *testing.M) {
	setupLoggingForTest()
	m.Run()
}

func setupLoggingForTest() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		os.Setenv("LOG_LEVEL", "debug")
	}
	SetupLogging()
}

func TestTrueCaseForIsSubscriptionForWrongXname(t *testing.T) {
	xname := "x3000c0s19b1"
	wrongXnameSubscriptions := []*hmcollector.EventSubscription{
		&hmcollector.EventSubscription{Destination: "http://10.94.100.71/x3000c0s19b7"},
		&hmcollector.EventSubscription{Destination: "http://10.94.100.71", Context: "x3000c0s19b7"},
	}
	for i, sub := range wrongXnameSubscriptions {
		if !isSubscriptionForWrongXname(xname, sub) {
			t.Errorf("Expected isSubscriptionForWrongXname to return true. "+
				"i: %d, xname: %s, destination: %s, context: %s",
				i, xname, sub.Destination, sub.Context)
		}
	}
}

func TestFalseCaseForIsSubscriptionForWrongXname(t *testing.T) {
	xname := "x3000c0s19b1"
	unnormalizedXname := "x3000c0s19b000001"
	subscriptions := []*hmcollector.EventSubscription{
		&hmcollector.EventSubscription{Destination: "http://10.94.100.71/" + xname, Context: ""},
		&hmcollector.EventSubscription{Destination: "http://10.94.100.71/", Context: xname},
		&hmcollector.EventSubscription{Destination: "http://10.94.100.71", Context: xname},
		&hmcollector.EventSubscription{Destination: "http://10.94.100.71/" + unnormalizedXname, Context: ""},
		&hmcollector.EventSubscription{Destination: "http://10.94.100.71", Context: unnormalizedXname},
		&hmcollector.EventSubscription{Destination: "http://10.94.100.71/" + unnormalizedXname, Context: unnormalizedXname},
		// subscriptions that are not owned by hmcollector
		&hmcollector.EventSubscription{Destination: "http://10.94.100.71", Context: "fabric-manager-hardware-telemetry-sub"},
		&hmcollector.EventSubscription{Destination: "http://10.94.100.71:80", Context: "fabric-manager-hardware-telemetry-sub"},
		&hmcollector.EventSubscription{Destination: "http://10.94.100.71", Context: ""},
		&hmcollector.EventSubscription{Destination: "http://10.94.100.71/junk", Context: ""},
		&hmcollector.EventSubscription{Destination: "", Context: ""},
	}
	for i, sub := range subscriptions {
		if isSubscriptionForWrongXname(xname, sub) {
			t.Errorf("Expected isSubscriptionForWrongXname to return false. "+
				"i: %d, xname: %s, destination: %s, context: %s",
				i, xname, sub.Destination, sub.Context)
		}
	}
}
func TestBadXnameCaseForIsSubscriptionForWrongXname(t *testing.T) {
	xname := "10.94.101.71"
	unnormalizedXname := "x3000c0s19b000001"
	subscriptions := []*hmcollector.EventSubscription{
		&hmcollector.EventSubscription{Destination: "http://" + xname, Context: ""},
		&hmcollector.EventSubscription{Destination: "http://" + xname, Context: xname},
		&hmcollector.EventSubscription{Destination: "http://" + xname, Context: unnormalizedXname},
	}
	for i, sub := range subscriptions {
		if isSubscriptionForWrongXname(xname, sub) {
			t.Errorf("Expected isSubscriptionForWrongXname to return false. "+
				"i: %d, xname: %s, destination: %s, context: %s",
				i, xname, sub.Destination, sub.Context)
		}
	}
}
