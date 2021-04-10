package utils

import (
	"bytes"
	"encoding/json"
	"io"
)

func EncodeJSON(inp interface{}) (io.Reader, error) {
	newBody := bytes.NewBuffer([]byte(""))
	e := json.NewEncoder(newBody)
	e.SetEscapeHTML(false)
	e.SetIndent("", "")
	return newBody, e.Encode(inp)

}

func Min(i, j int) (k int) {
	if i < j {
		k = i
	} else {
		k = j
	}
	return
}
