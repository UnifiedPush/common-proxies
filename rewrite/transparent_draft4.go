package rewrite

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/karmanyaahm/up_rewrite/utils"
)

type TransparentDraft4 struct {
	Enabled  bool   `env:"UP_REWRITE_TRANSPARENT_DRAFT4_ENABLE"`
	Address  string `env:"UP_REWRITE_TRANSPARENT_DRAFT4_ADDRESS"`
	Scheme   string `env:"UP_REWRITE_TRANSPARENT_DRAFT4_SCHEME"`
	BindPath string `env:"UP_REWRITE_TRANSPARENT_DRAFT4_PATH"`
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
		return []*http.Request{rewrittenRequest}, nil
	} else {
		request, err := http.NewRequest(req.Method, url.String(), bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		return []*http.Request{request}, nil
	}
}

func (proxyImpl TransparentDraft4) RespCode(resp *http.Response) *utils.ProxyError {
	defer resp.Body.Close()
	bodyString, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return utils.NewProxyErrS(resp.StatusCode, string(bodyString))
	} else {
		return utils.NewProxyError(500, err)
	}
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
	}

	proxyImpl.Scheme = strings.ToLower(proxyImpl.Scheme)
	if !(proxyImpl.Scheme == "http" || proxyImpl.Scheme == "https") {
		log.Println("Invalid Endpoint Scheme")
		failed = true
	}
	return
}
