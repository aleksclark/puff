package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/teamcurri/puff/test/helpers"
)

// TestSecurity_FilesAlwaysEncrypted verifies all config files are encrypted
func TestSecurity_FilesAlwaysEncrypted(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set values in various locations
	env.Set("SECRET_KEY", "sensitive-data", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("DB_PASSWORD", "secret123", "-a", "api", "-e", "prod").AssertSuccess()
	env.Set("API_KEY", "key123", "-a", "shared", "-e", "base").AssertSuccess()
	env.Set("TOKEN", "token456", "-a", "api", "-e", "dev", "-t", "local").AssertSuccess()

	// Check all created files are encrypted
	filesToCheck := []string{
		"dev/api.yml",
		"prod/api.yml",
		"base/shared.yml",
		"target-overrides/local/dev/api.yml",
	}

	for _, file := range filesToCheck {
		if !env.FileExists(file) {
			t.Errorf("Expected file %s was not created", file)
			continue
		}

		content := env.ReadFile(file)

		// Verify file contains SOPS metadata
		if !strings.Contains(content, "sops:") {
			t.Errorf("File %s is not encrypted (missing sops metadata)", file)
		}

		// Verify file contains age encryption
		if !strings.Contains(content, "age:") {
			t.Errorf("File %s does not use age encryption", file)
		}

		// Verify the plaintext secret is NOT in the file
		if strings.Contains(content, "sensitive-data") {
			t.Errorf("File %s contains plaintext secret 'sensitive-data'", file)
		}
		if strings.Contains(content, "secret123") {
			t.Errorf("File %s contains plaintext secret 'secret123'", file)
		}

		// Verify keys are shown encrypted (ENC[...])
		if !strings.Contains(content, "ENC[") {
			t.Errorf("File %s does not contain encrypted values (ENC[...])", file)
		}
	}
}

// TestSecurity_FilePermissions verifies encrypted files have restrictive permissions
func TestSecurity_FilePermissions(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Check .sops.yaml permissions
	sopsPath := filepath.Join(env.Dir, ".sops.yaml")
	info, err := os.Stat(sopsPath)
	if err != nil {
		t.Fatalf("Failed to stat .sops.yaml: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf(".sops.yaml has incorrect permissions: got %o, want 0600", perm)
	}

	// Create encrypted config file
	env.Set("SECRET", "value", "-a", "api", "-e", "dev").AssertSuccess()

	// Check encrypted file permissions
	configPath := filepath.Join(env.Dir, "dev/api.yml")
	info, err = os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat dev/api.yml: %v", err)
	}

	perm = info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("Encrypted file has incorrect permissions: got %o, want 0600", perm)
	}
}

// TestSecurity_DecryptedFilesAreTemporary verifies .dec files are removed after encryption
func TestSecurity_DecryptedFilesAreTemporary(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()
	env.Set("SECRET", "sensitive-value", "-a", "api", "-e", "dev").AssertSuccess()

	configFile := "dev/api.yml"
	decFile := "dev/api.dec.yml"

	// Decrypt the file
	env.Decrypt(configFile).AssertSuccess()

	// Verify .dec file exists
	if !env.FileExists(decFile) {
		t.Fatal("Decrypted file was not created")
	}

	// Verify .dec file contains plaintext
	decContent := env.ReadFile(decFile)
	if !strings.Contains(decContent, "sensitive-value") {
		t.Error("Decrypted file does not contain plaintext value")
	}
	if strings.Contains(decContent, "sops:") {
		t.Error("Decrypted file still contains SOPS metadata")
	}

	// Re-encrypt the file
	env.Encrypt(decFile).AssertSuccess()

	// Verify .dec file was removed
	if env.FileExists(decFile) {
		t.Error("Decrypted file still exists after encryption - security risk!")
	}

	// Verify encrypted file still exists and is encrypted
	if !env.FileExists(configFile) {
		t.Fatal("Encrypted file was removed")
	}

	encContent := env.ReadFile(configFile)
	if !strings.Contains(encContent, "sops:") {
		t.Error("File was not re-encrypted properly")
	}
}

// TestSecurity_KeyIsolation verifies keys can only decrypt their own files
func TestSecurity_KeyIsolation(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	// Initialize with first key
	env.Init().AssertSuccess()
	firstKey := env.AgeSecretKey

	// Set a secret with the first key
	env.Set("SECRET1", "value1", "-a", "api", "-e", "dev").AssertSuccess()

	// Generate a second key
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
		t.Fatal("Failed to generate second key")
	}

	// Try to decrypt with wrong key
	result = env.RunWithEnv(map[string]string{"SOPS_AGE_KEY": secondSecretKey}, "get", "-k", "SECRET1", "-a", "api", "-e", "dev", "-r", ".")
	result.AssertFailure()

	// Add the second key
	env.KeysAdd(secondPublicKey, "Second key").AssertSuccess()

	// Now decryption should work with second key
	result = env.RunWithEnv(map[string]string{"SOPS_AGE_KEY": secondSecretKey}, "get", "-k", "SECRET1", "-a", "api", "-e", "dev", "-r", ".")
	result.AssertSuccess().AssertStdoutEquals("value1")

	// Remove the first key (keep only second key)
	// This should fail because we need at least one key
	result = env.KeysRemove(env.AgeKey)
	result.AssertFailure()

	// Verify original key still works
	result = env.RunWithEnv(map[string]string{"SOPS_AGE_KEY": firstKey}, "get", "-k", "SECRET1", "-a", "api", "-e", "dev", "-r", ".")
	result.AssertSuccess().AssertStdoutEquals("value1")
}

// TestSecurity_KeyRotation verifies secure key rotation workflow
func TestSecurity_KeyRotation(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()
	oldKey := env.AgeKey
	oldSecretKey := env.AgeSecretKey

	// Set up secrets with old key
	env.Set("SECRET1", "sensitive1", "-a", "api", "-e", "dev").AssertSuccess()
	env.Set("SECRET2", "sensitive2", "-a", "api", "-e", "prod").AssertSuccess()

	// Generate new key
	result := env.RunSystem("age-keygen")
	if result.ExitCode != 0 {
		t.Fatalf("age-keygen failed: %s", result.GetStderr())
	}

	output := result.GetStdout()
	lines := strings.Split(output, "\n")
	var newPublicKey, newSecretKey string
	for _, line := range lines {
		if strings.HasPrefix(line, "# public key:") {
			newPublicKey = strings.TrimSpace(strings.TrimPrefix(line, "# public key:"))
		} else if strings.HasPrefix(line, "AGE-SECRET-KEY-") {
			newSecretKey = strings.TrimSpace(line)
		}
	}

	// Add new key (now both keys can decrypt)
	env.KeysAdd(newPublicKey, "New rotated key").AssertSuccess()

	// Verify both keys can decrypt
	result = env.RunWithEnv(map[string]string{"SOPS_AGE_KEY": oldSecretKey}, "get", "-k", "SECRET1", "-a", "api", "-e", "dev", "-r", ".")
	result.AssertSuccess()

	result = env.RunWithEnv(map[string]string{"SOPS_AGE_KEY": newSecretKey}, "get", "-k", "SECRET1", "-a", "api", "-e", "dev", "-r", ".")
	result.AssertSuccess()

	// Remove old key (complete rotation)
	// Use the new key for this operation since the old key is being removed
	env.RunWithEnv(map[string]string{"SOPS_AGE_KEY": newSecretKey}, "keys", "rm", "-k", oldKey, "-r", ".").AssertSuccess()

	// Verify old key can no longer decrypt
	result = env.RunWithEnv(map[string]string{"SOPS_AGE_KEY": oldSecretKey}, "get", "-k", "SECRET1", "-a", "api", "-e", "dev", "-r", ".")
	result.AssertFailure()

	// Verify new key still works
	result = env.RunWithEnv(map[string]string{"SOPS_AGE_KEY": newSecretKey}, "get", "-k", "SECRET1", "-a", "api", "-e", "dev", "-r", ".")
	result.AssertSuccess().AssertStdoutEquals("sensitive1")
}

// TestSecurity_NoPlaintextInGeneratedK8sSecrets verifies k8s secrets don't leak plaintext
func TestSecurity_NoPlaintextInGeneratedK8sSecrets(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set sensitive values
	env.Set("DB_PASSWORD", "super-secret-password", "-a", "api", "-e", "prod").AssertSuccess()
	env.Set("API_KEY", "secret-api-key-12345", "-a", "api", "-e", "prod").AssertSuccess()

	// Generate k8s secret WITHOUT base64
	result := env.Generate("api", "prod", "k8s", "--secret-name", "api-secrets")
	result.AssertSuccess()

	output := result.GetStdout()

	// This is expected behavior - k8s secrets contain plaintext in stringData
	// But verify they are marked as Secret type
	if !strings.Contains(output, "kind: Secret") {
		t.Error("Generated output is not a Kubernetes Secret")
	}

	// Verify secrets are in stringData (not data)
	if !strings.Contains(output, "stringData:") {
		t.Error("k8s secret should use stringData for unencoded values")
	}

	// Generate k8s secret WITH base64
	result = env.Generate("api", "prod", "k8s", "--secret-name", "api-secrets", "--base64")
	result.AssertSuccess()

	output = result.GetStdout()

	// Verify secrets are base64 encoded
	if !strings.Contains(output, "data:") {
		t.Error("k8s secret with --base64 should use 'data' field")
	}

	// Verify plaintext is not in output (should be base64 encoded)
	if strings.Contains(output, "super-secret-password") {
		t.Error("Plaintext password found in base64-encoded k8s secret")
	}
	if strings.Contains(output, "secret-api-key-12345") {
		t.Error("Plaintext API key found in base64-encoded k8s secret")
	}
}

// TestSecurity_EnvironmentVariableLeakage verifies internal vars don't leak
func TestSecurity_EnvironmentVariableLeakage(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set internal variables with sensitive data
	env.Set("_DB_PASSWORD", "super-secret-db-pass", "-a", "api", "-e", "prod").AssertSuccess()
	env.Set("_ADMIN_TOKEN", "admin-token-xyz", "-a", "api", "-e", "prod").AssertSuccess()
	env.Set("_INTERNAL_KEY", "internal-secret", "-a", "api", "-e", "prod").AssertSuccess()

	// Set public variables
	env.Set("PUBLIC_VAR", "public-value", "-a", "api", "-e", "prod").AssertSuccess()

	// Generate in all formats and verify internal vars are not exposed
	formats := []string{"env", "json", "yaml"}

	for _, format := range formats {
		result := env.Generate("api", "prod", format)
		result.AssertSuccess()

		output := result.GetStdout()

		// Verify internal variables are NOT in output
		if strings.Contains(output, "_DB_PASSWORD") {
			t.Errorf("Internal variable _DB_PASSWORD leaked in %s format", format)
		}
		if strings.Contains(output, "_ADMIN_TOKEN") {
			t.Errorf("Internal variable _ADMIN_TOKEN leaked in %s format", format)
		}
		if strings.Contains(output, "_INTERNAL_KEY") {
			t.Errorf("Internal variable _INTERNAL_KEY leaked in %s format", format)
		}

		// Verify public variable IS in output
		if !strings.Contains(output, "PUBLIC_VAR") {
			t.Errorf("Public variable PUBLIC_VAR not found in %s format", format)
		}
	}
}

// TestSecurity_SOPSMetadataNotInGeneratedOutput verifies SOPS metadata doesn't leak
func TestSecurity_SOPSMetadataNotInGeneratedOutput(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()
	env.Set("KEY", "value", "-a", "api", "-e", "dev").AssertSuccess()

	// Generate configuration
	result := env.Generate("api", "dev", "json")
	result.AssertSuccess()

	output := result.GetStdout()

	// Verify SOPS metadata is not in generated output
	if strings.Contains(output, "sops") {
		t.Error("SOPS metadata leaked into generated output")
	}
	if strings.Contains(output, "age") {
		t.Error("Age encryption metadata leaked into generated output")
	}
	if strings.Contains(output, "ENC[") {
		t.Error("Encrypted value markers leaked into generated output")
	}
	if strings.Contains(output, env.AgeKey) {
		t.Error("Age public key leaked into generated output")
	}
}

// TestSecurity_KeyCommentsDoNotContainSecrets verifies key comments are safe
func TestSecurity_KeyCommentsDoNotContainSecrets(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Add a key with a comment
	result := env.RunSystem("age-keygen")
	if result.ExitCode != 0 {
		t.Fatalf("age-keygen failed: %s", result.GetStderr())
	}

	output := result.GetStdout()
	lines := strings.Split(output, "\n")
	var newPublicKey string
	for _, line := range lines {
		if strings.HasPrefix(line, "# public key:") {
			newPublicKey = strings.TrimSpace(strings.TrimPrefix(line, "# public key:"))
			break
		}
	}

	// Add key with a descriptive comment
	env.KeysAdd(newPublicKey, "Production deployment key for team-alpha").AssertSuccess()

	// Verify comment is stored
	sopsContent := env.ReadFile(".sops.yaml")
	if !strings.Contains(sopsContent, "Production deployment key for team-alpha") {
		t.Error("Key comment not stored in .sops.yaml")
	}

	// Set a secret value
	env.Set("SECRET", "sensitive-data", "-a", "api", "-e", "prod").AssertSuccess()

	// Verify the comment does not appear in encrypted files
	configContent := env.ReadFile("prod/api.yml")
	if strings.Contains(configContent, "Production deployment key for team-alpha") {
		t.Error("Key comment should not appear in encrypted config files")
	}

	// Verify the comment does not appear in generated output
	result = env.Generate("api", "prod", "json")
	result.AssertSuccess()

	genOutput := result.GetStdout()
	if strings.Contains(genOutput, "Production deployment key for team-alpha") {
		t.Error("Key comment leaked into generated output")
	}
}

// TestSecurity_CannotReadWithoutCorrectKey verifies access control
func TestSecurity_CannotReadWithoutCorrectKey(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup()

	env.Init().AssertSuccess()

	// Set a secret
	env.Set("TOP_SECRET", "classified-information", "-a", "api", "-e", "prod").AssertSuccess()

	// Generate a different age key (attacker's key)
	result := env.RunSystem("age-keygen")
	if result.ExitCode != 0 {
		t.Fatalf("age-keygen failed: %s", result.GetStderr())
	}

	output := result.GetStdout()
	lines := strings.Split(output, "\n")
	var attackerSecretKey string
	for _, line := range lines {
		if strings.HasPrefix(line, "AGE-SECRET-KEY-") {
			attackerSecretKey = strings.TrimSpace(line)
			break
		}
	}

	// Try to read with attacker's key
	result = env.RunWithEnv(map[string]string{"SOPS_AGE_KEY": attackerSecretKey}, "get", "-k", "TOP_SECRET", "-a", "api", "-e", "prod", "-r", ".")
	result.AssertFailure()

	// Try to generate with attacker's key
	result = env.RunWithEnv(map[string]string{"SOPS_AGE_KEY": attackerSecretKey}, "generate", "-a", "api", "-e", "prod", "-f", "json", "-r", ".")
	result.AssertFailure()

	// Verify the correct key can still read
	env.Get("TOP_SECRET", "-a", "api", "-e", "prod").
		AssertSuccess().
		AssertStdoutEquals("classified-information")
}
