package commands

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/teamcurri/puff/internal/config"
	"github.com/teamcurri/puff/internal/output"
	"github.com/teamcurri/puff/internal/templating"
	"github.com/urfave/cli/v2"
)

// GenerateCommand creates the generate command for generating full config
func GenerateCommand() *cli.Command {
	return &cli.Command{
		Name:  "generate",
		Usage: "Generate full config for specified app/env/target in specified format",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "app",
				Aliases:  []string{"a"},
				Usage:    "Application name",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "env",
				Aliases:  []string{"e"},
				Usage:    "Environment name",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "target",
				Aliases: []string{"t"},
				Usage:   "Target platform (optional)",
			},
			&cli.StringFlag{
				Name:     "format",
				Aliases:  []string{"f"},
				Usage:    "Output format (env, json, yaml, k8s)",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output file (defaults to stdout)",
			},
			&cli.StringFlag{
				Name:  "secret-name",
				Usage: "Kubernetes secret name (required for k8s format)",
			},
			&cli.BoolFlag{
				Name:  "base64",
				Usage: "Base64 encode values for k8s secrets",
				Value: false,
			},
			&cli.StringFlag{
				Name:    "root",
				Aliases: []string{"r"},
				Usage:   "Root directory for config files",
				Value:   ".",
			},
		},
		Action: generateAction,
	}
}

func generateAction(c *cli.Context) error {
	// Get parameters
	app := c.String("app")
	env := c.String("env")
	target := c.String("target")
	formatStr := c.String("format")
	outputFile := c.String("output")
	secretName := c.String("secret-name")
	base64 := c.Bool("base64")
	rootDir := c.String("root")

	// Validate format
	var format output.Format
	switch formatStr {
	case "env":
		format = output.FormatEnv
	case "json":
		format = output.FormatJSON
	case "yaml":
		format = output.FormatYAML
	case "k8s":
		format = output.FormatK8s
		if secretName == "" {
			return fmt.Errorf("--secret-name is required for k8s format")
		}
	default:
		return fmt.Errorf("unknown format: %s (valid formats: env, json, yaml, k8s)", formatStr)
	}

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

	// Filter out underscore-prefixed variables
	exportValues := make(map[string]interface{})
	for key, value := range resolved {
		if len(key) > 0 && key[0] != '_' {
			exportValues[key] = value
		}
	}

	// Format output
	formatted, err := output.FormatOutput(exportValues, output.FormatOptions{
		Format:     format,
		SecretName: secretName,
		Base64:     base64,
	})
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	// Write output
	// NOTE: Output files are intentionally UNENCRYPTED as they are deployment
	// configurations consumed by runtime systems (Docker, Kubernetes, etc.).
	// These files contain the final, resolved configuration values after decryption
	// and template processing. Encryption at this stage would prevent deployment
	// systems from reading the configuration.
	//
	// Security Considerations:
	// - Source config files remain encrypted at rest
	// - This output is for deployment environments only
	// - Handle output files according to your deployment security practices
	// - For Kubernetes: pipe to kubectl, don't save to disk
	// - For Docker: use docker secrets or environment injection
	// - For sensitive data: use runtime encryption (Vault, AWS Secrets Manager, etc.)
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(formatted), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		color.Green("Config generated and written to %s", outputFile)
	} else {
		fmt.Println(formatted)
	}

	return nil
}
