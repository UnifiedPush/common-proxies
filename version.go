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

// UnifiedPush used to require returning {"unifiedpush":{"version":1}}
// to allow application to discover UP endpoint. This is no longer the case.
// We keep that request for legacy stuff.
func versionHandler() func(http.ResponseWriter) {
	b, err := json.Marshal(VHandler{UP{1}})
	if err != nil {
		panic(err) //should be const so can panic np
	}
	return func(w http.ResponseWriter) {
		w.Write(b)
	}
}
