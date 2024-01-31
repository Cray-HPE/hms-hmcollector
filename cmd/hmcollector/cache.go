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
	"sync"
)

type EndpointsCache struct {
	lock  sync.RWMutex
	cache map[string]*EndpointCache
}

type RfType int

const (
	CrayRfType RfType = iota
	GigabyteRfType
	HpeRfType
	IntelRfType
	OpenBmcRfType
	UnknownRfType // The redfish type is not recognized
	UnsetRfType   // The cache has not been set
)

type EndpointCache struct {
	LastDiscoveryAttempt string // The LastDiscoveryAttempt from hsm redfishEndpoints
	Type                 RfType
}

func NewEndpointsCache() *EndpointsCache {
	c := &EndpointsCache{}
	c.lock = sync.RWMutex{}
	c.cache = make(map[string]*EndpointCache)
	return c
}

func (c *EndpointsCache) ReadRfType(endpointId string, lastDiscoveryAttempt string) RfType {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if v, found := c.cache[endpointId]; found {
		if lastDiscoveryAttempt == v.LastDiscoveryAttempt {
			return v.Type
		}
		// cache has a value, however, hsm has since rediscovered the redfish endpoint,
		// and so return UnsetRfType as though there is no cached value.
	}
	return UnsetRfType
}

func (c *EndpointsCache) WriteRfType(endpointId string, lastDiscoveryAttempt string, rfType RfType) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if _, found := c.cache[endpointId]; !found {
		c.cache[endpointId] = &EndpointCache{}
	}

	endpointCache := c.cache[endpointId]
	endpointCache.Type = rfType
	endpointCache.LastDiscoveryAttempt = lastDiscoveryAttempt
}
