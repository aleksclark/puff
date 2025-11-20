package commands

import (
	"fmt"

	"github.com/teamcurri/puff/internal/config"
	"github.com/teamcurri/puff/internal/templating"
	"github.com/urfave/cli/v2"
)

// GetCommand creates the get command for retrieving config values
func GetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a config value for specified app/env/target",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "key",
				Aliases:  []string{"k"},
				Usage:    "Key to retrieve",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "app",
				Aliases: []string{"a"},
				Usage:   "Application name",
			},
			&cli.StringFlag{
				Name:    "env",
				Aliases: []string{"e"},
				Usage:   "Environment name",
			},
			&cli.StringFlag{
				Name:    "target",
				Aliases: []string{"t"},
				Usage:   "Target platform",
			},
			&cli.StringFlag{
				Name:    "root",
				Aliases: []string{"r"},
				Usage:   "Root directory for config files",
				Value:   ".",
			},
		},
		Action: getAction,
	}
}

func getAction(c *cli.Context) error {
	key := c.String("key")
	app := c.String("app")
	env := c.String("env")
	target := c.String("target")
	rootDir := c.String("root")

	// Load configuration
	cfg, err := config.Load(config.LoadContext{
		RootDir: rootDir,
		App:     app,
		Env:     env,
		Target:  target,
	})
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Resolve template variables
	resolver := templating.NewResolver(cfg.Values)
	resolved, err := resolver.Resolve()
	if err != nil {
		return fmt.Errorf("failed to resolve templates: %w", err)
	}

	// Get the value
	value, exists := resolved[key]
	if !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	// Print the value
	fmt.Printf("%v\n", value)

	return nil
}
