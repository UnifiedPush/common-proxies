package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	phttp "github.com/hakobe/paranoidhttp"
	. "github.com/karmanyaahm/up_rewrite/config"
)

type handler Proxy

type Gateway interface {
	Get() []byte
	Resp(*http.Response)
	Proxy
}

type Proxy interface {
	Req([]byte, http.Request) (*http.Request, error)
	Path() string
}

// various translaters
var handlers = []handler{}

func init() {
	Config = ParseConf("config.toml")
	if Config == nil {
		os.Exit(1)
	}

}

func main() {
	myRouter := http.NewServeMux()
	handlers = []handler{
		Config.Rewrite.Gotify,
		Config.Rewrite.FCM,
		Config.Gateway.Matrix,
	}
	for _, i := range handlers {
		if i != nil {
			myRouter.HandleFunc(i.Path(), handle(i))
		}
	}
	myRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Endpoint doesn't exist\n"))
	})

	server := &http.Server{
		Addr:    Config.ListenAddr,
		Handler: myRouter,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		log.Println("Server is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	log.Println("Server is ready to handle requests at", Config.ListenAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %s: %v\n", Config.ListenAddr, err)
	}

	<-done
	log.Println("Server stopped")

}

func versionHandler() func(http.ResponseWriter) {
	b, err := json.Marshal([]string{"UnifiedPush Provider"})
	if err != nil {
		panic(err) //should be const so can panic np
	}
	return func(w http.ResponseWriter) {
		w.Write(b)
	}
}

//function that runs on (almost) every http request
func handle(h handler) func(http.ResponseWriter, *http.Request) {
	if h, ok := h.(Gateway); ok {
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
				resp, err := client.Do(req)
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
	} else { //just proxy not gateway
		client := &http.Client{
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return errors.New("NO")
			},
			Timeout: 2 * time.Second,
		}
		versionWrite := versionHandler()
		return func(w http.ResponseWriter, r *http.Request) {
			var nread, nwritten int
			var respType string

			switch r.Method {

			case http.MethodGet:
				if h, ok := h.(Gateway); ok {
					w.Write(h.Get())
				} else {
					versionWrite(w)
				}
				return

			case http.MethodPost:
				//4000 max so little extra
				body, err := io.ReadAll(io.LimitReader(r.Body, 4005))
				r.Body.Close()

				if len(body) > 4001 {
					w.WriteHeader(http.StatusRequestEntityTooLarge)
					return
				}

				req, err := h.Req(body, *r)

				if err != nil {
					errHandle(err, w)
					respType = "err"
					break
				}
				resp, err := client.Do(req)
				if errHandle(err, w) {
					return
				}

				//read upto 4000 to be able to reuse conn then close
				io.ReadAll(io.LimitReader(r.Body, 4000))
				resp.Body.Close()

				w.WriteHeader(resp.StatusCode)
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
}
func errHandle(e error, w http.ResponseWriter) bool {
	if e != nil {
		if e.Error() == "length" {
			if Config.Verbose {
				log.Println("Too long request")
			}
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			w.Write([]byte("Request is too long\n"))
			return true

		} else if e.Error() == "Gateway URL" {
			if Config.Verbose {
				log.Println("Unknown URL to forward Gateway request to")
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Request format incorrect\n"))
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
