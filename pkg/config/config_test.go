package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary test directory
	tmpDir, err := os.MkdirTemp("", "puff-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Setup test files
	baseDir := filepath.Join(tmpDir, "base")
	os.MkdirAll(baseDir, 0755)

	devDir := filepath.Join(tmpDir, "dev")
	os.MkdirAll(devDir, 0755)

	// Write test files
	os.WriteFile(filepath.Join(baseDir, "shared.yml"), []byte("GLOBAL: base_shared\nOVERRIDE: base_shared"), 0644)
	os.WriteFile(filepath.Join(baseDir, "api.yml"), []byte("APP_NAME: api\nOVERRIDE: base_api"), 0644)
	os.WriteFile(filepath.Join(devDir, "shared.yml"), []byte("ENV: dev\nOVERRIDE: dev_shared"), 0644)
	os.WriteFile(filepath.Join(devDir, "api.yml"), []byte("ENV_APP: dev_api\nOVERRIDE: dev_api"), 0644)

	// Test loading with different contexts
	tests := []struct {
		name     string
		ctx      LoadContext
		expected map[string]string
	}{
		{
			name: "base only",
			ctx: LoadContext{
				RootDir: tmpDir,
			},
			expected: map[string]string{
				"GLOBAL":   "base_shared",
				"OVERRIDE": "base_shared",
			},
		},
		{
			name: "base and app",
			ctx: LoadContext{
				RootDir: tmpDir,
				App:     "api",
			},
			expected: map[string]string{
				"GLOBAL":   "base_shared",
				"APP_NAME": "api",
				"OVERRIDE": "base_api",
			},
		},
		{
			name: "base, env, and app",
			ctx: LoadContext{
				RootDir: tmpDir,
				App:     "api",
				Env:     "dev",
			},
			expected: map[string]string{
				"GLOBAL":   "base_shared",
				"APP_NAME": "api",
				"ENV":      "dev",
				"ENV_APP":  "dev_api",
				"OVERRIDE": "dev_api",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(tt.ctx)
			if err != nil {
				t.Fatalf("Load failed: %v", err)
			}

			for key, expectedValue := range tt.expected {
				if value, ok := cfg.GetString(key); !ok || value != expectedValue {
					t.Errorf("Key %s: expected %s, got %s (exists: %v)", key, expectedValue, value, ok)
				}
			}
		})
	}
}

func TestMerge(t *testing.T) {
	cfg := New()

	// First merge
	cfg.merge(map[string]interface{}{
		"KEY1": "value1",
		"KEY2": "value2",
	})

	// Second merge should override
	cfg.merge(map[string]interface{}{
		"KEY2": "new_value2",
		"KEY3": "value3",
	})

	tests := []struct {
		key      string
		expected string
	}{
		{"KEY1", "value1"},
		{"KEY2", "new_value2"},
		{"KEY3", "value3"},
	}

	for _, tt := range tests {
		if value, ok := cfg.GetString(tt.key); !ok || value != tt.expected {
			t.Errorf("Key %s: expected %s, got %s (exists: %v)", tt.key, tt.expected, value, ok)
		}
	}
}

func TestExportKeys(t *testing.T) {
	cfg := New()
	cfg.Set("PUBLIC_VAR", "public")
	cfg.Set("_PRIVATE_VAR", "private")
	cfg.Set("ANOTHER_PUBLIC", "public2")

	exportKeys := cfg.ExportKeys()

	// Check that private var is not in export keys
	for _, key := range exportKeys {
		if key == "_PRIVATE_VAR" {
			t.Error("Private variable should not be in export keys")
		}
	}

	// Check that public vars are in export keys
	foundPublic := 0
	for _, key := range exportKeys {
		if key == "PUBLIC_VAR" || key == "ANOTHER_PUBLIC" {
			foundPublic++
		}
	}

	if foundPublic != 2 {
		t.Errorf("Expected 2 public variables in export keys, got %d", foundPublic)
	}
}
