package output

import (
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestFormatEnv(t *testing.T) {
	values := map[string]interface{}{
		"SIMPLE":  "value",
		"SPACES":  "value with spaces",
		"SPECIAL": "value$with\"special",
		"NESTED": map[string]interface{}{
			"key": "value",
		},
	}

	result := formatEnv(values)

	// Check that all keys are present
	for key := range values {
		if !strings.Contains(result, key+"=") {
			t.Errorf("Output missing key: %s", key)
		}
	}

	// Check that values with spaces are quoted
	if !strings.Contains(result, `"value with spaces"`) {
		t.Error("Values with spaces should be quoted")
	}

	// Check that nested values are JSON-encoded
	if !strings.Contains(result, "NESTED=") {
		t.Error("Nested values should be present")
	}
}

func TestFormatJSON(t *testing.T) {
	values := map[string]interface{}{
		"KEY1": "value1",
		"KEY2": 42,
		"KEY3": map[string]interface{}{
			"nested": "value",
		},
	}

	result, err := formatJSON(values)
	if err != nil {
		t.Fatalf("formatJSON failed: %v", err)
	}

	// Parse back to verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	if parsed["KEY1"] != "value1" {
		t.Errorf("KEY1: expected value1, got %v", parsed["KEY1"])
	}
}

func TestFormatYAML(t *testing.T) {
	values := map[string]interface{}{
		"KEY1": "value1",
		"KEY2": 42,
		"KEY3": map[string]interface{}{
			"nested": "value",
		},
	}

	result, err := formatYAML(values)
	if err != nil {
		t.Fatalf("formatYAML failed: %v", err)
	}

	// Parse back to verify it's valid YAML
	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("Result is not valid YAML: %v", err)
	}

	if parsed["KEY1"] != "value1" {
		t.Errorf("KEY1: expected value1, got %v", parsed["KEY1"])
	}
}

func TestFormatK8s(t *testing.T) {
	values := map[string]interface{}{
		"KEY1": "value1",
		"KEY2": "value2",
	}

	tests := []struct {
		name       string
		secretName string
		base64     bool
	}{
		{"plain text", "my-secret", false},
		{"base64", "my-secret-b64", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatK8s(values, tt.secretName, tt.base64)
			if err != nil {
				t.Fatalf("formatK8s failed: %v", err)
			}

			// Parse as YAML to verify structure
			var parsed map[string]interface{}
			if err := yaml.Unmarshal([]byte(result), &parsed); err != nil {
				t.Fatalf("Result is not valid YAML: %v", err)
			}

			// Check required fields
			if parsed["apiVersion"] != "v1" {
				t.Error("apiVersion should be v1")
			}
			if parsed["kind"] != "Secret" {
				t.Error("kind should be Secret")
			}

			metadata := parsed["metadata"].(map[string]interface{})
			if metadata["name"] != tt.secretName {
				t.Errorf("Secret name: expected %s, got %v", tt.secretName, metadata["name"])
			}

			// Check data/stringData field
			if tt.base64 {
				if _, ok := parsed["data"]; !ok {
					t.Error("base64 secrets should have 'data' field")
				}
			} else {
				if _, ok := parsed["stringData"]; !ok {
					t.Error("plain text secrets should have 'stringData' field")
				}
			}
		})
	}
}

func TestFormatOutput(t *testing.T) {
	values := map[string]interface{}{
		"KEY": "value",
	}

	tests := []struct {
		name      string
		format    Format
		opts      FormatOptions
		expectErr bool
	}{
		{"env format", FormatEnv, FormatOptions{Format: FormatEnv}, false},
		{"json format", FormatJSON, FormatOptions{Format: FormatJSON}, false},
		{"yaml format", FormatYAML, FormatOptions{Format: FormatYAML}, false},
		{"k8s with secret name", FormatK8s, FormatOptions{Format: FormatK8s, SecretName: "test"}, false},
		{"k8s without secret name", FormatK8s, FormatOptions{Format: FormatK8s}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FormatOutput(values, tt.opts)
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestNeedsQuoting(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"simple", false},
		{"with space", true},
		{"with\ttab", true},
		{"with\nnewline", true},
		{"with\"quote", true},
		{"with'quote", true},
		{"with$dollar", true},
		{"with\\backslash", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := needsQuoting(tt.input)
			if result != tt.expected {
				t.Errorf("needsQuoting(%q): expected %v, got %v", tt.input, tt.expected, result)
			}
		})
	}
}
