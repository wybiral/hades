package app

import (
	"fmt"
	"github.com/urfave/cli"
	"github.com/wybiral/hades/internal/client/api"
	"sort"
)

func NewApp() *cli.App {
	a := cli.NewApp()
	cli.HelpFlag = cli.StringFlag{Hidden: true}
	cli.VersionFlag = cli.StringFlag{Hidden: true}
	a.Flags = []cli.Flag{
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
	installCommands(a)
	sort.Sort(cli.FlagsByName(a.Flags))
	sort.Sort(cli.CommandsByName(a.Commands))
	return a
}

func installCommands(a *cli.App) {
	keyFlag := cli.StringFlag{
		Name:  "key, k",
		Usage: "Daemon key value",
	}
	a.Commands = []cli.Command{
		cli.Command{
			Name:      "list",
			ArgsUsage: " ",
			Usage:     "List all daemons",
			Action:    api.List,
		},
		cli.Command{
			Name:      "add",
			ArgsUsage: "COMMAND",
			Usage:     "Add daemon",
			Action:    api.Add,
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
			Action:    api.Remove,
			Flags: []cli.Flag{
				keyFlag,
			},
		},
		cli.Command{
			Name:      "start",
			ArgsUsage: " ",
			Usage:     "Start daemon",
			Action:    api.Start,
			Flags: []cli.Flag{
				keyFlag,
			},
		},
		cli.Command{
			Name:      "stop",
			ArgsUsage: " ",
			Usage:     "Stop daemon",
			Action:    api.Stop,
			Flags: []cli.Flag{
				keyFlag,
			},
		},
		cli.Command{
			Name:      "pause",
			ArgsUsage: " ",
			Usage:     "Pause daemon",
			Action:    api.Pause,
			Flags: []cli.Flag{
				keyFlag,
			},
		},
		cli.Command{
			Name:      "continue",
			ArgsUsage: " ",
			Usage:     "Continue daemon",
			Action:    api.Continue,
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
				fmt.Println(a.Version)
			},
		},
	}
}