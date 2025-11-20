package main

import (
	"os"

	"github.com/fatih/color"
	"github.com/teamcurri/puff/pkg/commands"
	"github.com/urfave/cli/v2"
)

var version = "dev"

func main() {
	app := &cli.App{
		Name:    "puff",
		Usage:   "GitOps secret and environment variable management tool",
		Version: version,
		Commands: []*cli.Command{
			commands.InitCommand(),
			commands.KeysCommand(),
			commands.GetCommand(),
			commands.SetCommand(),
			commands.GenerateCommand(),
			commands.DecryptCommand(),
			commands.EncryptCommand(),
		},
		Before: func(c *cli.Context) error {
			// Set up color output
			color.NoColor = false
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		color.Red("Error: %v", err)
		os.Exit(1)
	}
}
