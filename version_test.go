package main

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

func ForceBypassValidProviderCheck(host ...string) {
	for _, i := range host {
		allowedProxies.Set(i, true, 1000*time.Minute)
	}
}

func TestCheckRewriteProxy(t *testing.T) {
	c := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("A")
		},
	}

	f := func(w http.ResponseWriter) {
		fmt.Fprintln(w, "Hello, client")
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { f(w) }))
	defer ts.Close()

	assert.False(t, actuallyDecideIfAllowed(ts.URL, c), "wrong data should be false")

	f = func(w http.ResponseWriter) {
		fmt.Fprintln(w, `{"unifiedpush": {"version":1}, "b":1, "c": "useless stuff"}`)
	}
	assert.True(t, actuallyDecideIfAllowed(ts.URL, c), "correct url should be true")

	f = func(w http.ResponseWriter) {
		w.WriteHeader(302)
	}
	assert.False(t, actuallyDecideIfAllowed(ts.URL, c), "redirects shouldn't be followed")
}

func TestRewriteProxyCache(t *testing.T) {
	allowedProxies = cache.New(500*time.Millisecond, 100*time.Millisecond)
	c := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("A")
		},
	}

	f := func(w http.ResponseWriter) {
		fmt.Fprintln(w, `{"unifiedpush": {"version":1}}`)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { f(w) }))
	defer ts.Close()

	assert.True(t, CheckIfRewriteProxy(ts.URL, c), "correct data should be true")

	f = func(w http.ResponseWriter) {
		fmt.Fprintln(w, `abc`)
	}
	assert.True(t, CheckIfRewriteProxy(ts.URL, c), "should still be cached true")

	time.Sleep(700 * time.Millisecond)
	assert.False(t, CheckIfRewriteProxy(ts.URL, c), "cache should've expired to see the real false")
}

//TODO test negative cache (would require mock of duration)

func TestVersionHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	versionHandler()(rec)

	assert.Equal(t, `{"unifiedpush":{"version":1}}`, rec.Body.String(), "version handler is wrong")
}
