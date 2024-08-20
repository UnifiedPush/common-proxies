package main

import (
	"time"

	"github.com/patrickmn/go-cache"
)

var endpointCache *cache.Cache

func init() {
	// default expiration time of 1 minute
	// and purges expired items every minutes
	endpointCache = cache.New(1*time.Minute, 1*time.Minute)
}

type EndpointStatus = int32

const (
	NotCached EndpointStatus = iota
	TemporaryUnavailable
	Refused
)

func getEndpointStatus(url string) EndpointStatus {
	status, found := endpointCache.Get(url)
	if found {
		if s, ok := status.(EndpointStatus); ok {
			return s
		}
	}
	return NotCached
}

func setEndpointStatus(url string, status EndpointStatus) {
	dur := cache.DefaultExpiration
	// Cache for 10 minutes if the endpoint is refused
	if status == Refused {
		dur = 10 * time.Minute
	}
	endpointCache.Set(url, status, dur)
}
