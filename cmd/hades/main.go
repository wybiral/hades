package main

import (
	"encoding/json"
	"fmt"
	"github.com/urfave/cli"
	"github.com/wybiral/hades/pkg/types"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
)

const version = "0.0.1"

func main() {
	app := cli.NewApp()
	app.Version = version
	app.Name = "hades"
	app.Usage = "remote cross-platform daemon manager"
	cli.HelpFlag = cli.StringFlag{Hidden: true}
	cli.VersionFlag = cli.StringFlag{Hidden: true}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "host",
			Value: "127.0.0.1",
			Usage: "Server host",
		},
		cli.IntFlag{
			Name:  "port",
			Value: 8666,
			Usage: "Server port",
		},
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
			ArgsUsage: "[key] <cmd> <dir>",
			Usage:     "Add daemon",
			Action:    cmdAdd,
		},
		cli.Command{
			Name:      "remove",
			ArgsUsage: "<key>",
			Usage:     "Remove daemon",
			Action:    cmdRemove,
		},
		cli.Command{
			Name:      "start",
			ArgsUsage: "<key>",
			Usage:     "Start daemon",
			Action:    cmdStart,
		},
		cli.Command{
			Name:      "stop",
			ArgsUsage: "<key>",
			Usage:     "Stop daemon",
			Action:    cmdStop,
		},
		cli.Command{
			Name:      "pause",
			ArgsUsage: "<key>",
			Usage:     "Pause daemon",
			Action:    cmdPause,
		},
		cli.Command{
			Name:      "continue",
			ArgsUsage: "<key>",
			Usage:     "Continue daemon",
			Action:    cmdContinue,
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

func printDaemon(daemon *types.Daemon) {
	fmt.Println(daemon.Key + " (" + daemon.Status + ")")
	fmt.Println("  Cmd: " + daemon.Cmd)
	fmt.Println("  Dir: " + daemon.Dir)
	fmt.Println("")
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
	dec := json.NewDecoder(res.Body)
	var daemons []*types.Daemon
	err = dec.Decode(&daemons)
	if err != nil {
		log.Fatal(err)
	}
	for _, daemon := range daemons {
		printDaemon(daemon)
	}
}

func cmdAdd(c *cli.Context) {
	args := c.Args()
	addr := getAddr(c)
	var key, cmd, dir string
	if len(args) == 3 {
		key = args[0]
		cmd = args[1]
		dir = args[2]
	} else if len(args) == 2 {
		cmd = args[0]
		dir = args[1]
	} else {
		cli.ShowCommandHelp(c, "add")
		return
	}
	res, err := http.PostForm(
		addr,
		url.Values{
			"key": {key},
			"cmd": {cmd},
			"dir": {dir},
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	dec := json.NewDecoder(res.Body)
	daemon := &types.Daemon{}
	err = dec.Decode(&daemon)
	if err != nil {
		log.Fatal(err)
	}
	printDaemon(daemon)
}

func cmdRemove(c *cli.Context) {
	args := c.Args()
	addr := getAddr(c)
	if len(args) != 1 {
		cli.ShowCommandHelp(c, "remove")
		return
	}
	key := args[0]
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", addr + "/" + key, nil)
	if err != nil {
		log.Fatal(err)
	}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	fmt.Println(key, "removed\n")
}

func simpleCommand(c *cli.Context, command string) {
	args := c.Args()
	addr := getAddr(c)
	if len(args) != 1 {
		cli.ShowCommandHelp(c, command)
		return
	}
	key := args[0]
	res, err := http.Get(addr + "/" + key + "/" + command)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	dec := json.NewDecoder(res.Body)
	daemon := &types.Daemon{}
	err = dec.Decode(&daemon)
	if err != nil {
		log.Fatal(err)
	}
	printDaemon(daemon)
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
