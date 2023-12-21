package gateway

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/karmanyaahm/up_rewrite/utils"
)

type TransparentDraft4 struct {
	Enabled    bool   `env:"UP_GATEWAY_TRANSPARENT_DRAFT4_ENABLE"`
	Address    string `env:"UP_GATEWAY_TRANSPARENT_DRAFT4_ADDRESS"`
	Scheme     string `env:"UP_GATEWAY_TRANSPARENT_DRAFT4_SCHEME"`
	BindPath   string `env:"UP_GATEWAY_TRANSPARENT_DRAFT4_PATH"`
	GetPayload []byte
}

func (proxyImpl TransparentDraft4) Path() string {
	if proxyImpl.Enabled {
		if len(proxyImpl.BindPath) > 0 && proxyImpl.BindPath[len(proxyImpl.BindPath)-1] != '/' {
			proxyImpl.BindPath += "/"
		}
		return proxyImpl.BindPath
	}
	return ""
}

func (proxyImpl TransparentDraft4) Get() []byte {
	return proxyImpl.GetPayload
}

func (proxyImpl TransparentDraft4) Req(body []byte, req http.Request) ([]*http.Request, error) {
	url := *req.URL
	url.Scheme = proxyImpl.Scheme
	url.Host = proxyImpl.Address

	if req.Header.Get("content-encoding") == "aesgcm" {
		rewrittenBody := new(bytes.Buffer)
		var err error
		bodyFragments := []string{
			"aesgcm\r\nEncryption: ",
			req.Header.Get("Encryption"),
			"\r\nCrypto-Key: ",
			req.Header.Get("Crypto-Key"),
			"\r\n",
		}
		for _, fragment := range bodyFragments {
			_, err = rewrittenBody.WriteString(fragment)
			if err != nil {
				return nil, err
			}
		}
		_, err = rewrittenBody.Write(body)
		if err != nil {
			return nil, err
		}
		rewrittenRequest, err := http.NewRequest(req.Method, url.String(), rewrittenBody)
		if err != nil {
			return nil, err
		}
		rewrittenRequest.Header = req.Header.Clone()
		return []*http.Request{rewrittenRequest}, nil
	} else {
		request, err := http.NewRequest(req.Method, url.String(), bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		request.Header = req.Header.Clone()
		return []*http.Request{request}, nil
	}
}

func (proxyImpl TransparentDraft4) Resp(responseArray []*http.Response, gatewayResponse http.ResponseWriter) {
	response := responseArray[0]
	bodyBytes, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		gatewayResponse.Write([]byte(err.Error()))
		gatewayResponse.WriteHeader(500)
	}
	gatewayResponse.Write(bodyBytes)
	for key, value := range response.Header {
		gatewayResponse.Header()[http.CanonicalHeaderKey(key)] = value
	}
	gatewayResponse.WriteHeader(response.StatusCode)
}

func (proxyImpl *TransparentDraft4) Defaults() (failed bool) {
	failed = false
	if !proxyImpl.Enabled {
		return
	}

	if proxyImpl.BindPath == "" {
		proxyImpl.BindPath = "/"
	}

	if len(proxyImpl.Address) <= 0 {
		log.Println("Endpoint Address cannot be empty")
		failed = true
		return
	}

	proxyImpl.Scheme = strings.ToLower(proxyImpl.Scheme)
	if !(proxyImpl.Scheme == "http" || proxyImpl.Scheme == "https") {
		log.Println("Invalid Endpoint Scheme")
		failed = true
		return
	}

	var err error
	proxyImpl.GetPayload, err = json.Marshal(utils.DefaultUnifiedPushVHandler)
	if err != nil {
		log.Println(err)
		failed = true
		return
	}

	return
}
