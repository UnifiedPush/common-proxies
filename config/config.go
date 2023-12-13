package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"

	"github.com/caarlos0/env/v6"
	"github.com/karmanyaahm/up_rewrite/gateway"
	"github.com/karmanyaahm/up_rewrite/rewrite"
	"github.com/komkom/toml"
)

var Version string = "dev"

var Config Configuration
var ConfigLock sync.RWMutex

type Configuration struct {
	MaxUPSize int64 // not user configurable, overriden in defaults

	ListenAddr  string `env:"UP_LISTEN"`
	Verbose     bool   `env:"UP_VERBOSE"`
	UserAgentID string `env:"UP_UAID"`

	Gateway struct {
		AllowedHosts []string `env:"UP_GATEWAY_ALLOWEDHOSTS"`
		Matrix       gateway.Matrix
		Generic      gateway.Generic
	}

	Rewrite struct {
		FCM               rewrite.FCM
		Gotify            rewrite.Gotify
		TransparentDraft4 rewrite.TransparentDraft4
	}
}

var ua string

func (c Configuration) GetUserAgent() string {
	if ua != "" {
		return ua
	}
	ua = "UnifiedPush-Common-Proxies/" + Version
	if Config.UserAgentID != "" {
		ua += " (" + Config.UserAgentID + ")"
	}
	return ua
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

	if err := env.Parse(&config); err != nil {
		return errors.New(fmt.Sprint("Error parsing config file exiting...", err))
	}

	if Defaults(&config) {
		os.Exit(1)
	}
	log.Println("Loading new config")
	Config = config
	return nil
}

func Defaults(c *Configuration) (failed bool) {
	c.MaxUPSize = 4096 // this forces it to be this, ignoring user config
	return c.Rewrite.Gotify.Defaults() ||
		c.Rewrite.TransparentDraft4.Defaults() ||
		c.Rewrite.FCM.Defaults() ||
		c.Gateway.Matrix.Defaults() ||
		c.Gateway.Generic.Defaults()
}
