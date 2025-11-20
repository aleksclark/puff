package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/teamcurri/puff/internal/keys"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// EncryptCommand creates the encrypt command for re-encrypting bulk-edited files
func EncryptCommand() *cli.Command {
	return &cli.Command{
		Name:  "encrypt",
		Usage: "Encrypt a decrypted file (removes .dec file after encryption)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "file",
				Aliases:  []string{"f"},
				Usage:    "Decrypted file to encrypt (must have .dec extension)",
				Required: true,
			},
		},
		Action: encryptAction,
	}
}

func encryptAction(c *cli.Context) error {
	decFilePath := c.String("file")

	// Check if file exists
	if _, err := os.Stat(decFilePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", decFilePath)
	}

	// Ensure file has .dec extension
	if !strings.Contains(decFilePath, ".dec.") && !strings.HasSuffix(decFilePath, ".dec") {
		return fmt.Errorf("file must be a decrypted file with .dec extension: %s", decFilePath)
	}

	// Determine original encrypted file path
	var encFilePath string
	if strings.HasSuffix(decFilePath, ".dec.yml") {
		encFilePath = strings.TrimSuffix(decFilePath, ".dec.yml") + ".yml"
	} else if strings.HasSuffix(decFilePath, ".dec.yaml") {
		encFilePath = strings.TrimSuffix(decFilePath, ".dec.yaml") + ".yaml"
	} else if strings.HasSuffix(decFilePath, ".dec") {
		encFilePath = strings.TrimSuffix(decFilePath, ".dec")
	} else {
		return fmt.Errorf("unexpected file extension: %s", decFilePath)
	}

	// Get encryption keys from the original file if it exists, otherwise from directory
	var ageKeys []string
	if _, err := os.Stat(encFilePath); err == nil {
		// Original file exists, extract its keys
		data, err := os.ReadFile(encFilePath)
		if err != nil {
			return fmt.Errorf("failed to read original file: %w", err)
		}

		var yamlData map[string]interface{}
		if err := yaml.Unmarshal(data, &yamlData); err != nil {
			return fmt.Errorf("failed to parse original file: %w", err)
		}

		ageKeys = keys.ExtractAgeKeys(yamlData)
	}

	// If no keys found from original file, get from directory
	if len(ageKeys) == 0 {
		rootDir := filepath.Dir(filepath.Dir(encFilePath)) // Go up to root
		if rootDir == "." {
			rootDir, _ = os.Getwd()
		}

		var err error
		ageKeys, err = getDirectoryEncryptionKeys(rootDir)
		if err != nil {
			return fmt.Errorf("failed to get encryption keys: %w", err)
		}

		if len(ageKeys) == 0 {
			return fmt.Errorf("no encryption keys found - cannot encrypt file")
		}
	}

	// Read decrypted content
	decData, err := os.ReadFile(decFilePath)
	if err != nil {
		return fmt.Errorf("failed to read decrypted file: %w", err)
	}

	// Validate it's valid YAML
	var testYaml map[string]interface{}
	if err := yaml.Unmarshal(decData, &testYaml); err != nil {
		return fmt.Errorf("decrypted file is not valid YAML: %w", err)
	}

	// Write to original location (will be encrypted)
	if err := os.WriteFile(encFilePath, decData, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Encrypt the file
	if err := keys.EncryptFile(encFilePath, ageKeys); err != nil {
		return fmt.Errorf("failed to encrypt file: %w", err)
	}

	// Remove the .dec file - critical for security
	if err := os.Remove(decFilePath); err != nil {
		color.Red("Error: failed to remove decrypted file %s: %v", decFilePath, err)
		color.Red("Please manually delete this file to prevent security risk!")
		return fmt.Errorf("cleanup failed: %w", err)
	}

	color.Green("Encrypted %s to %s", decFilePath, encFilePath)
	color.Green("Removed temporary decrypted file")

	return nil
}
