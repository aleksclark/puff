package helpers

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnv represents a test environment with isolated directory and age keys
type TestEnv struct {
	t            *testing.T
	Dir          string
	AgeKey       string
	AgeSecretKey string
	PuffBinary   string
}

// NewTestEnv creates a new isolated test environment
func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "puff-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Generate age key for this test
	cmd := exec.Command("age-keygen")
	output, err := cmd.Output()
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to generate age key: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	var secretKey, publicKey string
	for _, line := range lines {
		if strings.HasPrefix(line, "# public key:") {
			publicKey = strings.TrimSpace(strings.TrimPrefix(line, "# public key:"))
		} else if strings.HasPrefix(line, "AGE-SECRET-KEY-") {
			secretKey = strings.TrimSpace(line)
		}
	}

	if publicKey == "" || secretKey == "" {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to extract keys from age-keygen output")
	}

	// Find puff binary - search in common locations
	// When running 'go test ./test', working dir is project root
	// When running from within test dir, need to go up one level
	possiblePaths := []string{
		"bin/puff",                    // From project root
		"../bin/puff",                 // From test/ dir
		filepath.Join("..", "..", "bin", "puff"), // From test/helpers/ (shouldn't happen but just in case)
	}

	var puffBinary string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			absPath, err := filepath.Abs(path)
			if err == nil {
				puffBinary = absPath
				break
			}
		}
	}

	if puffBinary == "" {
		os.RemoveAll(tmpDir)
		t.Fatalf("Puff binary not found in any of: %v - run 'go build -o bin/puff ./cmd/puff' first", possiblePaths)
	}

	return &TestEnv{
		t:            t,
		Dir:          tmpDir,
		AgeKey:       publicKey,
		AgeSecretKey: secretKey,
		PuffBinary:   puffBinary,
	}
}

// Cleanup removes the test environment directory
func (e *TestEnv) Cleanup() {
	e.t.Helper()
	if err := os.RemoveAll(e.Dir); err != nil {
		e.t.Logf("Warning: Failed to cleanup test dir %s: %v", e.Dir, err)
	}
}

// Run executes a puff command in the test environment
func (e *TestEnv) Run(args ...string) *CommandResult {
	e.t.Helper()
	return e.RunWithEnv(nil, args...)
}

// RunSystem executes a system command (not puff) in the test environment
func (e *TestEnv) RunSystem(command string, args ...string) *CommandResult {
	e.t.Helper()

	cmd := exec.Command(command, args...)
	cmd.Dir = e.Dir

	// Set environment
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("SOPS_AGE_KEY=%s", e.AgeSecretKey))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return &CommandResult{
		t:        e.t,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Err:      err,
	}
}

// RunWithEnv executes a puff command with custom environment variables
func (e *TestEnv) RunWithEnv(env map[string]string, args ...string) *CommandResult {
	e.t.Helper()

	cmd := exec.Command(e.PuffBinary, args...)
	cmd.Dir = e.Dir

	// Set environment
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("SOPS_AGE_KEY=%s", e.AgeSecretKey))
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return &CommandResult{
		t:        e.t,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Err:      err,
	}
}

// Init initializes a puff directory with encryption
func (e *TestEnv) Init() *CommandResult {
	e.t.Helper()
	return e.Run("init", "-d", ".", "-k", e.AgeKey)
}

// Set sets a configuration value
func (e *TestEnv) Set(key, value string, opts ...string) *CommandResult {
	e.t.Helper()
	args := []string{"set", "-k", key, "-v", value, "-r", "."}
	args = append(args, opts...)
	return e.Run(args...)
}

// Get retrieves a configuration value
func (e *TestEnv) Get(key string, opts ...string) *CommandResult {
	e.t.Helper()
	args := []string{"get", "-k", key, "-r", "."}
	args = append(args, opts...)
	return e.Run(args...)
}

// Generate generates configuration in the specified format
func (e *TestEnv) Generate(app, env, format string, opts ...string) *CommandResult {
	e.t.Helper()
	args := []string{"generate", "-a", app, "-e", env, "-f", format, "-r", "."}
	args = append(args, opts...)
	return e.Run(args...)
}

// KeysAdd adds an encryption key
func (e *TestEnv) KeysAdd(key, comment string, opts ...string) *CommandResult {
	e.t.Helper()
	args := []string{"keys", "add", "-k", key, "-c", comment, "-r", "."}
	args = append(args, opts...)
	return e.Run(args...)
}

// KeysRemove removes an encryption key
func (e *TestEnv) KeysRemove(key string, opts ...string) *CommandResult {
	e.t.Helper()
	args := []string{"keys", "rm", "-k", key, "-r", "."}
	args = append(args, opts...)
	return e.Run(args...)
}

// KeysList lists encryption keys
func (e *TestEnv) KeysList(opts ...string) *CommandResult {
	e.t.Helper()
	args := []string{"keys", "list", "-r", "."}
	args = append(args, opts...)
	return e.Run(args...)
}

// Decrypt decrypts a file for bulk editing
func (e *TestEnv) Decrypt(file string) *CommandResult {
	e.t.Helper()
	return e.Run("decrypt", "-f", file)
}

// Encrypt encrypts a decrypted file
func (e *TestEnv) Encrypt(file string) *CommandResult {
	e.t.Helper()
	return e.Run("encrypt", "-f", file)
}

// ReadFile reads a file from the test environment
func (e *TestEnv) ReadFile(path string) string {
	e.t.Helper()
	content, err := os.ReadFile(filepath.Join(e.Dir, path))
	if err != nil {
		e.t.Fatalf("Failed to read file %s: %v", path, err)
	}
	return string(content)
}

// WriteFile writes a file to the test environment
func (e *TestEnv) WriteFile(path, content string) {
	e.t.Helper()
	fullPath := filepath.Join(e.Dir, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		e.t.Fatalf("Failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		e.t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

// FileExists checks if a file exists in the test environment
func (e *TestEnv) FileExists(path string) bool {
	e.t.Helper()
	_, err := os.Stat(filepath.Join(e.Dir, path))
	return err == nil
}

// MkdirAll creates directories in the test environment
func (e *TestEnv) MkdirAll(path string) {
	e.t.Helper()
	if err := os.MkdirAll(filepath.Join(e.Dir, path), 0755); err != nil {
		e.t.Fatalf("Failed to create directory %s: %v", path, err)
	}
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	t        *testing.T
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// AssertSuccess asserts the command succeeded (exit code 0)
func (r *CommandResult) AssertSuccess() *CommandResult {
	r.t.Helper()
	if r.ExitCode != 0 {
		r.t.Fatalf("Command failed with exit code %d\nStdout: %s\nStderr: %s\nError: %v",
			r.ExitCode, r.Stdout, r.Stderr, r.Err)
	}
	return r
}

// AssertFailure asserts the command failed (non-zero exit code)
func (r *CommandResult) AssertFailure() *CommandResult {
	r.t.Helper()
	if r.ExitCode == 0 {
		r.t.Fatalf("Expected command to fail, but it succeeded\nStdout: %s\nStderr: %s",
			r.Stdout, r.Stderr)
	}
	return r
}

// AssertStdoutContains asserts stdout contains the given string
func (r *CommandResult) AssertStdoutContains(substr string) *CommandResult {
	r.t.Helper()
	if !strings.Contains(r.Stdout, substr) {
		r.t.Fatalf("Expected stdout to contain %q, but it didn't.\nStdout: %s\nStderr: %s",
			substr, r.Stdout, r.Stderr)
	}
	return r
}

// AssertStdoutEquals asserts stdout equals the given string (with trimmed whitespace)
func (r *CommandResult) AssertStdoutEquals(expected string) *CommandResult {
	r.t.Helper()
	actual := strings.TrimSpace(r.Stdout)
	expected = strings.TrimSpace(expected)
	if actual != expected {
		r.t.Fatalf("Expected stdout to equal %q, but got %q.\nStderr: %s",
			expected, actual, r.Stderr)
	}
	return r
}

// AssertStderrContains asserts stderr (or stdout for colored output) contains the given string
func (r *CommandResult) AssertStderrContains(substr string) *CommandResult {
	r.t.Helper()
	// Check both stderr and stdout since colored error output often goes to stdout
	if !strings.Contains(r.Stderr, substr) && !strings.Contains(r.Stdout, substr) {
		r.t.Fatalf("Expected stderr/stdout to contain %q, but it didn't.\nStdout: %s\nStderr: %s",
			substr, r.Stdout, r.Stderr)
	}
	return r
}

// AssertStdoutNotContains asserts stdout does NOT contain the given string
func (r *CommandResult) AssertStdoutNotContains(substr string) *CommandResult {
	r.t.Helper()
	if strings.Contains(r.Stdout, substr) {
		r.t.Fatalf("Expected stdout NOT to contain %q, but it did.\nStdout: %s",
			substr, r.Stdout)
	}
	return r
}

// GetStdout returns the stdout string
func (r *CommandResult) GetStdout() string {
	return strings.TrimSpace(r.Stdout)
}

// GetStderr returns the stderr string
func (r *CommandResult) GetStderr() string {
	return strings.TrimSpace(r.Stderr)
}
