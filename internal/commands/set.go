package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/getsops/sops/v3/decrypt"
	"github.com/teamcurri/puff/internal/keys"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// SetCommand creates the set command for setting config values
func SetCommand() *cli.Command {
	return &cli.Command{
		Name:  "set",
		Usage: "Set a config value for specified app/env/target",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "key",
				Aliases:  []string{"k"},
				Usage:    "Key to set",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "value",
				Aliases:  []string{"v"},
				Usage:    "Value to set",
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
		Action: setAction,
	}
}

func setAction(c *cli.Context) error {
	key := c.String("key")
	value := c.String("value")
	app := c.String("app")
	env := c.String("env")
	target := c.String("target")
	rootDir := c.String("root")

	// Get encryption keys from the directory - ALWAYS required
	directoryAgeKeys, err := getDirectoryEncryptionKeys(rootDir)
	if err != nil {
		return fmt.Errorf("failed to check directory encryption: %w", err)
	}
	if len(directoryAgeKeys) == 0 {
		return fmt.Errorf("no encryption keys found in directory - run 'puff init' first to initialize with encryption keys")
	}

	// Determine which file to update based on the flags
	var filePath string

	if target != "" {
		// Target-specific config: target-overrides/{target}/{env}/{app}.yml
		// Env is optional for targets - defaults to "base" if not specified
		targetEnv := env
		if targetEnv == "" {
			targetEnv = "base"
		}

		targetDir := filepath.Join(rootDir, "target-overrides", target, targetEnv)
		if err := os.MkdirAll(targetDir, 0700); err != nil {
			return fmt.Errorf("failed to create target directory: %w", err)
		}

		if app != "" {
			filePath = filepath.Join(targetDir, fmt.Sprintf("%s.yml", app))
		} else {
			filePath = filepath.Join(targetDir, "shared.yml")
		}
	} else if env != "" {
		// Environment-specific config
		envDir := filepath.Join(rootDir, env)
		if err := os.MkdirAll(envDir, 0700); err != nil {
			return fmt.Errorf("failed to create environment directory: %w", err)
		}

		if app != "" {
			filePath = filepath.Join(envDir, fmt.Sprintf("%s.yml", app))
		} else {
			filePath = filepath.Join(envDir, "shared.yml")
		}
	} else if app != "" {
		// Base app-specific config
		filePath = filepath.Join(rootDir, "base", fmt.Sprintf("%s.yml", app))
	} else {
		// Base shared config
		filePath = filepath.Join(rootDir, "base", "shared.yml")
	}

	// Load existing config or create new one
	var config map[string]interface{}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			config = make(map[string]interface{})
		} else {
			return fmt.Errorf("failed to read file: %w", err)
		}
	} else {
		// Check if file is SOPS-encrypted
		var checkMap map[string]interface{}
		if err := yaml.Unmarshal(data, &checkMap); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}
		if _, hasSops := checkMap["sops"]; hasSops {
			// Decrypt the file
			decryptedData, err := decrypt.File(filePath, "yaml")
			if err != nil {
				return fmt.Errorf("failed to decrypt file: %w", err)
			}
			data = decryptedData
		}

		// Parse the (possibly decrypted) YAML
		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}
		if config == nil {
			config = make(map[string]interface{})
		}
		// Remove SOPS metadata if it exists
		delete(config, "sops")
	}

	// Set the value
	config[key] = value

	// Write back to file
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the plain YAML first
	if err := os.WriteFile(filePath, yamlData, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// ALWAYS encrypt - encryption is mandatory
	if err := keys.EncryptFile(filePath, directoryAgeKeys); err != nil {
		return fmt.Errorf("failed to encrypt file: %w", err)
	}

	color.Green("Set %s=%s in %s (encrypted)", key, value, filePath)

	return nil
}

// getDirectoryEncryptionKeys scans the directory for any encrypted files and returns their age keys
func getDirectoryEncryptionKeys(rootDir string) ([]string, error) {
	keySet := make(map[string]bool)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories, non-yml files, and .sops.yaml
		if info.IsDir() || filepath.Ext(path) != ".yml" || filepath.Base(path) == ".sops.yaml" {
			return nil
		}

		// Check if file is SOPS-encrypted
		data, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		var yamlData map[string]interface{}
		if err := yaml.Unmarshal(data, &yamlData); err != nil {
			return nil // Skip files that aren't valid YAML
		}

		// If this is a SOPS file, extract its keys
		if _, hasSops := yamlData["sops"]; hasSops {
			fileKeys := keys.ExtractAgeKeys(yamlData)
			for _, key := range fileKeys {
				keySet[key] = true
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert set to slice
	result := make([]string, 0, len(keySet))
	for key := range keySet {
		result = append(result, key)
	}

	return result, nil
}
