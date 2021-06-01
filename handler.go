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

	. "github.com/karmanyaahm/up_rewrite/config"
)

func bothHandler(f HttpHandler) HttpHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		ConfigLock.RLock()
		defer ConfigLock.RUnlock()
		f(w, r)
	}
}

func gatewayHandler(h Gateway) HttpHandler {
	client, _, _ := phttp.NewClient()
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
			body, err := io.ReadAll(io.LimitReader(r.Body, 20000))
			r.Body.Close()

			req, err := h.Req(body, *r)

			if err != nil {
				errHandle(err, w)
				respType = "err"
				break
			}
			resp, err := client.Do(req[0])
			if errHandle(err, w) {
				return
			}

			//process resp
			h.Resp(resp)
			defer resp.Body.Close()
			//start forwarding resp
			body, err = ioutil.ReadAll(resp.Body)
			if errHandle(err, w) {
				return
			}

			w.WriteHeader(resp.StatusCode)
			_, err = w.Write(body)
			if errHandle(err, w) {
				return
			}
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
			body, err := io.ReadAll(io.LimitReader(r.Body, 4005))
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
			io.ReadAll(io.LimitReader(r.Body, 4000))
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
		if e.Error() == "length" {
			if Config.Verbose {
				log.Println("Too long request")
			}
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return true

		} else if e.Error() == "Gateway URL" {
			if Config.Verbose {
				log.Println("Unknown URL to forward Gateway request to")
			}
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
