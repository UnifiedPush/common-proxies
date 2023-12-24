package gateway

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/karmanyaahm/up_rewrite/utils"
)

// THIS IS CURRENTLY DISABLED, it's not initialized any other part of common proxies

// A Gateway that handles any URL in /generic/ENDPOINT_ENCODED/*
// ENDPOINT_ENCODED is just a base64 encoded endpoint
// the rest of common proxies checks that the endpoint is a real UnifiedPush server before pushing to it
// The path strategy is useful for Nextcloud

// NOTE: I'm using RawURLEncoded Base64 here, i.e. the URL Encoded Character set (it's going in a URL after all) and no padding (avoid unnecessary chars). That is also what WebPush most commonly uses.
type Generic struct {
	Enabled   bool `env:"UP_GATEWAY_GENERIC_ENABLE"`
	path      string
	discovery []byte
}

func (m Generic) Path() string {
	return m.path
}

func (m Generic) Get() []byte {
	return m.discovery
}

func (m Generic) Req(body []byte, req http.Request) ([]*http.Request, error) {
	myurl := req.URL.EscapedPath()
	encodedEndpoint := ""
	if encodedEndpoints := strings.SplitN(myurl, "/", 4); len(encodedEndpoints) >= 3 {
		encodedEndpoint = encodedEndpoints[2]
	}
	endpointBytes, err := base64.RawURLEncoding.DecodeString(encodedEndpoint)
	if err != nil {
		return nil, fmt.Errorf("Encoded endpoint not valid base64: %w", err)
	}
	endpoint := string(endpointBytes)

	// How would I identify Nextcloud requests without parsing JSON?
	// Basic checks are needed to protect against low-effort spammers
	newReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return []*http.Request{newReq}, nil
}

func (Generic) Resp(r []*http.Response, w http.ResponseWriter) {
	if r[0] != nil {
		w.WriteHeader(r[0].StatusCode)
	} else {
		w.WriteHeader(500)
	}
	w.Header().Add("TTL", "0")
}

func (m *Generic) Defaults() (failed bool) {
	if m.Enabled {
		m.path = "/generic/"

		vhandler := utils.VHandler{utils.UP{0, "generic"}}
		var err error
		m.discovery, err = json.Marshal(vhandler)
		if err != nil {
			panic(err) // running at configure time
		}
	}

	return
}
