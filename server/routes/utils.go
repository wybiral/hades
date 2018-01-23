package routes

import (
	"encoding/json"
	"net/http"
)

func jsonResponse(w http.ResponseWriter, obj interface{}) {
	encoder := json.NewEncoder(w)
	err := encoder.Encode(obj)
	if err != nil {
		jsonError(w, "marshalling error")
		return
	}
}

func jsonError(w http.ResponseWriter, msg string) {
	encoder := json.NewEncoder(w)
	err := encoder.Encode(struct {
		Error string `json:"error"`
	} {
		Error: msg,
	})
	if err != nil {
		return
	}
}
