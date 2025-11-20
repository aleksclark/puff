package keys

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SOPSConfig represents the .sops.yaml configuration structure
type SOPSConfig struct {
	CreationRules []CreationRule    `yaml:"creation_rules"`
	KeyComments   map[string]string `yaml:"-"` // Not in YAML, but tracked for comments
}

// CreationRule represents a single creation rule in SOPS config
type CreationRule struct {
	PathRegex string `yaml:"path_regex"`
	Age       string `yaml:"age"`
}

// LoadSOPSConfig loads and parses the .sops.yaml file
func LoadSOPSConfig(rootDir string) (*SOPSConfig, error) {
	sopsPath := filepath.Join(rootDir, ".sops.yaml")

	data, err := os.ReadFile(sopsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read .sops.yaml: %w", err)
	}

	// Parse key comments from anywhere in the file
	keyComments := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Parse comment format: # age1... (Comment Text)
		if strings.HasPrefix(trimmed, "# age1") {
			content := strings.TrimPrefix(trimmed, "# ")
			if idx := strings.Index(content, " ("); idx > 0 {
				key := content[:idx]
				comment := strings.TrimSuffix(content[idx+2:], ")")
				keyComments[key] = comment
			}
		}
	}

	// Parse YAML structure
	var config SOPSConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse .sops.yaml: %w", err)
	}

	config.KeyComments = keyComments
	return &config, nil
}

// SaveSOPSConfig writes the SOPS config back to .sops.yaml
func SaveSOPSConfig(rootDir string, config *SOPSConfig) error {
	sopsPath := filepath.Join(rootDir, ".sops.yaml")

	var output strings.Builder

	// Write commented keys at the top
	output.WriteString("# SOPS configuration for Puff\n")
	output.WriteString("# Age encryption keys with their associated comments\n")

	// Get all keys from the first creation rule
	keys := getKeysFromConfig(config)
	for _, key := range keys {
		comment := config.KeyComments[key]
		if comment == "" {
			comment = "No comment"
		}
		output.WriteString(fmt.Sprintf("# %s (%s)\n", key, comment))
	}

	output.WriteString("\n")

	// Write YAML structure
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal SOPS config: %w", err)
	}

	output.Write(yamlData)

	// Write with restricted permissions
	if err := os.WriteFile(sopsPath, []byte(output.String()), 0600); err != nil {
		return fmt.Errorf("failed to write .sops.yaml: %w", err)
	}

	return nil
}

// AddKeyToSOPSConfig adds an age key to the SOPS configuration
func AddKeyToSOPSConfig(rootDir, ageKey, comment string) error {
	config, err := LoadSOPSConfig(rootDir)
	if err != nil {
		return fmt.Errorf("failed to load SOPS config: %w", err)
	}

	// Get existing keys
	keys := getKeysFromConfig(config)

	// Check if key already exists
	for _, k := range keys {
		if k == ageKey {
			// Update comment if provided
			if comment != "" {
				config.KeyComments[ageKey] = comment
			}
			return SaveSOPSConfig(rootDir, config)
		}
	}

	// Add new key
	keys = append(keys, ageKey)
	if comment != "" {
		config.KeyComments[ageKey] = comment
	}

	// Update the first creation rule
	if len(config.CreationRules) > 0 {
		config.CreationRules[0].Age = formatAgeKeys(keys)
	}

	return SaveSOPSConfig(rootDir, config)
}

// RemoveKeyFromSOPSConfig removes an age key from the SOPS configuration
func RemoveKeyFromSOPSConfig(rootDir, ageKey string) error {
	config, err := LoadSOPSConfig(rootDir)
	if err != nil {
		return fmt.Errorf("failed to load SOPS config: %w", err)
	}

	// Get existing keys
	keys := getKeysFromConfig(config)

	// Remove the key
	newKeys := []string{}
	found := false
	for _, k := range keys {
		if k != ageKey {
			newKeys = append(newKeys, k)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("key not found in .sops.yaml: %s", ageKey)
	}

	// Remove from comments
	delete(config.KeyComments, ageKey)

	// Update the first creation rule
	if len(config.CreationRules) > 0 {
		config.CreationRules[0].Age = formatAgeKeys(newKeys)
	}

	return SaveSOPSConfig(rootDir, config)
}

// getKeysFromConfig extracts age keys from the SOPS config
func getKeysFromConfig(config *SOPSConfig) []string {
	if len(config.CreationRules) == 0 {
		return []string{}
	}

	ageStr := config.CreationRules[0].Age
	keys := []string{}

	// Parse comma-separated or newline-separated keys
	for _, part := range strings.Split(ageStr, ",") {
		for _, line := range strings.Split(part, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && strings.HasPrefix(trimmed, "age1") {
				keys = append(keys, trimmed)
			}
		}
	}

	return keys
}

// formatAgeKeys formats age keys for the YAML age field
func formatAgeKeys(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	if len(keys) == 1 {
		return keys[0]
	}
	return strings.Join(keys, ",\n      ")
}
