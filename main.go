package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/karmanyaahm/up_rewrite/config"
	. "github.com/karmanyaahm/up_rewrite/config"
)

var configFile = flag.String("conf", "config.toml", "path to toml file for config")

type HttpHandler func(http.ResponseWriter, *http.Request)
type Gateway interface {
	Handler
	Get() []byte
	Resp(*http.Response)
}

type Proxy interface {
	Handler
	RespCode(*http.Response) int
}

type Handler interface {
	Req([]byte, http.Request) (*http.Request, error)
	Path() string
}

// various translaters
var handlers = []Handler{}

func init() {
	Config = ParseConf(*configFile)
	if Config == nil {
		os.Exit(1)
	}
}

func main() {
	myRouter := http.NewServeMux()
	handlers = []Handler{
		Config.Rewrite.Gotify,
		Config.Rewrite.FCM,
		Config.Gateway.Matrix,
	}
	for _, i := range handlers {
		if !reflect.ValueOf(i).IsNil() {
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
	signal.Notify(quit, os.Interrupt, syscall.SIGHUP)

	go func() {
		for {
			switch <-quit {
			case syscall.SIGHUP:
				config.ParseConf(*configFile)
				log.Println("Reloading conf")
			case os.Interrupt:
				log.Println("Server is shutting down...")

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				server.SetKeepAlivesEnabled(false)
				if err := server.Shutdown(ctx); err != nil {
					log.Fatalf("Could not gracefully shutdown the server: %v\n", err)
				}
				close(done)
				return
			default:
				log.Println("UNKNOWN SIGNAL")
			}
		}
	}()

	log.Println("Server is ready to handle requests at", Config.ListenAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %s: %v\n", Config.ListenAddr, err)
	}

	<-done
	log.Println("Server stopped")

}

//function that runs on (almost) every http request
func handle(handler Handler) HttpHandler {
	if h, ok := handler.(Gateway); ok {
		return gatewayHandler(h)
	} else if h, ok := handler.(Proxy); ok {
		return proxyHandler(h)
	} else {
		//should be const so np abt fatal
		log.Fatalf("UNABLE TO HANDLE HANDLER %#v\n", handler)
		return nil
	}
}
