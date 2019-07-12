package main

import (
	"log"
	"os"

	"github.com/wybiral/hades/internal/client/app"
)

const version = "0.3.0"

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
