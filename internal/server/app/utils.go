package app

import (
	"encoding/json"
	"net/http"

	"github.com/wybiral/hades/pkg/types"
)

// jsonResponse writes obj JSON to http.ResponseWriter.
func jsonResponse(w http.ResponseWriter, obj interface{}) {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(obj)
	if err != nil {
		jsonError(w, "marshalling error")
		return
	}
}

// jsonResponse writes error msg JSON to http.ResponseWriter.
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
