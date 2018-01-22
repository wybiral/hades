package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/wybiral/hades/server/app"
	"log"
	"net/http"
)

func main() {
	a, err := app.NewApp("app.db")
	if err != nil {
		log.Println(err)
		return
	}
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		indexHandler(a, w, r)
	}).Methods("GET")
	r.HandleFunc("/{key}", func(w http.ResponseWriter, r *http.Request) {
		daemonGetHandler(a, w, r)
	}).Methods("GET")
	r.HandleFunc("/{key}/start", func(w http.ResponseWriter, r *http.Request) {
		daemonStartHandler(a, w, r)
	})
	r.HandleFunc("/{key}/stop", func(w http.ResponseWriter, r *http.Request) {
		daemonStopHandler(a, w, r)
	})
	r.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		addPostHandler(a, w, r)
	}).Methods("POST")
	addr := "127.0.0.1:8080"
	log.Println("Serving at", addr)
	err = http.ListenAndServe(addr, r)
	if err != nil {
		log.Println(err)
	}
}

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

func addPostHandler(a *app.App, w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	cmd := r.PostForm.Get("cmd")
	if len(cmd) == 0 {
		return
	}
	key, err := a.CreateDaemon(cmd)
	if err != nil {
		log.Println(err)
		return
	}
	daemon, err := a.GetDaemon(key)
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
