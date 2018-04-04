package app

import (
	"encoding/json"
	"github.com/wybiral/hades/pkg/types"
	"net/http"
)

func jsonResponse(w http.ResponseWriter, obj interface{}) {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(obj)
	if err != nil {
		jsonError(w, "marshalling error")
		return
	}
}

func jsonError(w http.ResponseWriter, msg string) {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	obj := types.Error{
		Error: msg,
	}
	err := encoder.Encode(obj)
	if err != nil {
		return
	}
}
