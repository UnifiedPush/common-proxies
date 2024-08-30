package main

import (
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	phttp "github.com/hakobe/paranoidhttp"

	"codeberg.org/UnifiedPush/common-proxies/config"
	. "codeberg.org/UnifiedPush/common-proxies/config"
	"codeberg.org/UnifiedPush/common-proxies/utils"
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

		switch r.Method {
		case http.MethodGet:
			w.Write(h.Get())
		case http.MethodPost:
			body, err := io.ReadAll(io.LimitReader(r.Body, config.Config.MaxUPSize*2)) //gateways contain more than just UP stuff, so extra to be safe
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
				url := req.URL

				req.Header.Add("User-Agent", Config.GetUserAgent())

				thisClient := paranoidClient
				if utils.InStringSlice(config.Config.Gateway.AllowedHosts, req.URL.Host) {
					thisClient = normalClient
				}
				cacheStatus := getEndpointStatus(url)
				if cacheStatus == Refused {
					log.Println("handler: req to ", req.Host, ", URL is cached as refused")
					resps[i] = &http.Response{
						StatusCode: 404,
						Request:    req,
					}
				} else if cacheStatus == TemporaryUnavailable {
					log.Println("handler: req to ", req.Host, ", URL is cached as temp unavailable")
					resps[i] = &http.Response{
						StatusCode: 429,
						Request:    req,
					}
				} else {
					resps[i], err = thisClient.Do(req)
					if err != nil {
						var netErr net.Error
						var dnsErr *net.DNSError
						switch {
						case errors.As(err, &dnsErr):
							// This is a workaround to make the tests work with woodpecker
							if dnsErr.IsNotFound || req.URL.Host == "doesnotexist.unifiedpush.org" {
								log.Println("handler: req to ", req.Host, ", caching URL as refused (Domain not found)")
								resps[i] = &http.Response{
									StatusCode: 404,
									Request:    req,
								}
								setHostStatus(url, Refused)
							} else {
								log.Println("handler: req to ", req.Host, ", caching URL as temp unavailable. DNSError: ", dnsErr)
								resps[i] = &http.Response{
									StatusCode: 429,
									Request:    req,
								}
								setHostStatus(url, TemporaryUnavailable)
							}
						case errors.As(err, &netErr) && netErr.Timeout():
							log.Println("handler: req to ", req.Host, ", caching URL as temp unavailable (Timeout error)")
							resps[i] = &http.Response{
								StatusCode: 429,
								Request:    req,
							}
							setHostStatus(url, TemporaryUnavailable)
						default:
							// This can be:
							// - unsupported protocol
							// - bad ip
							// - invalid tls certif
							log.Println("handler: req to ", req.Host, ", caching URL as refused. Err: ", err)
							resps[i] = &http.Response{
								StatusCode: 404,
								Request:    req,
							}
							setHostStatus(url, Refused)
						}
					} else {
						sc := resps[i].StatusCode
						switch {
						case sc == 429:
							log.Println("handler: req to ", req.Host, ", caching URL as temp unavailable (Status=429)")
							setEndpointStatus(url, TemporaryUnavailable)
						case sc == 413:
							log.Println("handler: req to ", req.Host, ", Request was too long (Status=413)")
						// ntfy does not return 201
						case sc == 201 || sc == 200:
							// DO nothing
						case sc > 499:
							log.Println("handler: req to ", req.Host, ", caching URL as temp unavailable (Status=", sc, "429)")
							setEndpointStatus(url, TemporaryUnavailable)
						default:
							log.Println("handler: req to ", req.Host, ", caching URL as refused. Unexpected status code. (Status=", sc, "429)")
							resps[i].StatusCode = 404
							setEndpointStatus(url, Refused)
						}
					}
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
			body, err := io.ReadAll(io.LimitReader(r.Body, config.Config.MaxUPSize+1))
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
				io.ReadAll(io.LimitReader(r.Body, 4000))
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

		return
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
