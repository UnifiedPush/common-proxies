package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/karmanyaahm/up_rewrite/gateway"
	"github.com/karmanyaahm/up_rewrite/rewrite"
	"github.com/komkom/toml"
)

var Config Configuration
var ConfigLock sync.RWMutex

type Configuration struct {
	ListenAddr string
	Verbose    bool

	Gateway struct {
		AllowedHosts []string
		Matrix       *gateway.Matrix
	}

	Rewrite struct {
		FCM    *rewrite.FCM
		Gotify *rewrite.Gotify
	}
}

func ParseConf(location string) error {
	ConfigLock.Lock()
	defer ConfigLock.Unlock()

	config := Configuration{}
	b, err := ioutil.ReadFile(location)
	if err != nil {
		return errors.New(fmt.Sprint("Unable to find", location, "exiting..."))
	}
	b, err = ioutil.ReadAll(toml.New(bytes.NewReader(b)))
	err = json.Unmarshal(b, &config)
	if err != nil {
		return errors.New(fmt.Sprint("Error parsing config file exiting...", err))
	}

	if defaults(&config) {
		os.Exit(1)
	}
	log.Println("Loading new config")
	Config = config
	return nil
}

func defaults(c *Configuration) (failed bool) {
	g := c.Rewrite.Gotify
	if g != nil {
		if len(g.Address) == 0 {
			log.Println("Gotify Address cannot be empty")
			failed = true
		}

		g.Scheme = strings.ToLower(g.Scheme)
		if !(g.Scheme == "http" || g.Scheme == "https") {
			log.Println("Gotify Scheme incorrect")
			failed = true
		}
	}

	f := c.Rewrite.FCM
	if f != nil {
		if len(f.Key) == 0 {
			log.Println("FCM Key cannot be empty")
			failed = true
		}
		f.APIURL = "https://fcm.googleapis.com/fcm/send"
	}

	m := c.Gateway.Matrix
	if m != nil {
	}
	return

}
