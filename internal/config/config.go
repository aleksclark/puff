package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/getsops/sops/v3/decrypt"
	"gopkg.in/yaml.v3"
)

// Config represents the configuration structure.
// Config is safe for concurrent read access via Get methods,
// but Load() should not be called concurrently.
type Config struct {
	Values map[string]interface{}
	mu     sync.RWMutex
	files  []string // Track which files contributed to this config
}

// LoadContext defines the parameters for loading config
type LoadContext struct {
	RootDir string
	App     string
	Env     string
	Target  string
}

// New creates a new empty Config
func New() *Config {
	return &Config{
		Values: make(map[string]interface{}),
		files:  make([]string, 0),
	}
}

// Load loads and merges configuration files based on the precedence order
// Precedence (lowest to highest):
// 1. base/shared.yml
// 2. base/{app}.yml
// 3. {env}/shared.yml
// 4. {env}/{app}.yml
// 5. target-overrides/{target}/shared.yml
// 6. target-overrides/{target}/{app}.yml
func Load(ctx LoadContext) (*Config, error) {
	cfg := New()

	// Build list of files to load in precedence order
	filesToLoad := []string{}

	// 1. base/shared.yml
	filesToLoad = append(filesToLoad, filepath.Join(ctx.RootDir, "base", "shared.yml"))

	// 2. base/{app}.yml
	if ctx.App != "" {
		filesToLoad = append(filesToLoad, filepath.Join(ctx.RootDir, "base", fmt.Sprintf("%s.yml", ctx.App)))
	}

	// 3. {env}/shared.yml
	if ctx.Env != "" {
		filesToLoad = append(filesToLoad, filepath.Join(ctx.RootDir, ctx.Env, "shared.yml"))
	}

	// 4. {env}/{app}.yml
	if ctx.Env != "" && ctx.App != "" {
		filesToLoad = append(filesToLoad, filepath.Join(ctx.RootDir, ctx.Env, fmt.Sprintf("%s.yml", ctx.App)))
	}

	// 5. target-overrides/{target}/{env}/shared.yml
	if ctx.Target != "" {
		targetEnv := ctx.Env
		if targetEnv == "" {
			targetEnv = "base"
		}
		filesToLoad = append(filesToLoad, filepath.Join(ctx.RootDir, "target-overrides", ctx.Target, targetEnv, "shared.yml"))
	}

	// 6. target-overrides/{target}/{env}/{app}.yml
	if ctx.Target != "" && ctx.App != "" {
		targetEnv := ctx.Env
		if targetEnv == "" {
			targetEnv = "base"
		}
		filesToLoad = append(filesToLoad, filepath.Join(ctx.RootDir, "target-overrides", ctx.Target, targetEnv, fmt.Sprintf("%s.yml", ctx.App)))
	}

	// Load and merge each file
	for _, file := range filesToLoad {
		if err := cfg.loadFile(file); err != nil {
			// If file doesn't exist, that's okay - just skip it
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("error loading %s: %w", file, err)
			}
		}
	}

	return cfg, nil
}

// loadFile loads a single YAML file and merges it into the config
// If the file is SOPS-encrypted, it will be decrypted automatically
func (c *Config) loadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Try to detect and decrypt SOPS-encrypted files
	// SOPS files contain "sops:" in the YAML structure
	if isSopsEncrypted(data) {
		decrypted, err := decrypt.File(path, "yaml")
		if err != nil {
			return fmt.Errorf("error decrypting SOPS file %s: %w", path, err)
		}
		data = decrypted
	}

	// Parse YAML once
	var values map[string]interface{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return fmt.Errorf("error parsing YAML in %s: %w", path, err)
	}

	// Remove the 'sops' metadata key if it exists (shouldn't be merged into config)
	delete(values, "sops")

	// Merge the values
	c.merge(values)

	// Protect files slice access with mutex
	c.mu.Lock()
	c.files = append(c.files, path)
	c.mu.Unlock()

	return nil
}

// isSopsEncrypted checks if data contains SOPS metadata using a simple heuristic
func isSopsEncrypted(data []byte) bool {
	// Quick check: look for "sops:" string in the YAML
	// This avoids parsing YAML twice for non-encrypted files
	return len(data) > 0 && (
		// Common SOPS patterns
		contains(data, []byte("\nsops:")) ||
		contains(data, []byte("sops:")))
}

// contains checks if slice contains subslice
func contains(data, subslice []byte) bool {
	if len(subslice) == 0 {
		return true
	}
	if len(subslice) > len(data) {
		return false
	}
	for i := 0; i <= len(data)-len(subslice); i++ {
		match := true
		for j := 0; j < len(subslice); j++ {
			if data[i+j] != subslice[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// merge performs a deep merge of new values into the existing config.
// Values from 'new' override values in the existing config, but nested
// maps are recursively merged rather than replaced.
func (c *Config) merge(new map[string]interface{}) {
	for key, value := range new {
		if existing, exists := c.Values[key]; exists {
			// Recursively merge nested maps
			if existingMap, ok := existing.(map[string]interface{}); ok {
				if newMap, ok := value.(map[string]interface{}); ok {
					c.mergeMap(existingMap, newMap)
					continue
				}
			}
		}
		c.Values[key] = value
	}
}

// mergeMap recursively merges two maps
func (c *Config) mergeMap(existing, new map[string]interface{}) {
	for key, value := range new {
		if existingVal, exists := existing[key]; exists {
			if existingNested, ok := existingVal.(map[string]interface{}); ok {
				if newNested, ok := value.(map[string]interface{}); ok {
					c.mergeMap(existingNested, newNested)
					continue
				}
			}
		}
		existing[key] = value
	}
}

// Get retrieves a value from the config
func (c *Config) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.Values[key]
	return val, ok
}

// Set sets a value in the config
func (c *Config) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Values[key] = value
}

// GetString retrieves a string value from the config
func (c *Config) GetString(key string) (string, bool) {
	val, ok := c.Get(key)
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// Keys returns all keys in the config (excluding underscore-prefixed internal variables)
func (c *Config) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]string, 0, len(c.Values))
	for key := range c.Values {
		keys = append(keys, key)
	}
	return keys
}

// ExportKeys returns all keys that should be exported (excluding underscore-prefixed internal variables)
func (c *Config) ExportKeys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]string, 0)
	for key := range c.Values {
		if len(key) > 0 && key[0] != '_' {
			keys = append(keys, key)
		}
	}
	return keys
}

// Files returns the list of files that contributed to this config
func (c *Config) Files() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Return a copy to prevent external modification
	return append([]string(nil), c.files...)
}
