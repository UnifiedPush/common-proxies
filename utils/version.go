package utils

import (
	"encoding/json"
	"net/http"
)

type UP struct {
	Version int    `json:"version,omitempty"`
	Gateway string `json:"gateway,omitempty"`
}
type VHandler struct {
	UnifiedPush UP `json:"unifiedpush"`
}

var DefaultUnifiedPushVHandler VHandler = VHandler{UP{1, ""}}
var DefaultUnifiedPushVHandlerPayload []byte

func init() {
	var err error
	DefaultUnifiedPushVHandlerPayload, err = json.Marshal(DefaultUnifiedPushVHandler)
	if err != nil {
		panic(err)
	}
}

func VersionHandler() func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.Write(DefaultUnifiedPushVHandlerPayload)
	}
}
