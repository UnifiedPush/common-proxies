package main

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"time"

	phttp "github.com/hakobe/paranoidhttp"
	. "github.com/karmanyaahm/up_rewrite/config"
	"github.com/karmanyaahm/up_rewrite/gateway"
	"github.com/karmanyaahm/up_rewrite/rewrite"
)

type handlerAction = func([]byte, http.Request) (*http.Request, *http.Response, error)
type handlerRespAction = func(*http.Response)
type handler struct {
	path       string
	reqAction  handlerAction
	respAction handlerRespAction
	variable   interface{}
	gateway    bool //false if rewrite proxy
}

// various translaters
var handlers = []handler{}

func init() {
	Config = ParseConf("config.toml")
	if Config == nil {
		os.Exit(1)
	}

	handlers = []handler{
		{"/UP", rewrite.Gotify, nil, Config.Rewrite.Gotify, false},
		{"/FCM", rewrite.FCM, nil, Config.Rewrite.FCM, false},
		{"/_matrix/push/v1/notify", gateway.Matrix, gateway.MatrixResp, Config.Gateway.Matrix, true},
	}
}

func main() {
	myRouter := http.NewServeMux()
	for _, i := range handlers {
		if !reflect.ValueOf(i.variable).IsNil() {
			myRouter.HandleFunc(i.path, handle(i))
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

//function that runs on (almost) every http request
func handle(h handler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var nread, nwritten int

		body := make([]byte, 1024*100) //read limited chunk of request body
		nread, err := io.ReadFull(r.Body, body)
		r.Body.Close()
		body = bytes.Trim(body, "\x00")

		req, resp, err := h.reqAction(body, *r)

		var respType string

		if err != nil {
			errHandle(err, w)
			respType = "err"
		} else if req != nil {
			client := &http.Client{} //actually process the translated request

			if h.gateway {
				client, _, _ = phttp.NewClient()
			}
			client.Timeout = 10 * time.Second

			resp, err = client.Do(req)
			if errHandle(err, w) {
				return
			}

			if h.respAction != nil {
				h.respAction(resp)
			}

			defer resp.Body.Close()

			//copy reply into new request
			//for i, j := range resp.Header {
			//	w.Header()[i] = j
			//}
			body, err = ioutil.ReadAll(resp.Body)
			if errHandle(err, w) {
				return
			}

			w.WriteHeader(resp.StatusCode)
			nwritten, err = w.Write(body)
			if errHandle(err, w) {
				return
			}
			respType = "forward"
		} else if resp != nil {
			w.WriteHeader(resp.StatusCode)

			b, _ := ioutil.ReadAll(resp.Body) //TODO handle
			resp.Body.Close()
			w.Write(b)
			//TODO copy headers?

			respType = "return"
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			respType = "unknown error"
		}
		log.Println(r.Method, r.URL.Path, r.RemoteAddr, nread, "bytes read;", nwritten, "bytes written;", r.UserAgent(), respType)

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
				log.Print("panic: ")
				log.Println(e)
			}
			w.WriteHeader(http.StatusBadGateway)
			return true
		}
	}
	return false
}
