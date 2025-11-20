package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/getsops/sops/v3/decrypt"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// DecryptCommand creates the decrypt command for decrypting files for bulk edits
func DecryptCommand() *cli.Command {
	return &cli.Command{
		Name:  "decrypt",
		Usage: "Decrypt a file for bulk editing (creates .dec file)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "file",
				Aliases:  []string{"f"},
				Usage:    "File to decrypt",
				Required: true,
			},
		},
		Action: decryptAction,
	}
}

func decryptAction(c *cli.Context) error {
	filePath := c.String("file")

	// Validate and normalize path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", absPath)
	}

	// Check if file is encrypted
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Properly detect SOPS encryption by parsing YAML
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return fmt.Errorf("file is not valid YAML: %w", err)
	}

	if _, hasSops := yamlData["sops"]; !hasSops {
		return fmt.Errorf("file is not SOPS-encrypted: %s", absPath)
	}

	// Decrypt the file
	decrypted, err := decrypt.File(absPath, "yaml")
	if err != nil {
		return fmt.Errorf("failed to decrypt file: %w", err)
	}

	// Determine output file path (SOPS standard: .dec extension)
	outputPath := absPath + ".dec"
	if strings.HasSuffix(absPath, ".yml") {
		outputPath = strings.TrimSuffix(absPath, ".yml") + ".dec.yml"
	} else if strings.HasSuffix(absPath, ".yaml") {
		outputPath = strings.TrimSuffix(absPath, ".yaml") + ".dec.yaml"
	}

	// Write decrypted content
	if err := os.WriteFile(outputPath, decrypted, 0600); err != nil {
		return fmt.Errorf("failed to write decrypted file: %w", err)
	}

	color.Green("Decrypted %s to %s", absPath, outputPath)
	color.Yellow("\nEdit the decrypted file, then run:")
	color.Cyan("  puff encrypt -f %s", outputPath)

	return nil
}
