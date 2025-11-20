package test

import (
	"strings"
	"testing"

	"github.com/teamcurri/puff/test/helpers"
)

// TestWorkflow_BasicInitAndUsage tests the basic initialization and usage workflow
func TestWorkflow_BasicInitAndUsage(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	// Initialize puff directory
	env.Init().AssertSuccess()

	// Verify .sops.yaml was created
	if !env.FileExists(".sops.yaml") {
		t.Fatal(".sops.yaml was not created")
	}

	// Set a simple configuration value
	env.Set("DATABASE_URL", "postgres://localhost/mydb", "-a", "api", "-e", "dev").
		AssertSuccess()

	// Verify the encrypted file was created
	if !env.FileExists("dev/api.yml") {
		t.Fatal("dev/api.yml was not created")
	}

	// Read the encrypted file to verify it contains SOPS metadata
	content := env.ReadFile("dev/api.yml")
	if !strings.Contains(content, "sops:") {
		t.Fatal("File is not encrypted with SOPS")
	}
	if !strings.Contains(content, "age:") {
		t.Fatal("File does not contain age encryption metadata")
	}

	// Retrieve the value
	env.Get("DATABASE_URL", "-a", "api", "-e", "dev").
		AssertSuccess().
		AssertStdoutEquals("postgres://localhost/mydb")

	// Set another value in the same app/env
	env.Set("REDIS_URL", "redis://localhost:6379", "-a", "api", "-e", "dev").
		AssertSuccess()

	// Verify both values are retrievable
	env.Get("DATABASE_URL", "-a", "api", "-e", "dev").
		AssertSuccess().
		AssertStdoutEquals("postgres://localhost/mydb")

	env.Get("REDIS_URL", "-a", "api", "-e", "dev").
		AssertSuccess().
		AssertStdoutEquals("redis://localhost:6379")
}

// TestWorkflow_MultiAppMultiEnv tests setting up multiple apps and environments
func TestWorkflow_MultiAppMultiEnv(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set up shared configuration
	env.Set("COMPANY_NAME", "Acme Corp", "-a", "shared", "-e", "base").
		AssertSuccess()

	// Set up api app in dev environment
	env.Set("API_PORT", "3000", "-a", "api", "-e", "dev").
		AssertSuccess()
	env.Set("DB_HOST", "localhost", "-a", "api", "-e", "dev").
		AssertSuccess()

	// Set up api app in production environment
	env.Set("API_PORT", "8080", "-a", "api", "-e", "production").
		AssertSuccess()
	env.Set("DB_HOST", "prod-db.example.com", "-a", "api", "-e", "production").
		AssertSuccess()

	// Set up worker app in dev environment
	env.Set("WORKER_THREADS", "4", "-a", "worker", "-e", "dev").
		AssertSuccess()

	// Set up worker app in production environment
	env.Set("WORKER_THREADS", "16", "-a", "worker", "-e", "production").
		AssertSuccess()

	// Verify shared value is accessible from all contexts
	env.Get("COMPANY_NAME", "-a", "api", "-e", "dev").
		AssertSuccess().
		AssertStdoutEquals("Acme Corp")

	env.Get("COMPANY_NAME", "-a", "worker", "-e", "production").
		AssertSuccess().
		AssertStdoutEquals("Acme Corp")

	// Verify environment-specific values
	env.Get("API_PORT", "-a", "api", "-e", "dev").
		AssertSuccess().
		AssertStdoutEquals("3000")

	env.Get("API_PORT", "-a", "api", "-e", "production").
		AssertSuccess().
		AssertStdoutEquals("8080")

	// Verify app-specific values
	env.Get("WORKER_THREADS", "-a", "worker", "-e", "dev").
		AssertSuccess().
		AssertStdoutEquals("4")

	env.Get("WORKER_THREADS", "-a", "worker", "-e", "production").
		AssertSuccess().
		AssertStdoutEquals("16")

	// Verify files were created in correct locations
	expectedFiles := []string{
		"base/shared.yml",
		"dev/api.yml",
		"production/api.yml",
		"dev/worker.yml",
		"production/worker.yml",
	}

	for _, file := range expectedFiles {
		if !env.FileExists(file) {
			t.Errorf("Expected file %s was not created", file)
		}
	}
}

// TestWorkflow_TemplateVariableChains tests template variable resolution across layers
func TestWorkflow_TemplateVariableChains(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set up base domain
	env.Set("BASE_DOMAIN", "example.com", "-a", "shared", "-e", "base").
		AssertSuccess()

	// Set up environment subdomain
	env.Set("ENV_SUBDOMAIN", "dev", "-a", "shared", "-e", "dev").
		AssertSuccess()

	// Set up app-specific path using templates
	env.Set("API_URL", "https://${ENV_SUBDOMAIN}.${BASE_DOMAIN}/api", "-a", "api", "-e", "dev").
		AssertSuccess()

	// Generate configuration to resolve templates
	result := env.Generate("api", "dev", "json")
	result.AssertSuccess()

	// Verify the template was resolved correctly
	output := result.GetStdout()
	if !strings.Contains(output, "https://dev.example.com/api") {
		t.Errorf("Template not resolved correctly. Output: %s", output)
	}

	// Test multi-level template chains
	env.Set("SERVICE_NAME", "payments", "-a", "payments", "-e", "dev").
		AssertSuccess()
	env.Set("SERVICE_URL", "https://${ENV_SUBDOMAIN}.${BASE_DOMAIN}/${SERVICE_NAME}", "-a", "payments", "-e", "dev").
		AssertSuccess()

	result = env.Generate("payments", "dev", "json")
	result.AssertSuccess()

	output = result.GetStdout()
	if !strings.Contains(output, "https://dev.example.com/payments") {
		t.Errorf("Multi-level template not resolved correctly. Output: %s", output)
	}
}

// TestWorkflow_TemplateWithInternalVariables tests underscore-prefixed internal variables
func TestWorkflow_TemplateWithInternalVariables(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set up internal variables (underscore-prefixed)
	env.Set("_DB_USER", "admin", "-a", "api", "-e", "dev").
		AssertSuccess()
	env.Set("_DB_PASS", "secret123", "-a", "api", "-e", "dev").
		AssertSuccess()
	env.Set("_DB_HOST", "localhost", "-a", "api", "-e", "dev").
		AssertSuccess()

	// Set up public variable using internal variables
	env.Set("DATABASE_URL", "postgres://${_DB_USER}:${_DB_PASS}@${_DB_HOST}/mydb", "-a", "api", "-e", "dev").
		AssertSuccess()

	// Generate configuration
	result := env.Generate("api", "dev", "json")
	result.AssertSuccess()

	output := result.GetStdout()

	// Verify that the DATABASE_URL was resolved
	if !strings.Contains(output, "postgres://admin:secret123@localhost/mydb") {
		t.Errorf("Template with internal variables not resolved. Output: %s", output)
	}

	// Verify that internal variables are NOT in the output
	if strings.Contains(output, "_DB_USER") {
		t.Error("Internal variable _DB_USER should not appear in generated output")
	}
	if strings.Contains(output, "_DB_PASS") {
		t.Error("Internal variable _DB_PASS should not appear in generated output")
	}
	if strings.Contains(output, "_DB_HOST") {
		t.Error("Internal variable _DB_HOST should not appear in generated output")
	}
}

// TestWorkflow_TargetOverrides tests target-specific configuration overrides
func TestWorkflow_TargetOverrides(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set base configuration
	env.Set("API_PORT", "3000", "-a", "api", "-e", "dev").
		AssertSuccess()
	env.Set("LOG_LEVEL", "info", "-a", "api", "-e", "dev").
		AssertSuccess()

	// Set target override for local development
	env.Set("API_PORT", "8080", "-a", "api", "-e", "dev", "-t", "local").
		AssertSuccess()
	env.Set("LOG_LEVEL", "debug", "-a", "api", "-e", "dev", "-t", "local").
		AssertSuccess()

	// Set target override for docker
	env.Set("API_PORT", "80", "-a", "api", "-e", "dev", "-t", "docker").
		AssertSuccess()

	// Verify target override files were created
	if !env.FileExists("target-overrides/local/dev/api.yml") {
		t.Fatal("Local target override file not created")
	}
	if !env.FileExists("target-overrides/docker/dev/api.yml") {
		t.Fatal("Docker target override file not created")
	}

	// Generate without target (should use base values)
	result := env.Generate("api", "dev", "json", "-t", "local")
	result.AssertSuccess()
	output := result.GetStdout()

	if !strings.Contains(output, "8080") {
		t.Error("Local target override not applied for API_PORT")
	}
	if !strings.Contains(output, "debug") {
		t.Error("Local target override not applied for LOG_LEVEL")
	}

	// Generate with docker target
	result = env.Generate("api", "dev", "json", "-t", "docker")
	result.AssertSuccess()
	output = result.GetStdout()

	if !strings.Contains(output, "\"API_PORT\":\"80\"") && !strings.Contains(output, "\"API_PORT\": \"80\"") {
		t.Errorf("Docker target override not applied for API_PORT. Output: %s", output)
	}
	// LOG_LEVEL should fall back to base value
	if !strings.Contains(output, "info") {
		t.Error("Base value not used when target override doesn't exist")
	}
}

// TestWorkflow_BulkEditDecryptEncrypt tests the decrypt/edit/encrypt workflow
func TestWorkflow_BulkEditDecryptEncrypt(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set up multiple values
	env.Set("KEY1", "value1", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("KEY2", "value2", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("KEY3", "value3", "-a", "api", "-e", "dev").AssertSuccess()

	configFile := "dev/api.yml"

	// Decrypt the file
	env.Decrypt(configFile).AssertSuccess()

	decFile := "dev/api.dec.yml"
	if !env.FileExists(decFile) {
		t.Fatal("Decrypted file was not created")
	}

	// Read and verify the decrypted file is plaintext YAML
	decContent := env.ReadFile(decFile)
	if strings.Contains(decContent, "sops:") {
		t.Fatal("Decrypted file still contains SOPS metadata")
	}
	if !strings.Contains(decContent, "KEY1: value1") {
		t.Error("Decrypted file doesn't contain KEY1")
	}

	// Edit the decrypted file manually
	editedContent := strings.ReplaceAll(decContent, "value2", "modified_value2")
	editedContent = strings.ReplaceAll(editedContent, "value3", "modified_value3")
	env.WriteFile(decFile, editedContent)

	// Re-encrypt the file
	env.Encrypt(decFile).AssertSuccess()

	// Verify the .dec file was removed
	if env.FileExists(decFile) {
		t.Error("Decrypted file should be removed after encryption")
	}

	// Verify the encrypted file was updated
	if !env.FileExists(configFile) {
		t.Fatal("Encrypted file was removed")
	}

	// Verify values were updated
	env.Get("KEY1", "-a", "api", "-e", "dev").
		AssertSuccess().
		AssertStdoutEquals("value1")

	env.Get("KEY2", "-a", "api", "-e", "dev").
		AssertSuccess().
		AssertStdoutEquals("modified_value2")

	env.Get("KEY3", "-a", "api", "-e", "dev").
		AssertSuccess().
		AssertStdoutEquals("modified_value3")
}

// TestWorkflow_KeyManagement tests adding and removing encryption keys
func TestWorkflow_KeyManagement(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set up some configuration
	env.Set("SECRET_KEY", "my-secret", "-a", "api", "-e", "dev").
		AssertSuccess()

	// Generate a second age key for testing
	result := env.RunSystem("age-keygen")
	if result.ExitCode != 0 {
		t.Fatalf("age-keygen failed: %s", result.GetStderr())
	}

	output := result.GetStdout()
	lines := strings.Split(output, "\n")
	var secondPublicKey, secondSecretKey string
	for _, line := range lines {
		if strings.HasPrefix(line, "# public key:") {
			secondPublicKey = strings.TrimSpace(strings.TrimPrefix(line, "# public key:"))
		} else if strings.HasPrefix(line, "AGE-SECRET-KEY-") {
			secondSecretKey = strings.TrimSpace(line)
		}
	}

	if secondPublicKey == "" || secondSecretKey == "" {
		t.Fatal("Failed to generate second age key")
	}

	// Add the second key
	env.KeysAdd(secondPublicKey, "Second admin key").
		AssertSuccess()

	// Verify .sops.yaml contains both keys
	sopsContent := env.ReadFile(".sops.yaml")
	if !strings.Contains(sopsContent, env.AgeKey) {
		t.Error("Original key not found in .sops.yaml")
	}
	if !strings.Contains(sopsContent, secondPublicKey) {
		t.Error("Second key not found in .sops.yaml")
	}
	if !strings.Contains(sopsContent, "Second admin key") {
		t.Error("Second key comment not found in .sops.yaml")
	}

	// Verify the config file was updated with the new key
	configContent := env.ReadFile("dev/api.yml")
	if !strings.Contains(configContent, secondPublicKey) {
		t.Error("Second key not added to encrypted file")
	}

	// Verify we can decrypt with the second key
	result = env.RunWithEnv(map[string]string{"SOPS_AGE_KEY": secondSecretKey}, "get", "-k", "SECRET_KEY", "-a", "api", "-e", "dev", "-r", ".")
	result.AssertSuccess().AssertStdoutEquals("my-secret")

	// List keys
	env.KeysList().AssertSuccess().
		AssertStdoutContains(env.AgeKey).
		AssertStdoutContains(secondPublicKey)

	// Remove the second key
	env.KeysRemove(secondPublicKey).AssertSuccess()

	// Verify .sops.yaml no longer contains the second key
	sopsContent = env.ReadFile(".sops.yaml")
	if strings.Contains(sopsContent, secondPublicKey) {
		t.Error("Second key still in .sops.yaml after removal")
	}

	// Verify the config file was updated
	configContent = env.ReadFile("dev/api.yml")
	if strings.Contains(configContent, secondPublicKey) {
		t.Error("Second key still in encrypted file after removal")
	}

	// Verify we can still decrypt with the original key
	env.Get("SECRET_KEY", "-a", "api", "-e", "dev").
		AssertSuccess().
		AssertStdoutEquals("my-secret")

	// Verify we cannot decrypt with the removed key
	result = env.RunWithEnv(map[string]string{"SOPS_AGE_KEY": secondSecretKey}, "get", "-k", "SECRET_KEY", "-a", "api", "-e", "dev", "-r", ".")
	result.AssertFailure()
}

// TestWorkflow_GenerateMultipleFormats tests generating configuration in different formats
func TestWorkflow_GenerateMultipleFormats(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set up configuration
	env.Set("DATABASE_URL", "postgres://localhost/mydb", "-a", "api", "-e", "dev").
		AssertSuccess()
	env.Set("REDIS_URL", "redis://localhost:6379", "-a", "api", "-e", "dev").
		AssertSuccess()
	env.Set("API_PORT", "3000", "-a", "api", "-e", "dev").
		AssertSuccess()

	// Generate as ENV format
	result := env.Generate("api", "dev", "env")
	result.AssertSuccess()
	envOutput := result.GetStdout()
	if !strings.Contains(envOutput, "DATABASE_URL=postgres://localhost/mydb") {
		t.Error("ENV format not generated correctly")
	}

	// Generate as JSON format
	result = env.Generate("api", "dev", "json")
	result.AssertSuccess()
	jsonOutput := result.GetStdout()
	if !strings.Contains(jsonOutput, "\"DATABASE_URL\"") {
		t.Error("JSON format not generated correctly")
	}

	// Generate as YAML format
	result = env.Generate("api", "dev", "yaml")
	result.AssertSuccess()
	yamlOutput := result.GetStdout()
	if !strings.Contains(yamlOutput, "DATABASE_URL: postgres://localhost/mydb") {
		t.Error("YAML format not generated correctly")
	}

	// Generate as K8s secret format
	result = env.Generate("api", "dev", "k8s", "--secret-name", "api-config")
	result.AssertSuccess()
	k8sOutput := result.GetStdout()
	if !strings.Contains(k8sOutput, "kind: Secret") {
		t.Error("K8s format not generated correctly")
	}
	if !strings.Contains(k8sOutput, "name: api-config") {
		t.Error("K8s secret name not set correctly")
	}

	// Generate K8s secret with base64 encoding
	result = env.Generate("api", "dev", "k8s", "--secret-name", "api-config", "--base64")
	result.AssertSuccess()
	k8sBase64Output := result.GetStdout()
	// Base64 values should be different from plain values
	if k8sBase64Output == k8sOutput {
		t.Error("Base64 encoding not applied")
	}
}

// TestWorkflow_ConfigurationPrecedence tests the 6-level precedence system
func TestWorkflow_ConfigurationPrecedence(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Level 1: base/shared - Lowest precedence
	env.Set("LOG_LEVEL", "error", "-a", "shared", "-e", "base").
		AssertSuccess()
	env.Set("MAX_CONNECTIONS", "100", "-a", "shared", "-e", "base").
		AssertSuccess()

	// Level 2: base/app
	env.Set("LOG_LEVEL", "warn", "-a", "api", "-e", "base").
		AssertSuccess()

	// Level 3: env/shared
	env.Set("LOG_LEVEL", "info", "-a", "shared", "-e", "dev").
		AssertSuccess()

	// Level 4: env/app
	env.Set("LOG_LEVEL", "debug", "-a", "api", "-e", "dev").
		AssertSuccess()

	// Level 5: target/shared
	env.Set("MAX_CONNECTIONS", "50", "-a", "shared", "-e", "dev", "-t", "local").
		AssertSuccess()

	// Level 6: target/app - Highest precedence
	env.Set("LOG_LEVEL", "trace", "-a", "api", "-e", "dev", "-t", "local").
		AssertSuccess()

	// Generate without target - should use up to level 4
	result := env.Generate("api", "dev", "json")
	result.AssertSuccess()
	output := result.GetStdout()

	if !strings.Contains(output, "\"LOG_LEVEL\":\"debug\"") && !strings.Contains(output, "\"LOG_LEVEL\": \"debug\"") {
		t.Errorf("Level 4 (env/app) not applied correctly. Output: %s", output)
	}
	if !strings.Contains(output, "\"MAX_CONNECTIONS\":\"100\"") && !strings.Contains(output, "\"MAX_CONNECTIONS\": \"100\"") {
		t.Errorf("Level 1 (base/shared) not applied for MAX_CONNECTIONS. Output: %s", output)
	}

	// Generate with target - should use all 6 levels
	result = env.Generate("api", "dev", "json", "-t", "local")
	result.AssertSuccess()
	output = result.GetStdout()

	if !strings.Contains(output, "\"LOG_LEVEL\":\"trace\"") && !strings.Contains(output, "\"LOG_LEVEL\": \"trace\"") {
		t.Errorf("Level 6 (target/app) not applied correctly. Output: %s", output)
	}
	if !strings.Contains(output, "\"MAX_CONNECTIONS\":\"50\"") && !strings.Contains(output, "\"MAX_CONNECTIONS\": \"50\"") {
		t.Errorf("Level 5 (target/shared) not applied for MAX_CONNECTIONS. Output: %s", output)
	}
}
