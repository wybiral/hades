package main

import (
	"flag"
	"fmt"
	"github.com/wybiral/hades/server/app"
	"github.com/wybiral/hades/server/routes"
	"log"
	"net/http"
)

func main() {
	// Setup db flag
	db := "app.db"
	flag.StringVar(
		&db,
		"db",
		db,
		"specify database file",
	)
	// Setup server host flag
	host := "127.0.0.1"
	flag.StringVar(
		&host,
		"host",
		host,
		"specify server host",
	)
	// Setup server port flag
	port := 8666
	flag.IntVar(
		&port,
		"port",
		port,
		"specify server port",
	)
	flag.Parse()
	a, err := app.NewApp(db)
	if err != nil {
		log.Println(err)
		return
	}
	r := routes.NewRouter(a)
	addr := fmt.Sprintf("%s:%d", host, port)
	log.Println("Serving at", addr)
	err = http.ListenAndServe(addr, r)
	if err != nil {
		log.Println(err)
	}
}
