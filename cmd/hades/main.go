package main

import (
	"fmt"
	"github.com/urfave/cli"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
)

const version = "0.1.0"

func main() {
	app := cli.NewApp()
	app.Version = version
	app.Name = "hades"
	app.Usage = "remote cross-platform daemon manager"
	cli.HelpFlag = cli.StringFlag{Hidden: true}
	cli.VersionFlag = cli.StringFlag{Hidden: true}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "host, h",
			Value: "127.0.0.1",
			Usage: "Server host",
		},
		cli.IntFlag{
			Name:  "port, p",
			Value: 8666,
			Usage: "Server port",
		},
	}
	keyFlag := cli.StringFlag{
		Name:  "key, k",
		Usage: "Daemon key value",
	}
	app.Commands = []cli.Command{
		cli.Command{
			Name:      "list",
			ArgsUsage: " ",
			Usage:     "List all daemons",
			Action:    cmdList,
		},
		cli.Command{
			Name:      "add",
			ArgsUsage: "COMMAND",
			Usage:     "Add daemon",
			Action:    cmdAdd,
			Flags: []cli.Flag{
				keyFlag,
				cli.StringFlag{
					Name:  "dir, d",
					Usage: "Directory for daemon",
				},
			},
		},
		cli.Command{
			Name:      "remove",
			ArgsUsage: " ",
			Usage:     "Remove daemon",
			Action:    cmdRemove,
			Flags: []cli.Flag{
				keyFlag,
			},
		},
		cli.Command{
			Name:      "start",
			ArgsUsage: " ",
			Usage:     "Start daemon",
			Action:    cmdStart,
			Flags: []cli.Flag{
				keyFlag,
			},
		},
		cli.Command{
			Name:      "stop",
			ArgsUsage: " ",
			Usage:     "Stop daemon",
			Action:    cmdStop,
			Flags: []cli.Flag{
				keyFlag,
			},
		},
		cli.Command{
			Name:      "pause",
			ArgsUsage: " ",
			Usage:     "Pause daemon",
			Action:    cmdPause,
			Flags: []cli.Flag{
				keyFlag,
			},
		},
		cli.Command{
			Name:      "continue",
			ArgsUsage: " ",
			Usage:     "Continue daemon",
			Action:    cmdContinue,
			Flags: []cli.Flag{
				keyFlag,
			},
		},
		cli.Command{
			Name:      "help",
			Usage:     "Shows all commands or help for one command",
			ArgsUsage: "[command]",
			Action: func(c *cli.Context) {
				args := c.Args()
				if args.Present() {
					cli.ShowCommandHelp(c, args.First())
					return
				}
				cli.ShowAppHelp(c)
			},
		},
		cli.Command{
			Name:  "version",
			Usage: "Print client version",
			Action: func(c *cli.Context) {
				fmt.Println(app.Version)
			},
		},
	}
	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func getAddr(c *cli.Context) string {
	host := c.GlobalString("host")
	port := c.GlobalInt("port")
	return fmt.Sprintf("http://%s:%d", host, port)
}

func cmdList(c *cli.Context) {
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

func cmdAdd(c *cli.Context) {
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

func cmdRemove(c *cli.Context) {
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

func simpleCommand(c *cli.Context, command string) {
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
	res, err := http.Get(addr + "/" + key + "/" + command)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	_, err = io.Copy(os.Stdout, res.Body)
	if err != nil {
		log.Fatal(err)
	}
}

func cmdStart(c *cli.Context) {
	simpleCommand(c, "start")
}

func cmdStop(c *cli.Context) {
	simpleCommand(c, "stop")
}

func cmdPause(c *cli.Context) {
	simpleCommand(c, "pause")
}

func cmdContinue(c *cli.Context) {
	simpleCommand(c, "continue")
}
