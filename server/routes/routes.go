package routes

import (
	"github.com/gorilla/mux"
	"github.com/wybiral/hades/server/app"
	"net/http"
)

func NewRouter(a *app.App) *mux.Router {
	withApp := AppMiddleware(a)
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/", withApp(indexGetHandler)).Methods("GET")
	r.HandleFunc("/", withApp(indexPostHandler)).Methods("POST")
	r.HandleFunc("/{key}", withApp(daemonGetHandler)).Methods("GET")
	r.HandleFunc("/{key}/start", withApp(daemonStartHandler))
	r.HandleFunc("/{key}/stop", withApp(daemonStopHandler))
	return r
}

// Respond with JSON list of daemons
func indexGetHandler(a *app.App, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	daemons, err := a.GetDaemons()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError(w, "database error")
		return
	}
	jsonResponse(w, daemons)
}

// Add new daemon from cmd string
func indexPostHandler(a *app.App, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	r.ParseForm()
	key := r.PostForm.Get("key")
	cmd := r.PostForm.Get("cmd")
	if len(cmd) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		jsonError(w, "cmd required")
		return
	}
	daemon, err := a.CreateDaemon(key, cmd)
	if err == app.ErrKeyNotUnique {
		w.WriteHeader(http.StatusBadRequest)
		jsonError(w, "key already exists")
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError(w, "database error")
		return
	}
	jsonResponse(w, daemon)
}

// Respond with JSON object for one daemon
func daemonGetHandler(a *app.App, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	key := vars["key"]
	daemon, err := a.GetDaemon(key)
	if err == app.ErrNotFound {
		w.WriteHeader(http.StatusNotFound)
		jsonError(w, "not found")
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError(w, "database error")
		return
	}
	jsonResponse(w, daemon)
}

// Start one daemon
func daemonStartHandler(a *app.App, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	key := vars["key"]
	err := a.StartDaemon(key)
	if err == app.ErrNotFound {
		w.WriteHeader(http.StatusNotFound)
		jsonError(w, "not found")
		return
	} else if err == app.ErrAlreadyRunning {
		w.WriteHeader(http.StatusBadRequest)
		jsonError(w, "already running")
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError(w, "database error")
		return
	}
	jsonResponse(w, struct{}{})
}

// Stop one daemon
func daemonStopHandler(a *app.App, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	key := vars["key"]
	err := a.StopDaemon(key)
	if err == app.ErrNotRunning {
		w.WriteHeader(http.StatusBadRequest)
		jsonError(w, "not running")
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		jsonError(w, "database error")
		return
	}
	jsonResponse(w, struct{}{})
}
