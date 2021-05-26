package main

import (
	"encoding/json"
	"net/http"
)

func versionHandler() func(http.ResponseWriter) {
	b, err := json.Marshal(map[string]interface{}{
		"unifiedpush": map[string]int{
			"version": 1,
		},
	})
	if err != nil {
		panic(err) //should be const so can panic np
	}
	return func(w http.ResponseWriter) {
		w.Write(b)
	}
}
