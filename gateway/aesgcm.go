package gateway

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"

	"codeberg.org/UnifiedPush/common-proxies/utils"
)

// A Gateway that handles any URL in /aesgcm?e=ENDPOINT_ENCODED*
// and puts the aesgcm headers in the body
type Aesgcm struct {
	Enabled   bool `env:"UP_GATEWAY_AESGCM_ENABLE"`
	path      string
	discovery []byte
}

func (m Aesgcm) Load() (err error) {
	// Nothing to do
	return
}

func (m Aesgcm) Path() string {
	return m.path
}

func (m Aesgcm) Get() []byte {
	return m.discovery
}

func (m Aesgcm) Req(body []byte, req http.Request) ([]*http.Request, error) {
	endpoint := req.URL.Query().Get("e")
	if _, err := url.Parse(endpoint); err != nil {
		return nil, fmt.Errorf("Not valid endpoint: %w", err)
	}

	// append WebPush draft 4 (ECE Draft 3) style "aesgcm" headers to the body, so UnifiedPush apps can recieve them
	if val := req.Header.Get("content-encoding"); val != "aesgcm" {
		return nil, fmt.Errorf("Request is not aesgcm: %s", val)
	}
	cryptoKey := req.Header.Get("crypto-key")
	encryption := req.Header.Get("encryption")

	if len(cryptoKey) < 65 || len(encryption) < 16 { // heuristic, not precise
		return nil, utils.NewProxyErrS(400, "Not real aesgcm: Headers too short")
	}
	newBody := []byte("aesgcm" +
		"\nEncryption: " + encryption +
		"\nCrypto-Key: " + cryptoKey +
		"\n")
	newBody = append(newBody, body...)
	newReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(newBody))
	if val := req.Header.Get("TTL"); val != "" {
		newReq.Header.Set("TTL", val)
	} else {
		newReq.Header.Set("TTL", "86400") // Cache for a day max
	}
	if val := req.Header.Get("Urgency"); val != "" {
		newReq.Header.Set("Urgency", val)
	} else {
		newReq.Header.Set("Urgency", "normal")
	}
	if val := req.Header.Get("Content-Encoding"); val != "" {
		newReq.Header.Set("Content-Encoding", val)
	} else {
		newReq.Header.Set("Content-Encoding", "aes128gcm")
	}
	// TODO: Do a draft VAPID to VAPID conversion
	if val := req.Header.Get("Authorization"); val != "" {
		newReq.Header.Set("Authorization", val)
	}
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
		m.path = "/aesgcm"
		m.discovery = []byte(`{"unifiedpush":{"gateway":"aesgcm"}}`)
	}
	return
}
