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
		return errors.New("NO redir")
	}

	normalClient = &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("NO redir")
		},
		Timeout: 2 * time.Second,
	}
}

// function that runs on (almost) every http request
func bothHandler(f HttpHandler) HttpHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		ConfigLock.RLock()
		defer ConfigLock.RUnlock()
		f(w, r)
	}
}

func gatewayHandler(h Gateway) HttpHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			nread    int
			nwritten int64
			respType string
			reqs     []*http.Request
		)

	topHandler:
		switch r.Method {
		case http.MethodGet:
			w.Write(h.Get())
		case http.MethodPost:
			body, err := ioutil.ReadAll(io.LimitReader(r.Body, config.Config.MaxUPSize*2)) //gateways contain more than just UP stuff, so extra to be safe
			r.Body.Close()
			nread = len(body)

			reqs, err = h.Req(body, *r)

			if err != nil {
				errHandle(err, w)
				respType = "err"
				break
			}

			resps := make([]*http.Response, len(reqs))
			for i, req := range reqs {
				nwritten += req.ContentLength

				req.Header.Add("User-Agent", Config.GetUserAgent())

				thisClient := paranoidClient
				if utils.InStringSlice(config.Config.Gateway.AllowedHosts, req.URL.Host) {
					thisClient = normalClient
				}
				if !CheckIfRewriteProxy(req.URL.String(), thisClient) {
					errHandle(utils.NewProxyErrS(403, "Target is not a UP Server"), w)
					respType = "err, not UP endpoint"
					break topHandler
				}
				resps[i], err = thisClient.Do(req)

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

		prints := []interface{}{r.Method, r.URL.Path, r.RemoteAddr, nread, "bytes read;", nwritten, "bytes written;", r.UserAgent(), respType, ";"}
		if Config.Verbose {
			hosts := []interface{}{}
			for _, i := range reqs {
				hosts = append(hosts, i.Host)
			}
			prints = append(prints, hosts...)
		}
		log.Println(prints...)
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
			body, err := ioutil.ReadAll(io.LimitReader(r.Body, config.Config.MaxUPSize+1))
			r.Body.Close()

			// Read one extra above to be able to tell whether the request body exceeds or not here
			nread = len(body)
			if nread > int(config.Config.MaxUPSize) {
				code = http.StatusRequestEntityTooLarge
				break
			}

			reqs, err := h.Req(body, *r)

			if errHandle(err, w) {
				respType = "err"
				break
			}

			var resp *http.Response
			for _, req := range reqs {
				resp, err = normalClient.Do(req)
				if errHandle(err, w) {
					respType = "err"
					break
				}

				resperr := h.RespCode(resp)
				code = utils.Max(code, resperr.Code)
				if errHandle(err, w) {
					respType = "err"
					break
				}
				// logic here is that bigger code is worse and should be returned. If one request was ok (200) but one failed (400-500s), the larger one should be returned. It's not perfect, but 🤷

				//read upto 4000 to be able to reuse conn then close
				// this 4000 is arbritary and not related to the size limit
				ioutil.ReadAll(io.LimitReader(r.Body, 4000))
				resp.Body.Close()
			}

			respType = "forward"

		default:
			code = http.StatusMethodNotAllowed
			respType = "method not allowed"
		}
		w.WriteHeader(code)
		w.Header().Add("TTL", "0")
		log.Println(r.Method, r.Host, r.URL.Path, r.RemoteAddr, nread, "bytes read;", r.UserAgent(), respType, code)
	}
}

func errHandle(e error, w http.ResponseWriter) bool {
	if e != nil {
		if err, ok := e.(*utils.ProxyError); ok && (err.S.Error() != "") {
			logV(err.Code, err.S.Error())
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
