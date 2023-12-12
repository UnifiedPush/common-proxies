package rewrite

import (
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/karmanyaahm/up_rewrite/utils"
)

type Draft4 struct {
	Enabled  bool   `env:"UP_REWRITE_DRAFT4_ENABLE"`
	Address  string `env:"UP_REWRITE_DRAFT4_ADDRESS"`
	Scheme   string `env:"UP_REWRITE_DRAFT4_SCHEME"`
	BindPath string `env:"UP_REWRITE_DRAFT4_PATH"`
}

func (proxyImpl Draft4) Path() string {
	if proxyImpl.Enabled {
		if len(proxyImpl.BindPath) > 0 && proxyImpl.BindPath[len(proxyImpl.BindPath)-1] != '/' {
			proxyImpl.BindPath += "/"
		}
		return proxyImpl.BindPath
	}
	return ""
}

func (proxyImpl Draft4) Req(body []byte, req http.Request) ([]*http.Request, error) {

	url := *req.URL
	url.Scheme = proxyImpl.Scheme
	url.Host = proxyImpl.Address

	panic("draft4 Gateway not implemented")
}

func (proxyImpl Draft4) RespCode(resp *http.Response) *utils.ProxyError {
	defer resp.Body.Close()
	bodyString, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return utils.NewProxyErrS(resp.StatusCode, string(bodyString))
	} else {
		return utils.NewProxyError(500, err)
	}
}

func (proxyImpl *Draft4) Defaults() (failed bool) {
	failed = false
	if !proxyImpl.Enabled {
		return
	}

	if proxyImpl.BindPath == "" {
		proxyImpl.BindPath = "/"
	}

	if len(proxyImpl.Address) == 0 {
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
