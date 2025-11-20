package commands

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/teamcurri/puff/pkg/keys"
	"github.com/urfave/cli/v2"
)

// KeysCommand creates the keys parent command for key management
func KeysCommand() *cli.Command {
	return &cli.Command{
		Name:  "keys",
		Usage: "Manage encryption keys",
		Subcommands: []*cli.Command{
			keysAddCommand(),
			keysRmCommand(),
			keysListCommand(),
		},
	}
}

func keysAddCommand() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "Add an age key and re-encrypt all files",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "key",
				Aliases:  []string{"k"},
				Usage:    "Age public key to add",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "comment",
				Aliases: []string{"c"},
				Usage:   "Comment for the key (e.g., 'Bob's laptop')",
			},
			&cli.StringFlag{
				Name:    "env",
				Aliases: []string{"e"},
				Usage:   "Only update files in specific environment",
			},
			&cli.StringFlag{
				Name:    "root",
				Aliases: []string{"r"},
				Usage:   "Root directory for config files",
				Value:   ".",
			},
		},
		Action: keysAddAction,
	}
}

func keysRmCommand() *cli.Command {
	return &cli.Command{
		Name:  "rm",
		Usage: "Remove an age key and re-encrypt all files",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "key",
				Aliases:  []string{"k"},
				Usage:    "Age public key to remove",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "env",
				Aliases: []string{"e"},
				Usage:   "Only update files in specific environment",
			},
			&cli.StringFlag{
				Name:    "root",
				Aliases: []string{"r"},
				Usage:   "Root directory for config files",
				Value:   ".",
			},
		},
		Action: keysRmAction,
	}
}

func keysListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all keys with their comments and environments",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "root",
				Aliases: []string{"r"},
				Usage:   "Root directory for config files",
				Value:   ".",
			},
		},
		Action: keysListAction,
	}
}

func keysAddAction(c *cli.Context) error {
	key := c.String("key")
	comment := c.String("comment")
	env := c.String("env")
	rootDir := c.String("root")

	color.Yellow("Adding key to encrypted files...")

	if err := keys.AddKey(rootDir, key, comment, env); err != nil {
		return fmt.Errorf("failed to add key: %w", err)
	}

	if env != "" {
		color.Green("Successfully added key to files in environment: %s", env)
	} else {
		color.Green("Successfully added key to all encrypted files")
	}

	if comment != "" {
		color.Cyan("Comment: %s", comment)
	}

	return nil
}

func keysRmAction(c *cli.Context) error {
	key := c.String("key")
	env := c.String("env")
	rootDir := c.String("root")

	color.Yellow("Removing key from encrypted files...")

	if err := keys.RemoveKey(rootDir, key, env); err != nil {
		return fmt.Errorf("failed to remove key: %w", err)
	}

	if env != "" {
		color.Green("Successfully removed key from files in environment: %s", env)
	} else {
		color.Green("Successfully removed key from all encrypted files")
	}

	return nil
}

func keysListAction(c *cli.Context) error {
	rootDir := c.String("root")

	keyList, err := keys.ListKeys(rootDir)
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	if len(keyList) == 0 {
		color.Yellow("No encryption keys found")
		return nil
	}

	color.Cyan("\nEncryption keys:")
	for i, keyInfo := range keyList {
		fmt.Printf("\n%d. %s\n", i+1, keyInfo.Key)
		if keyInfo.Comment != "" {
			fmt.Printf("   Comment: %s\n", keyInfo.Comment)
		}
		if len(keyInfo.Envs) > 0 {
			fmt.Printf("   Environments: %v\n", keyInfo.Envs)
		}
	}

	return nil
}
