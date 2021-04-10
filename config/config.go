package config

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"

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
	FCM *struct {
		Key string
	}
	Gotify *struct {
		Address string
		Scheme  string
	}
}
type Gateway struct {
	Matrix *struct{}
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

	if c.Rewrite.Gotify != nil {
		if len(c.Rewrite.Gotify.Address) == 0 {
			log.Println("Gotify Address cannot be empty")
			failed = true
		}
		if !(c.Rewrite.Gotify.Scheme == "http" || c.Rewrite.Gotify.Scheme == "https") {
			c.Rewrite.Gotify.Scheme = "https"
			log.Println("Warn: Gotify Scheme incorrect")
		}
	}

	if c.Rewrite.FCM != nil {
		if len(c.Rewrite.FCM.Key) == 0 {
			log.Println("FCM Key cannot be empty")
			failed = true
		}
	}
	return

}
