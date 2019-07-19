// Package main implements the main hades server command line application.
package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"syscall"

	"github.com/boltdb/bolt"
	"github.com/wybiral/hades/internal/app"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	// setup flags
	host := "127.0.0.1"
	flag.StringVar(&host, "h", host, "server host")
	port := 0
	flag.IntVar(&port, "p", port, "server port")
	setPassword := false
	flag.BoolVar(&setPassword, "s", setPassword, "set password")
	genPassword := false
	flag.BoolVar(&genPassword, "g", genPassword, "generate password")
	flag.Parse()
	a, err := app.NewApp(host, port)
	if err != nil {
		log.Fatal(err)
	}
	// handle password options
	if setPassword {
		err = askForPassword(a)
	} else if genPassword {
		err = generatePassword(a)
	} else {
		err = setupPassword(a)
	}
	if err != nil {
		log.Fatal(err)
	}
	addr := fmt.Sprintf("%s:%d", a.Host, a.Port)
	log.Printf("Local server: http://%s", addr)
	err = a.Run()
	if err != nil {
		log.Fatal(err)
	}
}

// ensures that password hash exists in db or generates one
func setupPassword(a *app.App) error {
	found := false
	err := a.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("settings"))
		h := b.Get([]byte("password-hash"))
		if h != nil {
			found = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	if found {
		return nil
	}
	// generate password if not found
	return generatePassword(a)
}

// generate random password and save hash in database
func generatePassword(a *app.App) error {
	r := make([]byte, 16)
	_, err := rand.Read(r)
	if err != nil {
		return err
	}
	s := base64.RawStdEncoding.EncodeToString(r)
	hash, err := bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = a.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("settings"))
		return b.Put([]byte("password-hash"), hash)
	})
	if err != nil {
		return err
	}
	fmt.Println("New password: " + s)
	return nil
}

// ask user for random password and save hash in database
func askForPassword(a *app.App) error {
	stdin := int(syscall.Stdin)
	fmt.Print("New password: ")
	p1, err := terminal.ReadPassword(stdin)
	if err != nil {
		return err
	}
	fmt.Print("\n")
	fmt.Print("New password (again): ")
	p2, err := terminal.ReadPassword(stdin)
	if err != nil {
		return err
	}
	fmt.Print("\n")
	if !bytes.Equal(p1, p2) {
		log.Fatal("passwords must match")
	}
	hash, err := bcrypt.GenerateFromPassword(p1, bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = a.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("settings"))
		return b.Put([]byte("password-hash"), hash)
	})
	return err
}
