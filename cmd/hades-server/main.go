package main

import (
	"flag"
	"fmt"
	"github.com/wybiral/hades/internal/server/app"
	"log"
)

func main() {
	// Setup db flag
	db := "hades.db"
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
	addr := fmt.Sprintf("%s:%d", host, port)
	log.Println("Serving at", addr)
	a, err := app.NewApp(db)
	if err != nil {
		log.Println(err)
		return
	}
	err = a.ListenAndServe(addr)
	if err != nil {
		log.Println(err)
	}
}
