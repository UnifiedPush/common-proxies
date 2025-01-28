package gateway

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
)

// A Gateway that hanldles any URL in /generic/ENDPOINT_ENCODED/*
// ENDPOINT_ENCODED is just a base64 encoded endpoint
// the rest of common proxies checks that the endpoint is a real UnifiedPush server before pushing to it
// The path strategy is useful for Nextcloud
// and aesgcm style WebPush applications

// NOTE: I'm using RawURLEncoded Base64 here, i.e. the URL Encoded Character set (it's going in a URL after all) and no padding (avoid unnecessary chars). That is also what WebPush most commonly uses.
type Generic struct {
	Enabled bool `env:"UP_GATEWAY_GENERIC_ENABLE"`
	path    string
}

func (m Generic) Load() (err error) {
	// Nothing to do
	return
}

func (m Generic) Path() string {
	if m.Enabled {
		return m.path
	}
	return ""
}

func (m Generic) Get() []byte {
	return []byte(``)
}

func (m Generic) Req(body []byte, req http.Request) ([]*http.Request, error) {
	endpoint := req.URL.Query().Get("e")
	if _, err := url.Parse(endpoint); err != nil {
		return nil, fmt.Errorf("Not valid endpoint: %w", err)
	}
	newReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	newReq.Header.Set("TTL", "86400") // Cache for a day max
	newReq.Header.Set("Urgency", "normal")
	newReq.Header.Set("Content-Encoding", "aes128gcm")
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
	m.path = "/generic/"
	return
}
