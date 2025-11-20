package output

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Format represents the output format type
type Format string

const (
	FormatEnv  Format = "env"
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
	FormatK8s  Format = "k8s"
)

// FormatOptions holds options for output formatting
type FormatOptions struct {
	Format     Format
	SecretName string // For k8s format
	Base64     bool   // For k8s format
}

// FormatOutput formats the given config values according to the specified format
func FormatOutput(values map[string]interface{}, opts FormatOptions) (string, error) {
	switch opts.Format {
	case FormatEnv:
		return formatEnv(values), nil
	case FormatJSON:
		return formatJSON(values)
	case FormatYAML:
		return formatYAML(values)
	case FormatK8s:
		if opts.SecretName == "" {
			return "", fmt.Errorf("secret-name is required for k8s format")
		}
		return formatK8s(values, opts.SecretName, opts.Base64)
	default:
		return "", fmt.Errorf("unknown format: %s", opts.Format)
	}
}

// formatEnv formats values as a .env file
// Nested values are converted to JSON
func formatEnv(values map[string]interface{}) string {
	var lines []string

	// Sort keys for consistent output
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := values[key]

		// Convert value to string
		var valueStr string
		switch v := value.(type) {
		case string:
			valueStr = v
		case map[string]interface{}, []interface{}:
			// Nested structures: convert to JSON
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				valueStr = fmt.Sprintf("%v", v)
			} else {
				valueStr = string(jsonBytes)
			}
		default:
			valueStr = fmt.Sprintf("%v", v)
		}

		// Quote value if it contains special characters or spaces
		if needsQuoting(valueStr) {
			valueStr = quoteValue(valueStr)
		}

		lines = append(lines, fmt.Sprintf("%s=%s", key, valueStr))
	}

	return strings.Join(lines, "\n")
}

// needsQuoting determines if a value needs to be quoted in .env format
func needsQuoting(value string) bool {
	// Quote if contains spaces, quotes, or special characters
	return strings.ContainsAny(value, " \t\n\r\"'$\\")
}

// quoteValue quotes a value for .env format
func quoteValue(value string) string {
	// Escape existing quotes and backslashes
	escaped := strings.ReplaceAll(value, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	return fmt.Sprintf("\"%s\"", escaped)
}

// formatJSON formats values as JSON
func formatJSON(values map[string]interface{}) (string, error) {
	jsonBytes, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(jsonBytes), nil
}

// formatYAML formats values as YAML
func formatYAML(values map[string]interface{}) (string, error) {
	yamlBytes, err := yaml.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return string(yamlBytes), nil
}

// formatK8s formats values as a Kubernetes secret
func formatK8s(values map[string]interface{}, secretName string, encodeBase64 bool) (string, error) {
	// Build the secret data
	data := make(map[string]interface{})

	// Sort keys for consistent output
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := values[key]

		// Convert value to string
		var valueStr string
		switch v := value.(type) {
		case string:
			valueStr = v
		case map[string]interface{}, []interface{}:
			// Nested structures: convert to JSON
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				valueStr = fmt.Sprintf("%v", v)
			} else {
				valueStr = string(jsonBytes)
			}
		default:
			valueStr = fmt.Sprintf("%v", v)
		}

		// Base64 encode if requested
		if encodeBase64 {
			valueStr = base64.StdEncoding.EncodeToString([]byte(valueStr))
		}

		data[key] = valueStr
	}

	// Build the Kubernetes secret structure
	secret := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"type":       "Opaque",
		"metadata": map[string]interface{}{
			"name": secretName,
		},
	}

	if encodeBase64 {
		secret["data"] = data
	} else {
		secret["stringData"] = data
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(secret)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Kubernetes secret: %w", err)
	}

	return string(yamlBytes), nil
}
