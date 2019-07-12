package api

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/urfave/cli"
)

func getAddr(c *cli.Context) string {
	host := c.GlobalString("host")
	port := c.GlobalInt("port")
	return fmt.Sprintf("http://%s:%d", host, port)
}

// List command returns a list of registered daemons.
func List(c *cli.Context) {
	addr := getAddr(c)
	res, err := http.Get(addr)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	_, err = io.Copy(os.Stdout, res.Body)
	if err != nil {
		log.Fatal(err)
	}
}

// Add command adds a daemon.
func Add(c *cli.Context) {
	args := c.Args()
	if len(args) != 1 {
		cli.ShowCommandHelp(c, "add")
		return
	}
	values := url.Values{}
	values["cmd"] = []string{args[0]}
	key := c.String("key")
	if len(key) > 0 {
		values["key"] = []string{key}
	}
	dir := c.String("dir")
	if len(dir) > 0 {
		values["dir"] = []string{dir}
	}
	addr := getAddr(c)
	res, err := http.PostForm(
		addr,
		values,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	_, err = io.Copy(os.Stdout, res.Body)
	if err != nil {
		log.Fatal(err)
	}
}

// Remove command removes a daemon.
func Remove(c *cli.Context) {
	key := c.String("key")
	if len(key) == 0 {
		cli.ShowCommandHelp(c, "remove")
		return
	}
	args := c.Args()
	if len(args) > 0 {
		cli.ShowCommandHelp(c, "remove")
		return
	}
	addr := getAddr(c)
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", addr+"/"+key, nil)
	if err != nil {
		log.Fatal(err)
	}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	_, err = io.Copy(os.Stdout, res.Body)
	if err != nil {
		log.Fatal(err)
	}
}

// putCommand handles /{key}/{command} PUT requests for controlling daemons.
func putCommand(c *cli.Context, command string) {
	key := c.String("key")
	if len(key) == 0 {
		cli.ShowCommandHelp(c, command)
		return
	}
	args := c.Args()
	if len(args) > 0 {
		cli.ShowCommandHelp(c, command)
		return
	}
	addr := getAddr(c)
	client := &http.Client{}
	req, err := http.NewRequest("PUT", addr+"/"+key+"/"+command, nil)
	if err != nil {
		log.Fatal(err)
	}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	_, err = io.Copy(os.Stdout, res.Body)
	if err != nil {
		log.Fatal(err)
	}
}

// Start command starts a daemon.
func Start(c *cli.Context) {
	putCommand(c, "start")
}

// Stop command stops a daemon.
func Stop(c *cli.Context) {
	putCommand(c, "stop")
}

// Pause command pauses a daemon.
func Pause(c *cli.Context) {
	putCommand(c, "pause")
}

// Continue command continues a paused daemon.
func Continue(c *cli.Context) {
	putCommand(c, "continue")
}
