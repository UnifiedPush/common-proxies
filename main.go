package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"codeberg.org/UnifiedPush/common-proxies/config"
	. "codeberg.org/UnifiedPush/common-proxies/config"
	"codeberg.org/UnifiedPush/common-proxies/vapid"
)

var configFile = flag.String("c", "config.toml", "path to toml file for config")
var genVapidFlag = flag.Bool("vapid", false, "Generate a new VAPID private key and exit")

// various translaters
var handlers = []Handler{}

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}

func genVapid() {
	private, err := vapid.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalln(err)
	}
	out, err := vapid.EncodePriv(*private)
	fmt.Println(out)
}

func main() {
	flag.Parse()

	if *genVapidFlag {
		genVapid()
		return
	}

	err := ParseConf(*configFile)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Starting", Config.GetUserAgent())

	handlers = []Handler{
		&Config.Rewrite.FCM,
		&Config.Rewrite.WebPushFCM,
		&Config.Gateway.Matrix,
		&Config.Gateway.Generic,
		&Config.Gateway.Aesgcm,
	}

	myRouter := http.NewServeMux()
	stopTickers := make(chan bool)

	for _, i := range handlers {
		i.Load()
		if i.Path() != "" {
			myRouter.HandleFunc(i.Path(), handle(i))
			if config.Config.Verbose {
				fmt.Println("Handling", i.Path())
			}
		}
		handleTicker(i, stopTickers)
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

				stopTickers <- true
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

func handleTicker(handler Handler, done chan bool) {
	if h, ok := handler.(TickerHandler); ok {
		ticker := time.NewTicker(h.Duration())
		go func() {
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					h.Tick()
				}
			}
		}()
	}
}
