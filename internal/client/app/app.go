package app

import (
	"fmt"
	"sort"

	"github.com/urfave/cli"
	"github.com/wybiral/hades/internal/client/api"
)

// NewApp returns a new client application.
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
				cli.StringFlag{
					Name:  "key, k",
					Usage: "Daemon key value (optional)",
				},
				cli.StringFlag{
					Name:  "dir, d",
					Usage: "Directory for daemon (optional)",
				},
			},
		},
		cli.Command{
			Name:      "remove",
			ArgsUsage: "KEY",
			Usage:     "Remove daemon",
			Action:    api.Remove,
		},
		cli.Command{
			Name:      "start",
			ArgsUsage: "KEY",
			Usage:     "Start daemon",
			Action:    api.Start,
		},
		cli.Command{
			Name:      "stop",
			ArgsUsage: "KEY",
			Usage:     "Stop daemon",
			Action:    api.Stop,
		},
		cli.Command{
			Name:      "pause",
			ArgsUsage: "KEY",
			Usage:     "Pause daemon",
			Action:    api.Pause,
		},
		cli.Command{
			Name:      "continue",
			ArgsUsage: "KEY",
			Usage:     "Continue daemon",
			Action:    api.Continue,
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
