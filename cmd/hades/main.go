package main

import (
	"github.com/wybiral/hades/internal/client/app"
	"log"
	"os"
)

const version = "0.1.0"

func main() {
	a := app.NewApp()
	a.Name = "hades"
	a.Version = version
	a.Usage = "remote cross-platform daemon manager"
	err := a.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
