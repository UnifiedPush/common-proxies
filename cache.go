package main

import (
	"net/url"
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

func getHost(url *url.URL) string {
	return url.Scheme + "://" + url.Host
}

func getEndpointStatus(url *url.URL) EndpointStatus {
	status, found := endpointCache.Get(getHost(url))
	if found {
		if s, ok := status.(EndpointStatus); ok {
			return s
		}
	}
	status, found = endpointCache.Get(url.String())
	if found {
		if s, ok := status.(EndpointStatus); ok {
			return s
		}
	}
	return NotCached
}

func cacheStatus(id string, status EndpointStatus) {
	dur := cache.DefaultExpiration
	// Cache for 10 minutes if the endpoint is refused
	if status == Refused {
		dur = 10 * time.Minute
	}
	endpointCache.Set(id, status, dur)
}

func setEndpointStatus(url *url.URL, status EndpointStatus) {
	cacheStatus(url.String(), status)
}

func setHostStatus(url *url.URL, status EndpointStatus) {
	cacheStatus(getHost(url), status)
}
