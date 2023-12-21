package utils

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

var DefaultUnifiedPushVHandler VHandler = VHandler{UP{1}}

func VersionHandler() func(http.ResponseWriter) {
	b, err := json.Marshal(DefaultUnifiedPushVHandler)
	if err != nil {
		panic(err) //should be const so can panic np
	}
	return func(w http.ResponseWriter) {
		w.Write(b)
	}
}
