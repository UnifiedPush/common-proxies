package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"time"

	"github.com/elazarl/goproxy"
	flag "github.com/ogier/pflag"
)

var listenAddr = flag.StringP("listen", "l", "127.0.0.1:5000", "What address to listen on")
var gotifyAddr = flag.String("gotify", "", "What address is gotify on")
var fcmServerKey = flag.String("fcm", "", "Firebase server key - See docs for more info")
var verbose = flag.BoolP("verbose", "v", false, "log all requests")

var proxy *goproxy.ProxyHttpServer

func init() {
	flag.Parse()
}
func main() {
	myRouter := http.NewServeMux()
	for _, i := range handlers {
		if len(*i.variable) > 0 {
			myRouter.HandleFunc(i.path, handle(i.action))
		}
	}
	myRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Endpoint doesn't exist\n"))
	})

	server := &http.Server{
		Addr:    *listenAddr,
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

	log.Println("Server is ready to handle requests at", *listenAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %s: %v\n", *listenAddr, err)
	}

	<-done
	log.Println("Server stopped")

}

//function that runs on (almost) every http request
func handle(translate func([]byte, http.Request) (*http.Request, error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())

		body := make([]byte, 1024*100) //read limited chunk of request body
		io.ReadFull(r.Body, body)
		body = bytes.Trim(body, "\x00")

		req, err := translate(body, *r)
		if errHandle(err, w) {
			return
		}

		client := http.Client{} //actually process the translated request
		resp, err := client.Do(req)
		if errHandle(err, w) {
			return
		}
		defer resp.Body.Close()

		//copy reply into new request
		for i, j := range resp.Header {
			w.Header()[i] = j
		}
		body, err = ioutil.ReadAll(resp.Body)
		if errHandle(err, w) {
			return
		}

		w.WriteHeader(resp.StatusCode)
		_, err = w.Write(body)
		if errHandle(err, w) {
			return
		}

		return
	}
}
func errHandle(e error, w http.ResponseWriter) bool {
	if e != nil {
		if e.Error() == "length" {
			if *verbose {
				log.Println("Too long request")
			}
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			w.Write([]byte("Request is too long\n"))
			return true

		} else {
			if *verbose {
				log.Println("panic")
				log.Println(e)
			}
			w.WriteHeader(http.StatusBadGateway)
			return true
		}
	}
	return false
}

// various translaters
var handlers = []struct {
	path     string
	action   func([]byte, http.Request) (*http.Request, error)
	variable *string
}{
	{"/UP", gotify, gotifyAddr},
	{"/FCM", fcm, fcmServerKey},
}

//gotify
var gotifyRegex = []*regexp.Regexp{
	regexp.MustCompile("\\\\"),
	regexp.MustCompile(`"`),
	regexp.MustCompile(`^`),
	regexp.MustCompile(`$`),
}

func gotify(body []byte, req http.Request) (newReq *http.Request, err error) {

	req.URL.Scheme = "http"
	req.URL.Host = *gotifyAddr
	req.URL.Path = "/message"

	body = gotifyRegex[0].ReplaceAll(body, []byte("\\\\"))
	body = gotifyRegex[1].ReplaceAll(body, []byte(`\"`))
	body = gotifyRegex[2].ReplaceAll(body, []byte(`{"message":"`))
	body = gotifyRegex[3].ReplaceAll(body, []byte(`"}`))

	newReq, err = http.NewRequest(req.Method, req.URL.String(), bytes.NewReader(body))

	for n, h := range req.Header {
		newReq.Header[n] = h
	}

	newReq.Header.Set("Content-Type", "application/json")

	return
}

//fcm
var fcmRegex = []*regexp.Regexp{
	regexp.MustCompile("\\\\"),
	regexp.MustCompile(`"`),
	regexp.MustCompile(`^`),
	regexp.MustCompile(`$`),
}

func fcm(body []byte, req http.Request) (newReq *http.Request, err error) {
	token := req.URL.Query().Get("token")

	if len(body) > 1024*4-4 {
		return nil, errors.New("length")
	}
	body = fcmRegex[0].ReplaceAll(body, []byte("\\\\"))
	body = fcmRegex[1].ReplaceAll(body, []byte(`\\"`))
	body = fcmRegex[2].ReplaceAll(body, []byte(`{"to":"`+token+`","data":{"body":"`))
	body = fcmRegex[3].ReplaceAll(body, []byte("\"}}"))

	newReq, err = http.NewRequest(req.Method, "https://fcm.googleapis.com/fcm/send", bytes.NewReader(body))

	for n, h := range req.Header {
		newReq.Header[n] = h
	}

	newReq.Header.Set("Content-Type", "application/json")
	newReq.Header.Set("Authorization", "key="+*fcmServerKey)

	return
}

// utilities
func min(i, j int) (k int) {
	if i < j {
		k = i
	} else {
		k = j
	}
	return
}
