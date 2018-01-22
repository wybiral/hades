package main

import (
	"github.com/wybiral/hades/server/app"
	"github.com/wybiral/hades/server/routes"
	"log"
	"net/http"
)

func main() {
	a, err := app.NewApp("app.db")
	if err != nil {
		log.Println(err)
		return
	}
	r := routes.NewRouter(a)
	addr := "127.0.0.1:8080"
	log.Println("Serving at", addr)
	err = http.ListenAndServe(addr, r)
	if err != nil {
		log.Println(err)
	}
}
