package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/karmanyaahm/up_rewrite/utils"
	"github.com/patrickmn/go-cache"
)

var allowedProxies *cache.Cache

func init() {
	//             default cache - if allowed, check every interval
	allowedProxies = cache.New(10*time.Minute, 2*time.Minute)
}

// since this is only run from gateway the http.Client should already ban redirects
func CheckIfRewriteProxy(url string, c *http.Client) bool {
	allowed, found := allowedProxies.Get(url)
	if found {
		return allowed.(bool)
	}

	toAllow := actuallyDecideIfAllowed(url, c)

	//default (10mins) if allowed else 1 min for not allowed
	dur := cache.DefaultExpiration
	if !toAllow {
		dur = 1 * time.Minute
	}
	allowedProxies.Set(url, toAllow, dur)

	return toAllow
}

func actuallyDecideIfAllowed(url string, c *http.Client) bool {
	resp, err := c.Get(url)
	//NOTE should request failing be cached failure or no?
	if err != nil {
		return false
	}

	if resp.StatusCode != http.StatusOK {
		return false
	}

	//NOTE 1000 ought to be enough?
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, 1000))
	if err != nil {
		return false
	}

	v := utils.VHandler{}
	err = json.Unmarshal(body, &v)
	if err != nil {
		return false
	}

	return v.UnifiedPush.Version == 1
}
