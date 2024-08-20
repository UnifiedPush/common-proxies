package main

import (
	"encoding/json"
	"net/http"
)

type UP struct {
	Version int `json:"version"`
}
type VHandler struct {
	UnifiedPush UP `json:"unifiedpush"`
}

func versionHandler() func(http.ResponseWriter) {
	b, err := json.Marshal(VHandler{UP{1}})
	if err != nil {
		panic(err) //should be const so can panic np
	}
	return func(w http.ResponseWriter) {
		w.Write(b)
	}
}
