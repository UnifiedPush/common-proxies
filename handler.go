package main

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/hakobe/paranoidhttp"
	phttp "github.com/hakobe/paranoidhttp"

	. "github.com/karmanyaahm/up_rewrite/config"
	"github.com/karmanyaahm/up_rewrite/utils"
)

//function that runs on (almost) every http request
func bothHandler(f HttpHandler) HttpHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		ConfigLock.RLock()
		defer ConfigLock.RUnlock()
		f(w, r)
	}
}

func gatewayHandler(h Gateway) HttpHandler {
	opts := []paranoidhttp.Option{}

	for _, i := range Config.Gateway.AllowedIPs {
		_, n, err := net.ParseCIDR(i)
		if err != nil {
			log.Fatal("error parsing permitted IPs", err)
		}
		opts = append(opts, paranoidhttp.PermittedIPNets(n))
	}

	client, _, _ := phttp.NewClient(opts...)
	client.Timeout = 10 * time.Second
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return errors.New("NO")
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var nread, nwritten int
		var respType string

		switch r.Method {
		case http.MethodGet:
			w.Write(h.Get())
		case http.MethodPost:
			//read upto 20,000 should be enough for any gateway
			body, err := ioutil.ReadAll(io.LimitReader(r.Body, 20000))
			r.Body.Close()

			reqs, err := h.Req(body, *r)

			if err != nil {
				errHandle(err, w)
				respType = "err"
				break
			}

			resps := make([]*http.Response, len(reqs))
			for i, req := range reqs {
				CheckIfRewriteProxy(req.URL.String(), client)
				resps[i], err = client.Do(req)
				//				if errHandle(err, w) {
				//			} //TODO proper err handle
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
	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("NO")
		},
		Timeout: 2 * time.Second,
	}
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

			resp, err := client.Do(req)
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
