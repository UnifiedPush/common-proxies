package gateway

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/karmanyaahm/up_rewrite/utils"
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
	myurl := req.URL.EscapedPath()
	encodedEndpoint := strings.SplitN(myurl, "/", 4)[2]
	endpointBytes, err := base64.RawURLEncoding.DecodeString(encodedEndpoint)
	if err != nil {
		return nil, fmt.Errorf("Encoded endpoint not valid base64: %w", err)
	}
	endpoint := string(endpointBytes)

	// append WebPush draft 4 (ECE Draft 3) style "aesgcm" headers to the body, so UnifiedPush apps can recieve them
	if req.Header.Get("content-encoding") == "aesgcm" {
		pubkey := req.Header.Get("crypto-key")
		salt := req.Header.Get("encryption")

		if len(pubkey) < 65 || len(salt) < 16 { // approx values
			return nil, utils.NewProxyErrS(400, "Not real WebPush: Headers too short")
		}
		body = append(body, []byte("\n"+pubkey+"\n"+salt+"\naesgcm")...)
	}
	// check that it is either aesgcm, aes128gcm or nextcloud??
	// How would I identify Nextcloud requests without parsing JSON?
	// Basic checks are needed to protect against low-effort spammers
	newReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return []*http.Request{newReq}, nil
}

func (Generic) Resp(r []*http.Response, w http.ResponseWriter) {
	w.WriteHeader(r[0].StatusCode)
}

func (m *Generic) Defaults() (failed bool) {
	m.path = "/generic/"
	return
}
