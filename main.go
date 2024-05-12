package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/karmanyaahm/up_rewrite/config"
	. "github.com/karmanyaahm/up_rewrite/config"
)

var configFile = flag.String("c", "config.toml", "path to toml file for config")

// various translaters
var handlers = []Handler{}

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}

func main() {
	flag.Parse()
	err := ParseConf(*configFile)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Starting", Config.GetUserAgent())

	handlers = []Handler{
		&Config.Rewrite.FCM,
		&Config.Gateway.Matrix,
		&Config.Gateway.Generic,
	}

	myRouter := http.NewServeMux()

	for _, i := range handlers {
		if i.Path() != "" {
			myRouter.HandleFunc(i.Path(), handle(i))
			if config.Config.Verbose {
				fmt.Println("Handling", i.Path())
			}
		}
	}

	myRouter.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(config.Config.GetUserAgent() + " OK"))
	})
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
				err := config.ParseConf(*configFile)
				if err != nil {
					log.Println("Unable to reload config: ", err)
				}
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

func handle(handler Handler) HttpHandler {
	if h, ok := handler.(Gateway); ok {
		return bothHandler(gatewayHandler(h))
	} else if h, ok := handler.(Proxy); ok {
		return bothHandler(proxyHandler(h))
	} else {
		//should be const so np abt fatal
		log.Fatalf("UNABLE TO HANDLE HANDLER %#v\n", handler)
		return nil
	}
}
