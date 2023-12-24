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

// A Gateway that handles any URL in /aesgcm/ENDPOINT_ENCODED/*
// ENDPOINT_ENCODED is just a base64 encoded endpoint
// and puts the aesgcm headers in the body
// the rest of common proxies checks that the endpoint is a real UnifiedPush server before pushing to it

// NOTE: I'm using RawURLEncoded Base64 here, i.e. the URL Encoded Character set (it's going in a URL after all) and no padding (avoid unnecessary chars). That is also what WebPush most commonly uses.
type Aesgcm struct {
	Enabled   bool `env:"UP_GATEWAY_AESGCM_ENABLE"`
	path      string
	discovery []byte
}

func (m Aesgcm) Path() string {
	return m.path
}

func (m Aesgcm) Get() []byte {
	return m.discovery
}

func (m Aesgcm) Req(body []byte, req http.Request) ([]*http.Request, error) {
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

	// append WebPush draft 4 (ECE Draft 3) style "aesgcm" headers to the body, so UnifiedPush apps can recieve them
	if req.Header.Get("content-encoding") != "aesgcm" {
		return nil, fmt.Errorf("Request is not aesgcm: %w", err)
	}
	cryptoKey := req.Header.Get("crypto-key")
	encryption := req.Header.Get("encryption")

	if len(cryptoKey) < 65 || len(encryption) < 16 { // heuristic, not precise
		return nil, utils.NewProxyErrS(400, "Not real aesgcm: Headers too short")
	}
	oldBody := body
	body = []byte("aesgcm" +
		"\nEncryption: " + encryption +
		"\nCrypto-Key: " + cryptoKey +
		"\n")
	body = append(body, oldBody...)
	newReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return []*http.Request{newReq}, nil
}

func (Aesgcm) Resp(r []*http.Response, w http.ResponseWriter) {
	if r[0] != nil {
		w.WriteHeader(r[0].StatusCode)
	} else {
		w.WriteHeader(500)
	}
	w.Header().Add("TTL", "0")
}

func (m *Aesgcm) Defaults() (failed bool) {
	if m.Enabled {
		m.path = "/aesgcm/"

		vhandler := utils.VHandler{utils.UP{0, "aesgcm"}}
		var err error
		m.discovery, err = json.Marshal(vhandler)
		if err != nil {
			panic(err) // running at configure time
		}
	}

	return
}
