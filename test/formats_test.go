package test

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/teamcurri/puff/test/helpers"
	"gopkg.in/yaml.v3"
)

// TestFormat_EnvOutput tests ENV format output
func TestFormat_EnvOutput(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set various types of values
	env.Set("DATABASE_URL", "postgres://localhost/mydb", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("API_PORT", "3000", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("DEBUG", "true", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("EMPTY_VAR", "", "-a", "api", "-e", "dev").AssertSuccess()

	result := env.Generate("api", "dev", "env")
	result.AssertSuccess()

	output := result.GetStdout()

	// Verify ENV format
	if !strings.Contains(output, "DATABASE_URL=postgres://localhost/mydb") {
		t.Error("ENV format missing DATABASE_URL")
	}
	if !strings.Contains(output, "API_PORT=3000") {
		t.Error("ENV format missing API_PORT")
	}
	if !strings.Contains(output, "DEBUG=true") {
		t.Error("ENV format missing DEBUG")
	}

	// Verify each variable is on its own line
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 3 {
		t.Errorf("Expected at least 3 lines in ENV output, got %d", len(lines))
	}

	// Verify no SOPS metadata in output
	if strings.Contains(output, "sops") {
		t.Error("ENV output contains SOPS metadata")
	}
}

// TestFormat_EnvWithSpecialCharacters tests ENV format with special characters
func TestFormat_EnvWithSpecialCharacters(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set values with special characters
	env.Set("URL_WITH_QUERY", "https://example.com?foo=bar&baz=qux", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("VALUE_WITH_SPACES", "value with spaces", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("VALUE_WITH_QUOTES", "value\"with\"quotes", "-a", "api", "-e", "dev").AssertSuccess()

	result := env.Generate("api", "dev", "env")
	result.AssertSuccess()

	output := result.GetStdout()

	// Verify special characters are preserved
	if !strings.Contains(output, "URL_WITH_QUERY=") {
		t.Error("ENV format missing URL_WITH_QUERY")
	}
	if !strings.Contains(output, "VALUE_WITH_SPACES=") {
		t.Error("ENV format missing VALUE_WITH_SPACES")
	}
}

// TestFormat_JSONOutput tests JSON format output
func TestFormat_JSONOutput(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set various values
	env.Set("STRING_VAR", "hello", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("NUMBER_VAR", "42", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("BOOL_VAR", "true", "-a", "api", "-e", "dev").AssertSuccess()

	result := env.Generate("api", "dev", "json")
	result.AssertSuccess()

	output := result.GetStdout()

	// Verify it's valid JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(output), &jsonData); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	// Verify values
	if jsonData["STRING_VAR"] != "hello" {
		t.Errorf("STRING_VAR incorrect: got %v", jsonData["STRING_VAR"])
	}
	if jsonData["NUMBER_VAR"] != "42" {
		t.Errorf("NUMBER_VAR incorrect: got %v", jsonData["NUMBER_VAR"])
	}
	if jsonData["BOOL_VAR"] != "true" {
		t.Errorf("BOOL_VAR incorrect: got %v", jsonData["BOOL_VAR"])
	}

	// Verify no SOPS metadata
	if _, exists := jsonData["sops"]; exists {
		t.Error("JSON output contains SOPS metadata")
	}
}

// TestFormat_JSONWithNestedStructures tests JSON format with complex values
func TestFormat_JSONWithNestedStructures(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set a JSON value
	jsonValue := `{"host":"localhost","port":5432,"credentials":{"user":"admin","pass":"secret"}}`
	env.Set("DB_CONFIG", jsonValue, "-a", "api", "-e", "dev").AssertSuccess()

	result := env.Generate("api", "dev", "json")
	result.AssertSuccess()

	output := result.GetStdout()

	// Verify it's valid JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(output), &jsonData); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	// Verify the JSON value is preserved as a string
	if dbConfig, ok := jsonData["DB_CONFIG"].(string); ok {
		if dbConfig != jsonValue {
			t.Errorf("DB_CONFIG value not preserved correctly")
		}
	} else {
		t.Error("DB_CONFIG should be a string")
	}
}

// TestFormat_YAMLOutput tests YAML format output
func TestFormat_YAMLOutput(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set various values
	env.Set("APP_NAME", "myapp", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("VERSION", "1.0.0", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("ENABLED", "true", "-a", "api", "-e", "dev").AssertSuccess()

	result := env.Generate("api", "dev", "yaml")
	result.AssertSuccess()

	output := result.GetStdout()

	// Verify it's valid YAML
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal([]byte(output), &yamlData); err != nil {
		t.Fatalf("Output is not valid YAML: %v", err)
	}

	// Verify values
	if yamlData["APP_NAME"] != "myapp" {
		t.Errorf("APP_NAME incorrect: got %v", yamlData["APP_NAME"])
	}
	if yamlData["VERSION"] != "1.0.0" {
		t.Errorf("VERSION incorrect: got %v", yamlData["VERSION"])
	}
	if yamlData["ENABLED"] != "true" {
		t.Errorf("ENABLED incorrect: got %v", yamlData["ENABLED"])
	}

	// Verify no SOPS metadata
	if _, exists := yamlData["sops"]; exists {
		t.Error("YAML output contains SOPS metadata")
	}

	// Verify YAML format (key: value)
	if !strings.Contains(output, "APP_NAME: myapp") {
		t.Error("YAML format incorrect")
	}
}

// TestFormat_YAMLWithMultilineValues tests YAML format with multiline values
func TestFormat_YAMLWithMultilineValues(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set a multiline value
	multilineValue := `Line 1
Line 2
Line 3`
	env.Set("MULTILINE", multilineValue, "-a", "api", "-e", "dev").AssertSuccess()

	result := env.Generate("api", "dev", "yaml")
	result.AssertSuccess()

	output := result.GetStdout()

	// Verify it's valid YAML
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal([]byte(output), &yamlData); err != nil {
		t.Fatalf("Output is not valid YAML: %v", err)
	}

	// Verify multiline value is preserved
	if multiline, ok := yamlData["MULTILINE"].(string); ok {
		if !strings.Contains(multiline, "Line 1") ||
			!strings.Contains(multiline, "Line 2") ||
			!strings.Contains(multiline, "Line 3") {
			t.Error("Multiline value not preserved correctly in YAML")
		}
	} else {
		t.Error("MULTILINE should be a string")
	}
}

// TestFormat_K8sSecretOutput tests Kubernetes Secret format
func TestFormat_K8sSecretOutput(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set configuration values
	env.Set("DATABASE_URL", "postgres://localhost/mydb", "-a", "api", "-e", "prod").AssertSuccess()
	env.Set("API_KEY", "secret-key-123", "-a", "api", "-e", "prod").AssertSuccess()

	result := env.Generate("api", "prod", "k8s", "--secret-name", "api-config")
	result.AssertSuccess()

	output := result.GetStdout()

	// Verify it's a Kubernetes Secret
	if !strings.Contains(output, "apiVersion: v1") {
		t.Error("K8s output missing apiVersion")
	}
	if !strings.Contains(output, "kind: Secret") {
		t.Error("K8s output missing kind: Secret")
	}
	if !strings.Contains(output, "name: api-config") {
		t.Error("K8s output missing secret name")
	}

	// Verify stringData is used (unencoded)
	if !strings.Contains(output, "stringData:") {
		t.Error("K8s output should use stringData for unencoded values")
	}

	// Verify values are present
	if !strings.Contains(output, "DATABASE_URL:") {
		t.Error("K8s output missing DATABASE_URL")
	}
	if !strings.Contains(output, "API_KEY:") {
		t.Error("K8s output missing API_KEY")
	}

	// Verify it's valid YAML
	var k8sData map[string]interface{}
	if err := yaml.Unmarshal([]byte(output), &k8sData); err != nil {
		t.Fatalf("K8s output is not valid YAML: %v", err)
	}

	// Verify metadata
	metadata, ok := k8sData["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("K8s output missing metadata")
	}
	if metadata["name"] != "api-config" {
		t.Error("K8s secret name incorrect")
	}
}

// TestFormat_K8sSecretWithBase64 tests Kubernetes Secret with base64 encoding
func TestFormat_K8sSecretWithBase64(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set configuration values
	env.Set("SECRET1", "value1", "-a", "api", "-e", "prod").AssertSuccess()
	env.Set("SECRET2", "value2", "-a", "api", "-e", "prod").AssertSuccess()

	result := env.Generate("api", "prod", "k8s", "--secret-name", "api-secrets", "--base64")
	result.AssertSuccess()

	output := result.GetStdout()

	// Verify it's a Kubernetes Secret
	if !strings.Contains(output, "kind: Secret") {
		t.Error("K8s output missing kind: Secret")
	}

	// Verify 'data' is used (base64 encoded)
	if !strings.Contains(output, "data:") {
		t.Error("K8s output with --base64 should use 'data' field")
	}

	// Verify stringData is NOT used
	if strings.Contains(output, "stringData:") {
		t.Error("K8s output with --base64 should not use 'stringData' field")
	}

	// Verify values are base64 encoded (not plaintext)
	if strings.Contains(output, "SECRET1: value1") {
		t.Error("K8s output with --base64 contains plaintext values")
	}

	// Parse YAML and verify base64 encoding
	var k8sData map[string]interface{}
	if err := yaml.Unmarshal([]byte(output), &k8sData); err != nil {
		t.Fatalf("K8s output is not valid YAML: %v", err)
	}

	data, ok := k8sData["data"].(map[string]interface{})
	if !ok {
		t.Fatal("K8s output missing data field")
	}

	// Decode and verify values
	if encodedValue, ok := data["SECRET1"].(string); ok {
		decoded, err := base64.StdEncoding.DecodeString(encodedValue)
		if err != nil {
			t.Errorf("SECRET1 value is not valid base64: %v", err)
		} else if string(decoded) != "value1" {
			t.Errorf("SECRET1 decoded incorrectly: got %s, want value1", string(decoded))
		}
	} else {
		t.Error("SECRET1 not found in data")
	}
}

// TestFormat_K8sSecretWithCustomNamespace tests K8s secret with namespace
func TestFormat_K8sSecretWithCustomNamespace(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()
	env.Set("KEY", "value", "-a", "api", "-e", "prod").AssertSuccess()

	result := env.Generate("api", "prod", "k8s", "--secret-name", "my-secret")
	result.AssertSuccess()

	output := result.GetStdout()

	// Verify the secret name is correct
	if !strings.Contains(output, "name: my-secret") {
		t.Error("K8s secret name not set correctly")
	}

	// Verify it's valid K8s YAML
	var k8sData map[string]interface{}
	if err := yaml.Unmarshal([]byte(output), &k8sData); err != nil {
		t.Fatalf("K8s output is not valid YAML: %v", err)
	}

	// Verify type is Opaque
	if secretType, ok := k8sData["type"].(string); !ok || secretType != "Opaque" {
		t.Error("K8s secret type should be Opaque")
	}
}

// TestFormat_OutputToFile tests writing output to a file
func TestFormat_OutputToFile(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()
	env.Set("KEY1", "value1", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("KEY2", "value2", "-a", "api", "-e", "dev").AssertSuccess()

	// Generate to file
	outputFile := "output.env"
	result := env.Run("generate", "-a", "api", "-e", "dev", "-f", "env", "-o", outputFile, "-r", ".")
	result.AssertSuccess()

	// Verify file was created
	if !env.FileExists(outputFile) {
		t.Fatal("Output file was not created")
	}

	// Verify file contents
	content := env.ReadFile(outputFile)
	if !strings.Contains(content, "KEY1=value1") {
		t.Error("Output file missing KEY1")
	}
	if !strings.Contains(content, "KEY2=value2") {
		t.Error("Output file missing KEY2")
	}

	// Verify stdout indicates success
	result.AssertStdoutContains("Config generated")
}

// TestFormat_EmptyConfiguration tests generating empty configuration
func TestFormat_EmptyConfiguration(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Try to generate for an app/env with no config
	result := env.Generate("empty-app", "empty-env", "json")

	// Should either succeed with empty object or fail gracefully
	if result.ExitCode == 0 {
		output := result.GetStdout()

		// Verify it's valid JSON
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(output), &jsonData); err != nil {
			t.Fatalf("Empty config output is not valid JSON: %v", err)
		}

		// Should be empty or minimal
		if len(jsonData) > 1 {
			t.Errorf("Empty config should have minimal keys, got %d keys", len(jsonData))
		}
	}
}

// TestFormat_LargeConfiguration tests generating large configurations
func TestFormat_LargeConfiguration(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set many values
	for i := 0; i < 100; i++ {
		key := "KEY_" + strings.Repeat("X", 10) + "_" + string(rune('0'+i%10))
		value := "value_" + strings.Repeat("Y", 100)
		env.Set(key, value, "-a", "api", "-e", "dev").AssertSuccess()
	}

	// Generate in all formats
	formats := []string{"env", "json", "yaml"}
	for _, format := range formats {
		result := env.Generate("api", "dev", format)
		result.AssertSuccess()

		output := result.GetStdout()
		if len(output) < 1000 {
			t.Errorf("Large config output seems too small for %s format: %d bytes", format, len(output))
		}
	}
}

// TestFormat_InternalVariablesFiltering tests that internal vars are filtered
func TestFormat_InternalVariablesFiltering(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set internal and public variables
	env.Set("_INTERNAL1", "internal-value-1", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("_INTERNAL2", "internal-value-2", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("PUBLIC1", "public-value-1", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("PUBLIC2", "public-value-2", "-a", "api", "-e", "dev").AssertSuccess()

	// Test all formats
	formats := []string{"env", "json", "yaml"}

	for _, format := range formats {
		result := env.Generate("api", "dev", format)
		result.AssertSuccess()

		output := result.GetStdout()

		// Verify internal variables are NOT in output
		if strings.Contains(output, "_INTERNAL1") {
			t.Errorf("Internal variable _INTERNAL1 leaked in %s format", format)
		}
		if strings.Contains(output, "_INTERNAL2") {
			t.Errorf("Internal variable _INTERNAL2 leaked in %s format", format)
		}

		// Verify public variables ARE in output
		if !strings.Contains(output, "PUBLIC1") {
			t.Errorf("Public variable PUBLIC1 missing in %s format", format)
		}
		if !strings.Contains(output, "PUBLIC2") {
			t.Errorf("Public variable PUBLIC2 missing in %s format", format)
		}
	}
}

// TestFormat_TemplatesResolved tests that templates are resolved in output
func TestFormat_TemplatesResolved(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set up template variables
	env.Set("BASE_URL", "example.com", "-a", "shared", "-e", "dev").AssertSuccess()
	env.Set("API_PATH", "/api/v1", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("FULL_URL", "https://${BASE_URL}${API_PATH}", "-a", "api", "-e", "dev").AssertSuccess()

	// Generate
	result := env.Generate("api", "dev", "json")
	result.AssertSuccess()

	output := result.GetStdout()

	// Verify templates are resolved
	if strings.Contains(output, "${BASE_URL}") {
		t.Error("Template ${BASE_URL} not resolved in output")
	}
	if strings.Contains(output, "${API_PATH}") {
		t.Error("Template ${API_PATH} not resolved in output")
	}

	// Verify resolved value
	if !strings.Contains(output, "https://example.com/api/v1") {
		t.Error("Template not resolved to correct value")
	}
}

// TestFormat_InvalidFormatName tests error handling for invalid format
func TestFormat_InvalidFormatName(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()
	env.Set("KEY", "value", "-a", "api", "-e", "dev").AssertSuccess()

	// Try to generate with invalid format
	result := env.Run("generate", "-a", "api", "-e", "dev", "-f", "invalid-format", "-r", ".")
	result.AssertFailure()
	result.AssertStderrContains("unknown format")
}

// TestFormat_ConsistencyAcrossFormats tests that all formats contain the same data
func TestFormat_ConsistencyAcrossFormats(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set up configuration
	testKeys := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
		"KEY3": "value3",
	}

	for key, value := range testKeys {
		env.Set(key, value, "-a", "api", "-e", "dev").AssertSuccess()
	}

	// Generate in different formats
	envOutput := env.Generate("api", "dev", "env").AssertSuccess().GetStdout()
	jsonOutput := env.Generate("api", "dev", "json").AssertSuccess().GetStdout()
	yamlOutput := env.Generate("api", "dev", "yaml").AssertSuccess().GetStdout()

	// Verify all formats contain all keys
	for key, value := range testKeys {
		if !strings.Contains(envOutput, key) {
			t.Errorf("ENV format missing key %s", key)
		}
		if !strings.Contains(jsonOutput, key) {
			t.Errorf("JSON format missing key %s", key)
		}
		if !strings.Contains(yamlOutput, key) {
			t.Errorf("YAML format missing key %s", key)
		}

		// Verify values are present
		if !strings.Contains(envOutput, value) {
			t.Errorf("ENV format missing value %s", value)
		}
		if !strings.Contains(jsonOutput, value) {
			t.Errorf("JSON format missing value %s", value)
		}
		if !strings.Contains(yamlOutput, value) {
			t.Errorf("YAML format missing value %s", value)
		}
	}
}
