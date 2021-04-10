package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/komkom/toml"
)

type Config struct {
	ListenAddr string
	verbose    bool

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

func Parse(location string) *Config {

	config := Config{}
	b, err := ioutil.ReadFile(location)
	b, err = ioutil.ReadAll(toml.New(bytes.NewReader(b)))
	err = json.Unmarshal(b, &config)
	if err != nil {
		log.Println("Error reading/parsing config file", err)
		return nil
	}

	if defaults(&config) {
		return nil
	}
	return &config
}

func defaults(c *Config) (failed bool) {

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
