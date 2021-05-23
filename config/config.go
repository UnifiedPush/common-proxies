package config

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/karmanyaahm/up_rewrite/gateway"
	"github.com/karmanyaahm/up_rewrite/rewrite"
	"github.com/komkom/toml"
)

var Config *Configuration

type Configuration struct {
	ListenAddr string
	Verbose    bool

	Gateway Gateway

	Rewrite Rewrite
}

type Rewrite struct {
	FCM    *rewrite.FCM
	Gotify *rewrite.Gotify
}
type Gateway struct {
	Matrix *gateway.Matrix
}

func ParseConf(location string) *Configuration {

	config := Configuration{}
	b, err := ioutil.ReadFile(location)
	if err != nil {
		log.Println("Unable to find", location, "exiting...")
		return nil
	}
	b, err = ioutil.ReadAll(toml.New(bytes.NewReader(b)))
	err = json.Unmarshal(b, &config)
	if err != nil {
		log.Println("Error parsing config file exiting...", err)
		return nil
	}

	if defaults(&config) {
		return nil
	}
	return &config
}

func defaults(c *Configuration) (failed bool) {
	g := c.Rewrite.Gotify
	if g != nil {
		if len(g.Address) == 0 {
			log.Println("Gotify Address cannot be empty")
			failed = true
		}
		if !(g.Scheme == "http" || c.Rewrite.Gotify.Scheme == "https") {
			g.Scheme = "https"
			log.Println("Warn: Gotify Scheme incorrect")
		}
	}

	f := c.Rewrite.FCM
	if f != nil {
		if len(f.Key) == 0 {
			log.Println("FCM Key cannot be empty")
			failed = true
		}
	}

	m := c.Gateway.Matrix
	if m != nil {
	}
	return

}
