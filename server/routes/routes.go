package routes

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/wybiral/hades/server/app"
	"log"
	"net/http"
)

func NewRouter(a *app.App) *mux.Router {
	withApp := AppMiddleware(a)
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/", withApp(indexHandler)).Methods("GET")
	r.HandleFunc("/{key}", withApp(daemonGetHandler)).Methods("GET")
	r.HandleFunc("/{key}/start", withApp(daemonStartHandler))
	r.HandleFunc("/{key}/stop", withApp(daemonStopHandler))
	r.HandleFunc("/add", withApp(addPostHandler)).Methods("POST")
	return r
}

// Respond with JSON list of daemons
func indexHandler(a *app.App, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	daemons, err := a.GetDaemons()
	if err != nil {
		return
	}
	encoder := json.NewEncoder(w)
	err = encoder.Encode(daemons)
	if err != nil {
		return
	}
}

// Respond with JSON object for one daemon
func daemonGetHandler(a *app.App, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	key := vars["key"]
	daemon, err := a.GetDaemon(key)
	if err != nil {
		return
	}
	encoder := json.NewEncoder(w)
	err = encoder.Encode(daemon)
	if err != nil {
		return
	}
}

// Start one daemon
func daemonStartHandler(a *app.App, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	key := vars["key"]
	err := a.StartDaemon(key)
	if err != nil {
		return
	}
	encoder := json.NewEncoder(w)
	err = encoder.Encode("OK")
	if err != nil {
		return
	}
}

// Stop one daemon
func daemonStopHandler(a *app.App, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	key := vars["key"]
	err := a.StopDaemon(key)
	if err != nil {
		return
	}
	encoder := json.NewEncoder(w)
	err = encoder.Encode("OK")
	if err != nil {
		return
	}
}

// Add new daemon from cmd string
func addPostHandler(a *app.App, w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := r.PostForm.Get("key")
	cmd := r.PostForm.Get("cmd")
	if len(cmd) == 0 {
		return
	}
	daemon, err := a.CreateDaemon(key, cmd)
	if err != nil {
		log.Println(err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err = encoder.Encode(daemon)
	if err != nil {
		return
	}
}
