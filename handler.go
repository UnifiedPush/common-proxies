package main

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	phttp "github.com/hakobe/paranoidhttp"

	"github.com/karmanyaahm/up_rewrite/config"
	. "github.com/karmanyaahm/up_rewrite/config"
	"github.com/karmanyaahm/up_rewrite/utils"
)

var normalClient *http.Client
var paranoidClient *http.Client

func init() {
	paranoidClient, _, _ = phttp.NewClient()
	paranoidClient.Timeout = 10 * time.Second
	paranoidClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return errors.New("NO")
	}

	normalClient = &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("NO")
		},
		Timeout: 2 * time.Second,
	}
}

//function that runs on (almost) every http request
func bothHandler(f HttpHandler) HttpHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		ConfigLock.RLock()
		defer ConfigLock.RUnlock()
		f(w, r)
	}
}

func gatewayHandler(h Gateway) HttpHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		var nread, nwritten int
		var respType string

		switch r.Method {
		case http.MethodGet:
			w.Write(h.Get())
		case http.MethodPost:
			body, err := ioutil.ReadAll(io.LimitReader(r.Body, 4005))
			r.Body.Close()

			reqs, err := h.Req(body, *r)

			if err != nil {
				errHandle(err, w)
				respType = "err"
				break
			}

			resps := make([]*http.Response, len(reqs))
			for i, req := range reqs {
				if utils.InStringSlice(config.Config.Gateway.AllowedHosts, req.URL.Host) {
					CheckIfRewriteProxy(req.URL.String(), normalClient)
					resps[i], err = normalClient.Do(req)
				} else {
					CheckIfRewriteProxy(req.URL.String(), paranoidClient)
					resps[i], err = paranoidClient.Do(req)
				}

				if err != nil {
					resps[i] = nil
					log.Println(err)
				}
			}

			//process resp
			h.Resp(resps, w)
			respType = "forward"
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			respType = strconv.Itoa(http.StatusMethodNotAllowed)
		}

		log.Println(r.Method, r.URL.Path, r.RemoteAddr, nread, "bytes read;", nwritten, "bytes written;", r.UserAgent(), respType)

		return

	}

}

func proxyHandler(h Proxy) HttpHandler {

	versionWrite := versionHandler()
	return func(w http.ResponseWriter, r *http.Request) {
		var nread, code int = 0, 200
		var respType string

		switch r.Method {

		case http.MethodGet:
			versionWrite(w)
		case http.MethodPost:
			//4000 max so little extra
			body, err := ioutil.ReadAll(io.LimitReader(r.Body, 4005))
			r.Body.Close()

			nread = len(body)
			if nread > 4001 {
				code = http.StatusRequestEntityTooLarge
				break
			}

			req, err := h.Req(body, *r)

			if errHandle(err, w) {
				respType = "err"
				break
			}

			resp, err := normalClient.Do(req)
			if errHandle(err, w) {
				respType = "err"
				break
			}

			//read upto 4000 to be able to reuse conn then close
			ioutil.ReadAll(io.LimitReader(r.Body, 4000))
			resp.Body.Close()

			code = h.RespCode(resp)
			respType = "forward"

		default:
			code = http.StatusMethodNotAllowed
			respType = "method not allowed"
		}
		w.WriteHeader(code)

		log.Println(r.Method, r.Host, r.URL.Path, r.RemoteAddr, nread, "bytes read;", r.UserAgent(), respType)

		return
	}
}

func errHandle(e error, w http.ResponseWriter) bool {
	if e != nil {
		if err, ok := e.(utils.ProxyError); ok {
			logV(err.S.Error())
			w.WriteHeader(err.Code)
			return true

		} else if e.Error() == "length" {
			logV("Too long request")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return true

		} else if e.Error() == "Gateway URL" {
			logV("Unknown URL to forward Gateway request to")
			w.WriteHeader(http.StatusBadRequest)
			return true
		} else {
			if Config.Verbose {
				log.Print("panic-ish: ")
				log.Println(e)
			}
			w.WriteHeader(http.StatusBadGateway)
			return true
		}
	}
	return false
}

func logV(args ...interface{}) {
	if Config.Verbose {
		log.Println(args...)
	}
}
