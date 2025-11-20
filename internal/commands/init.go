package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/teamcurri/puff/internal/keys"
	"github.com/urfave/cli/v2"
)

// InitCommand creates the init command for setting up the config directory
func InitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize a new puff configuration directory with encryption",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "dir",
				Aliases: []string{"d"},
				Usage:   "Directory to initialize (defaults to current directory)",
				Value:   ".",
			},
			&cli.StringFlag{
				Name:     "age-keys",
				Aliases:  []string{"k"},
				Usage:    "Comma-separated list of age public keys for encryption (required)",
				Required: true,
			},
		},
		Action: initAction,
	}
}

func initAction(c *cli.Context) error {
	dir := c.String("dir")
	ageKeysStr := c.String("age-keys")

	// Parse age keys
	ageKeys := []string{}
	for _, key := range strings.Split(ageKeysStr, ",") {
		trimmed := strings.TrimSpace(key)
		if trimmed != "" {
			ageKeys = append(ageKeys, trimmed)
		}
	}

	if len(ageKeys) == 0 {
		return fmt.Errorf("at least one age public key is required for encryption")
	}

	// Validate age keys format
	for _, key := range ageKeys {
		if !strings.HasPrefix(key, "age1") {
			return fmt.Errorf("invalid age key format: %s (must start with 'age1')", key)
		}
	}

	// Create base directory structure
	dirs := []string{
		filepath.Join(dir, "base"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0700); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}

	// Create base/shared.yml with example content
	sharedYml := filepath.Join(dir, "base", "shared.yml")
	if _, err := os.Stat(sharedYml); os.IsNotExist(err) {
		// Create a valid YAML file with at least one key-value pair
		content := `# Global shared configuration
# Add your variables below
_PUFF_INITIALIZED: "true"
`
		// Write plain YAML first
		if err := os.WriteFile(sharedYml, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to create %s: %w", sharedYml, err)
		}

		// Encrypt the file
		if err := keys.EncryptFile(sharedYml, ageKeys); err != nil {
			return fmt.Errorf("failed to encrypt %s: %w", sharedYml, err)
		}

		color.Green("Created %s (encrypted)", sharedYml)
	}

	// Create .sops.yaml with the provided age keys
	sopsYml := filepath.Join(dir, ".sops.yaml")
	if _, err := os.Stat(sopsYml); os.IsNotExist(err) {
		// Build age keys list for SOPS config
		ageKeysList := strings.Join(ageKeys, ",\n      ")
		content := fmt.Sprintf(`# SOPS configuration for Puff
# This file was automatically generated during init
creation_rules:
  - path_regex: .*\.yml$
    age: >-
      %s
`, ageKeysList)
		// Write with restricted permissions (0600) as this contains encryption configuration
		if err := os.WriteFile(sopsYml, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to create %s: %w", sopsYml, err)
		}
		color.Green("Created %s", sopsYml)
	}

	color.Green("\nPuff configuration directory initialized with encryption!")
	color.Cyan("\nAll configuration files will be encrypted with the provided age keys.")
	color.Cyan("\nNext steps:")
	color.Cyan("1. Create environment directories: mkdir %s/dev %s/prod", dir, dir)
	color.Cyan("2. Add configuration: puff set -k KEY -v VALUE -r %s", dir)
	color.Cyan("3. For bulk edits: puff decrypt <file> (edit) puff encrypt <file>")

	return nil
}
