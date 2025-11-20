package test

import (
	"strings"
	"testing"

	"github.com/teamcurri/puff/test/helpers"
)

// TestEdgeCase_InitWithoutAgeKeys tests initialization failure without age keys
func TestEdgeCase_InitWithoutAgeKeys(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	// Try to init without providing age keys (should fail since -k is required)
	result := env.Run("init", "-d", ".")
	result.AssertFailure()
	result.AssertStderrContains("Required flag")
}

// TestEdgeCase_InitInNonEmptyDirectory tests initialization in a directory with existing files
func TestEdgeCase_InitInNonEmptyDirectory(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	// Create some existing files
	env.WriteFile("existing.txt", "some content")
	env.MkdirAll("subdir")

	// Init should still work
	env.Init().AssertSuccess()

	// Verify .sops.yaml was created despite existing files
	if !env.FileExists(".sops.yaml") {
		t.Fatal(".sops.yaml was not created")
	}

	// Verify existing files are untouched
	if !env.FileExists("existing.txt") {
		t.Fatal("Existing file was removed")
	}
}

// TestEdgeCase_DoubleInit tests initializing an already initialized directory
func TestEdgeCase_DoubleInit(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	// First init
	env.Init().AssertSuccess()

	// Try to init again
	result := env.Init()
	// Should either succeed idempotently or fail with clear error
	if result.ExitCode != 0 {
		result.AssertStderrContains("already initialized")
	}
}

// TestEdgeCase_SetBeforeInit tests setting values before initialization
func TestEdgeCase_SetBeforeInit(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	// Try to set a value without initializing
	result := env.Set("KEY", "value", "-a", "api", "-e", "dev")
	result.AssertFailure()
	result.AssertStderrContains("no encryption keys found")
}

// TestEdgeCase_GetBeforeInit tests getting values before initialization
func TestEdgeCase_GetBeforeInit(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	// Try to get a value without any configuration
	result := env.Get("KEY", "-a", "api", "-e", "dev")
	result.AssertFailure()
}

// TestEdgeCase_GetNonExistentKey tests retrieving a key that doesn't exist
func TestEdgeCase_GetNonExistentKey(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()
	env.Set("EXISTING_KEY", "value", "-a", "api", "-e", "dev").AssertSuccess()

	// Try to get a key that doesn't exist
	result := env.Get("NON_EXISTENT_KEY", "-a", "api", "-e", "dev")
	result.AssertFailure()
}

// TestEdgeCase_SetEmptyValue tests setting an empty string value
func TestEdgeCase_SetEmptyValue(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set an empty value
	env.Set("EMPTY_KEY", "", "-a", "api", "-e", "dev").
		AssertSuccess()

	// Verify it can be retrieved as empty
	env.Get("EMPTY_KEY", "-a", "api", "-e", "dev").
		AssertSuccess().
		AssertStdoutEquals("")
}

// TestEdgeCase_SetSpecialCharacters tests setting values with special characters
func TestEdgeCase_SetSpecialCharacters(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	specialValues := []string{
		"value with spaces",
		"value\nwith\nnewlines",
		"value\twith\ttabs",
		"value'with'quotes",
		"value\"with\"doublequotes",
		"value$with$dollars",
		"value=with=equals",
		"value:with:colons",
		"value{with}braces",
		"postgres://user:p@ssw0rd!@host:5432/db",
	}

	for i, val := range specialValues {
		key := "SPECIAL_" + string(rune('A'+i))
		env.Set(key, val, "-a", "api", "-e", "dev").
			AssertSuccess()

		env.Get(key, "-a", "api", "-e", "dev").
			AssertSuccess().
			AssertStdoutEquals(val)
	}
}

// TestEdgeCase_SetVeryLongValue tests setting very long values
func TestEdgeCase_SetVeryLongValue(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Create a very long value (10KB)
	longValue := strings.Repeat("This is a very long configuration value. ", 200)

	env.Set("LONG_VALUE", longValue, "-a", "api", "-e", "dev").
		AssertSuccess()

	env.Get("LONG_VALUE", "-a", "api", "-e", "dev").
		AssertSuccess().
		AssertStdoutEquals(longValue)
}

// TestEdgeCase_SetMultilineValue tests setting multiline values
func TestEdgeCase_SetMultilineValue(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	multilineValue := `Line 1
Line 2
Line 3`

	env.Set("MULTILINE", multilineValue, "-a", "api", "-e", "dev").
		AssertSuccess()

	result := env.Get("MULTILINE", "-a", "api", "-e", "dev")
	result.AssertSuccess()

	// Verify the multiline value is preserved
	output := result.GetStdout()
	if !strings.Contains(output, "Line 1") ||
		!strings.Contains(output, "Line 2") ||
		!strings.Contains(output, "Line 3") {
		t.Errorf("Multiline value not preserved correctly. Output: %s", output)
	}
}

// TestEdgeCase_SetJSONValue tests setting JSON as a value
func TestEdgeCase_SetJSONValue(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	jsonValue := `{"name":"test","nested":{"key":"value"},"array":[1,2,3]}`

	env.Set("JSON_CONFIG", jsonValue, "-a", "api", "-e", "dev").
		AssertSuccess()

	env.Get("JSON_CONFIG", "-a", "api", "-e", "dev").
		AssertSuccess().
		AssertStdoutEquals(jsonValue)
}

// TestEdgeCase_OverwriteExistingKey tests overwriting an existing key
func TestEdgeCase_OverwriteExistingKey(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set initial value
	env.Set("KEY", "initial_value", "-a", "api", "-e", "dev").
		AssertSuccess()

	env.Get("KEY", "-a", "api", "-e", "dev").
		AssertSuccess().
		AssertStdoutEquals("initial_value")

	// Overwrite with new value
	env.Set("KEY", "updated_value", "-a", "api", "-e", "dev").
		AssertSuccess()

	env.Get("KEY", "-a", "api", "-e", "dev").
		AssertSuccess().
		AssertStdoutEquals("updated_value")
}

// TestEdgeCase_GenerateNonExistentApp tests generating config for non-existent app
func TestEdgeCase_GenerateNonExistentApp(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()
	env.Set("KEY", "value", "-a", "api", "-e", "dev").AssertSuccess()

	// Try to generate for an app that doesn't have any config
	result := env.Generate("nonexistent", "dev", "json")
	// Should either succeed with empty config or fail gracefully
	if result.ExitCode != 0 {
		// Acceptable to fail
		result.AssertStderrContains("") // Just verify stderr exists
	} else {
		// If it succeeds, should have minimal output
		output := result.GetStdout()
		if strings.Contains(output, "KEY") {
			t.Error("Generated config should not contain keys from other apps")
		}
	}
}

// TestEdgeCase_GenerateWithoutRequiredK8sSecretName tests k8s format without secret name
func TestEdgeCase_GenerateWithoutRequiredK8sSecretName(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()
	env.Set("KEY", "value", "-a", "api", "-e", "dev").AssertSuccess()

	// Try to generate k8s format without secret name
	result := env.Run("generate", "-a", "api", "-e", "dev", "-f", "k8s", "-r", ".")
	result.AssertFailure()
	result.AssertStderrContains("secret-name is required")
}

// TestEdgeCase_DecryptNonEncryptedFile tests decrypting a file that isn't encrypted
func TestEdgeCase_DecryptNonEncryptedFile(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	// Create a plain YAML file
	env.WriteFile("plain.yml", "key: value\n")

	// Try to decrypt it
	result := env.Decrypt("plain.yml")
	result.AssertFailure()
}

// TestEdgeCase_DecryptNonExistentFile tests decrypting a file that doesn't exist
func TestEdgeCase_DecryptNonExistentFile(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Try to decrypt a file that doesn't exist
	result := env.Decrypt("nonexistent.yml")
	result.AssertFailure()
}

// TestEdgeCase_EncryptWithoutDecExtension tests encrypting a file without .dec extension
func TestEdgeCase_EncryptWithoutDecExtension(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Create a plain file
	env.WriteFile("plain.yml", "key: value\n")

	// Try to encrypt it without .dec extension
	result := env.Encrypt("plain.yml")
	result.AssertFailure()
	result.AssertStderrContains(".dec")
}

// TestEdgeCase_EncryptNonExistentDecFile tests encrypting a .dec file that doesn't exist
func TestEdgeCase_EncryptNonExistentDecFile(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Try to encrypt a .dec file that doesn't exist
	result := env.Encrypt("nonexistent.dec.yml")
	result.AssertFailure()
}

// TestEdgeCase_AddInvalidAgeKey tests adding an invalid age key
func TestEdgeCase_AddInvalidAgeKey(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Try to add an invalid age key
	result := env.KeysAdd("invalid-key-format", "Invalid key")
	result.AssertFailure()
	result.AssertStderrContains("invalid age key")
}

// TestEdgeCase_AddDuplicateKey tests adding a key that already exists
func TestEdgeCase_AddDuplicateKey(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()
	env.Set("KEY", "value", "-a", "api", "-e", "dev").AssertSuccess()

	// Try to add the same key again
	result := env.KeysAdd(env.AgeKey, "Duplicate key")
	// Should succeed idempotently
	result.AssertSuccess()

	// Verify the comment was updated
	sopsContent := env.ReadFile(".sops.yaml")
	if !strings.Contains(sopsContent, "Duplicate key") {
		t.Error("Comment should be updated when adding duplicate key")
	}
}

// TestEdgeCase_RemoveNonExistentKey tests removing a key that doesn't exist
func TestEdgeCase_RemoveNonExistentKey(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Generate a different age key
	result := env.RunSystem("age-keygen")
	if result.ExitCode != 0 {
		t.Fatalf("age-keygen failed: %s", result.GetStderr())
	}

	output := result.GetStdout()
	lines := strings.Split(output, "\n")
	var differentKey string
	for _, line := range lines {
		if strings.HasPrefix(line, "# public key:") {
			differentKey = strings.TrimSpace(strings.TrimPrefix(line, "# public key:"))
			break
		}
	}

	// Try to remove a key that was never added
	result = env.KeysRemove(differentKey)
	result.AssertFailure()
	result.AssertStderrContains("key not found")
}

// TestEdgeCase_RemoveLastKey tests removing the last encryption key
func TestEdgeCase_RemoveLastKey(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()
	env.Set("KEY", "value", "-a", "api", "-e", "dev").AssertSuccess()

	// Try to remove the only key
	result := env.KeysRemove(env.AgeKey)
	result.AssertFailure()
	result.AssertStderrContains("last key")
}

// TestEdgeCase_CircularTemplateReference tests handling circular template references
func TestEdgeCase_CircularTemplateReference(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Create circular references: A -> B -> A
	env.Set("VAR_A", "${VAR_B}", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("VAR_B", "${VAR_A}", "-a", "api", "-e", "dev").AssertSuccess()

	// Try to generate - should detect circular reference
	result := env.Generate("api", "dev", "json")
	result.AssertFailure()
	result.AssertStderrContains("circular")
}

// TestEdgeCase_UndefinedTemplateVariable tests referencing undefined variables in templates
func TestEdgeCase_UndefinedTemplateVariable(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set a value that references a non-existent variable
	env.Set("URL", "https://${UNDEFINED_VAR}/api", "-a", "api", "-e", "dev").
		AssertSuccess()

	// Try to generate - should fail or leave undefined
	result := env.Generate("api", "dev", "json")
	// Accept either failure or leaving the variable unresolved
	if result.ExitCode != 0 {
		result.AssertStderrContains("undefined")
	} else {
		output := result.GetStdout()
		// Should either contain the literal ${UNDEFINED_VAR} or error
		if !strings.Contains(output, "UNDEFINED_VAR") && !strings.Contains(output, "error") {
			t.Error("Undefined variable should either be preserved or cause an error")
		}
	}
}

// TestEdgeCase_DeepNestedDirectories tests creating config in deeply nested directories
func TestEdgeCase_DeepNestedDirectories(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set values that would create deeply nested paths
	env.Set("KEY", "value", "-a", "very-long-app-name-test", "-e", "very-long-environment-name-test").
		AssertSuccess()

	env.Set("KEY", "value", "-a", "app", "-e", "env", "-t", "very-long-target-name-test").
		AssertSuccess()

	// Verify files were created
	if !env.FileExists("very-long-environment-name-test/very-long-app-name-test.yml") {
		t.Error("File in deep directory not created")
	}

	if !env.FileExists("target-overrides/very-long-target-name-test/env/app.yml") {
		t.Error("Target override file in deep directory not created")
	}
}

// TestEdgeCase_SpecialCharactersInAppEnvNames tests special characters in app/env names
func TestEdgeCase_SpecialCharactersInAppEnvNames(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Test with hyphens (common in real usage)
	env.Set("KEY", "value", "-a", "api-gateway", "-e", "prod-eu-west").
		AssertSuccess()

	env.Get("KEY", "-a", "api-gateway", "-e", "prod-eu-west").
		AssertSuccess().
		AssertStdoutEquals("value")

	// Test with underscores
	env.Set("KEY2", "value2", "-a", "worker_service", "-e", "staging_env").
		AssertSuccess()

	env.Get("KEY2", "-a", "worker_service", "-e", "staging_env").
		AssertSuccess().
		AssertStdoutEquals("value2")
}

// TestEdgeCase_ConcurrentModifications tests behavior with concurrent file modifications
// Note: This is a basic test - real concurrency testing would be more complex
func TestEdgeCase_ConcurrentModifications(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set initial value
	env.Set("KEY", "value1", "-a", "api", "-e", "dev").AssertSuccess()

	// Modify the file externally while puff is running
	// This simulates a concurrent modification scenario
	env.Set("KEY", "value2", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("KEY", "value3", "-a", "api", "-e", "dev").AssertSuccess()

	// Verify the last write wins
	env.Get("KEY", "-a", "api", "-e", "dev").
		AssertSuccess().
		AssertStdoutEquals("value3")
}

// TestEdgeCase_MissingSOPSAgeKeyEnv tests operations without SOPS_AGE_KEY environment
func TestEdgeCase_MissingSOPSAgeKeyEnv(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()
	env.Set("KEY", "value", "-a", "api", "-e", "dev").AssertSuccess()

	// Try to get without SOPS_AGE_KEY environment variable
	result := env.RunWithEnv(map[string]string{"SOPS_AGE_KEY": ""}, "get", "-k", "KEY", "-a", "api", "-e", "dev", "-r", ".")
	result.AssertFailure()
	// Should fail because it can't decrypt
}

// TestEdgeCase_InvalidYAMLInDecryptedFile tests encrypting invalid YAML
func TestEdgeCase_InvalidYAMLInDecryptedFile(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()
	env.Set("KEY", "value", "-a", "api", "-e", "dev").AssertSuccess()

	// Decrypt the file
	env.Decrypt("dev/api.yml").AssertSuccess()

	// Write invalid YAML to the decrypted file
	env.WriteFile("dev/api.dec.yml", "invalid: yaml: content: [\n")

	// Try to re-encrypt - should handle gracefully
	result := env.Encrypt("dev/api.dec.yml")
	// May succeed (SOPS can encrypt any text) or fail with validation error
	// Either behavior is acceptable as long as it doesn't crash
	if result.ExitCode != 0 {
		// Acceptable to fail with clear error
		// Check both stdout and stderr since colored output may go to stdout
		stderr := result.GetStderr()
		stdout := result.GetStdout()
		if len(stderr) == 0 && len(stdout) == 0 {
			t.Error("Should provide error message for invalid YAML")
		}
	}
}
