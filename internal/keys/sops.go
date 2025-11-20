package keys

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/age"
	"github.com/getsops/sops/v3/aes"
	"github.com/getsops/sops/v3/cmd/sops/common"
	"github.com/getsops/sops/v3/keyservice"
	sopsyaml "github.com/getsops/sops/v3/stores/yaml"
	"gopkg.in/yaml.v3"
)

// KeyInfo holds information about an encryption key
type KeyInfo struct {
	Key     string
	Comment string
	Envs    []string // Environments this key has access to
}

// EncryptFile encrypts a YAML file using SOPS with the specified age keys
func EncryptFile(filePath string, ageKeys []string) error {
	// Read the plain file
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Load plain YAML into SOPS tree
	store := sopsyaml.Store{}
	branches, err := store.LoadPlainFile(fileBytes)
	if err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Create age master keys from recipients
	var ageMasterKeys []age.MasterKey
	for _, key := range ageKeys {
		masterKey, err := age.MasterKeyFromRecipient(key)
		if err != nil {
			return fmt.Errorf("failed to create master key from recipient %s: %w", key, err)
		}
		ageMasterKeys = append(ageMasterKeys, *masterKey)
	}

	// Build KeyGroups for metadata
	var keyGroups []sops.KeyGroup
	keyGroup := sops.KeyGroup{}
	for i := range ageMasterKeys {
		keyGroup = append(keyGroup, &ageMasterKeys[i])
	}
	keyGroups = append(keyGroups, keyGroup)

	// Create tree with metadata
	tree := sops.Tree{
		Branches: branches,
		Metadata: sops.Metadata{
			KeyGroups:      keyGroups,
			UnencryptedSuffix: "_unencrypted",
			EncryptedSuffix:   "",
			Version:           "3.9.0",
		},
		FilePath: filePath,
	}

	// Generate data key
	dataKey, errs := tree.GenerateDataKeyWithKeyServices(
		[]keyservice.KeyServiceClient{
			keyservice.NewLocalClient(),
		},
	)
	if len(errs) > 0 {
		return fmt.Errorf("failed to generate data key (%d errors)", len(errs))
	}

	// Encrypt the tree
	cipher := aes.NewCipher()
	err = common.EncryptTree(common.EncryptTreeOpts{
		DataKey: dataKey,
		Tree:    &tree,
		Cipher:  cipher,
	})
	if err != nil {
		return fmt.Errorf("failed to encrypt tree: %w", err)
	}

	// Emit encrypted file
	encryptedFile, err := store.EmitEncryptedFile(tree)
	if err != nil {
		return fmt.Errorf("failed to emit encrypted file: %w", err)
	}

	// Ensure directory exists with restrictive permissions
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write encrypted file back with restricted permissions
	err = os.WriteFile(filePath, encryptedFile, 0600)
	if err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	return nil
}


// ListKeys lists all keys from SOPS-encrypted files in the config directory
func ListKeys(rootDir string) ([]KeyInfo, error) {
	keyMap := make(map[string]*KeyInfo)

	// Walk through all .yml files in the config directory
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-yml files
		if info.IsDir() || filepath.Ext(path) != ".yml" {
			return nil
		}

		// Skip .sops.yaml
		if filepath.Base(path) == ".sops.yaml" {
			return nil
		}

		// Try to read SOPS metadata from the file
		data, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		// Parse YAML to extract SOPS metadata
		var yamlData map[string]interface{}
		if err := yaml.Unmarshal(data, &yamlData); err != nil {
			return nil // Skip files that aren't valid YAML
		}

		// Check if this is a SOPS file
		sopsData, ok := yamlData["sops"]
		if !ok {
			return nil // Not a SOPS file, skip
		}

		// Extract age keys from SOPS metadata
		if sopsMap, ok := sopsData.(map[string]interface{}); ok {
			if ageArray, ok := sopsMap["age"].([]interface{}); ok {
				// Determine which env this file belongs to
				relPath, _ := filepath.Rel(rootDir, path)
				env := filepath.Dir(relPath)
				if env == "base" || env == "." {
					env = "base"
				} else if filepath.Dir(env) == "target-overrides" {
					env = fmt.Sprintf("target:%s", filepath.Base(env))
				}

				// Process each age key
				for _, ageEntry := range ageArray {
					if ageMap, ok := ageEntry.(map[string]interface{}); ok {
						recipient, _ := ageMap["recipient"].(string)
						if recipient != "" {
							if _, exists := keyMap[recipient]; !exists {
								keyMap[recipient] = &KeyInfo{
									Key:  recipient,
									Envs: []string{},
								}
							}
							// Add env if not already present
							keyInfo := keyMap[recipient]
							found := false
							for _, e := range keyInfo.Envs {
								if e == env {
									found = true
									break
								}
							}
							if !found {
								keyInfo.Envs = append(keyInfo.Envs, env)
							}
						}
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Convert map to slice
	result := make([]KeyInfo, 0, len(keyMap))
	for _, info := range keyMap {
		result = append(result, *info)
	}

	return result, nil
}

// AddKey adds an age key to all encrypted files, optionally filtering by environment
func AddKey(rootDir, ageKey, comment, env string) error {
	files, err := findEncryptedFiles(rootDir, env)
	if err != nil {
		return fmt.Errorf("failed to find encrypted files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no encrypted files found in %s", rootDir)
	}

	// Validate the age key format
	_, err = age.MasterKeyFromRecipient(ageKey)
	if err != nil {
		return fmt.Errorf("invalid age key: %w", err)
	}

	// Update .sops.yaml with the new key
	if err := AddKeyToSOPSConfig(rootDir, ageKey, comment); err != nil {
		return fmt.Errorf("failed to update .sops.yaml: %w", err)
	}

	// Process each file
	for _, file := range files {
		if err := addKeyToFile(file, ageKey); err != nil {
			return fmt.Errorf("failed to add key to %s: %w", file, err)
		}
	}

	return nil
}

// RemoveKey removes an age key from all encrypted files, optionally filtering by environment
func RemoveKey(rootDir, ageKey, env string) error {
	files, err := findEncryptedFiles(rootDir, env)
	if err != nil {
		return fmt.Errorf("failed to find encrypted files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no encrypted files found in %s", rootDir)
	}

	// Update .sops.yaml to remove the key
	if err := RemoveKeyFromSOPSConfig(rootDir, ageKey); err != nil {
		return fmt.Errorf("failed to update .sops.yaml: %w", err)
	}

	// Process each file
	for _, file := range files {
		if err := removeKeyFromFile(file, ageKey); err != nil {
			return fmt.Errorf("failed to remove key from %s: %w", file, err)
		}
	}

	return nil
}

// findEncryptedFiles finds all SOPS-encrypted YAML files in the directory
func findEncryptedFiles(rootDir, envFilter string) ([]string, error) {
	var files []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-yml files
		if info.IsDir() || filepath.Ext(path) != ".yml" {
			return nil
		}

		// Skip .sops.yaml
		if filepath.Base(path) == ".sops.yaml" {
			return nil
		}

		// If env filter is specified, check if file is in that env
		if envFilter != "" {
			relPath, _ := filepath.Rel(rootDir, path)
			fileEnv := filepath.Dir(relPath)

			// Check if this matches the env filter
			match := false
			if envFilter == "base" && (fileEnv == "base" || fileEnv == ".") {
				match = true
			} else if fileEnv == envFilter {
				match = true
			} else if filepath.Dir(fileEnv) == "target-overrides" && filepath.Base(fileEnv) == envFilter {
				match = true
			}

			if !match {
				return nil
			}
		}

		// Check if file is SOPS-encrypted
		data, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		var yamlData map[string]interface{}
		if err := yaml.Unmarshal(data, &yamlData); err != nil {
			return nil // Skip files that aren't valid YAML
		}

		// Check if this is a SOPS file
		if _, hasSops := yamlData["sops"]; hasSops {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// addKeyToFile adds an age key to a single encrypted file
func addKeyToFile(filePath, recipientKey string) error {
	store := sopsyaml.Store{}

	// Read file and load it properly
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	tree, err := store.LoadEncryptedFile(fileBytes)
	if err != nil {
		return fmt.Errorf("failed to load encrypted file: %w", err)
	}

	// Create new master key
	newMasterKey, err := age.MasterKeyFromRecipient(recipientKey)
	if err != nil {
		return fmt.Errorf("failed to create master key: %w", err)
	}

	// Check if key already exists
	for _, group := range tree.Metadata.KeyGroups {
		for _, key := range group {
			if existingAgeKey, ok := key.(*age.MasterKey); ok {
				if existingAgeKey.Recipient == newMasterKey.Recipient {
					// Key already exists, skip
					return nil
				}
			}
		}
	}

	// Add the new key to the first key group (or create one if none exist)
	if len(tree.Metadata.KeyGroups) == 0 {
		tree.Metadata.KeyGroups = append(tree.Metadata.KeyGroups, sops.KeyGroup{})
	}
	tree.Metadata.KeyGroups[0] = append(tree.Metadata.KeyGroups[0], newMasterKey)

	// Get existing data key
	dataKey, err := tree.Metadata.GetDataKey()
	if err != nil {
		return fmt.Errorf("failed to get data key: %w", err)
	}

	// Update all master keys with the data key (including the new one)
	errs := tree.Metadata.UpdateMasterKeysWithKeyServices(dataKey, []keyservice.KeyServiceClient{
		keyservice.NewLocalClient(),
	})
	if len(errs) > 0 {
		return fmt.Errorf("failed to update master keys (%d errors)", len(errs))
	}

	// Emit the updated encrypted file
	encryptedFile, err := store.EmitEncryptedFile(tree)
	if err != nil {
		return fmt.Errorf("failed to emit encrypted file: %w", err)
	}

	// Ensure directory exists with restrictive permissions
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write back to file with restricted permissions
	if err := os.WriteFile(filePath, encryptedFile, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// removeKeyFromFile removes an age key from a single encrypted file
func removeKeyFromFile(filePath, ageKey string) error {
	// Load the encrypted file
	store := sopsyaml.Store{}

	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	tree, err := store.LoadEncryptedFile(fileBytes)
	if err != nil {
		return fmt.Errorf("failed to load encrypted file: %w", err)
	}

	// Remove the key from all key groups
	found := false
	for i, group := range tree.Metadata.KeyGroups {
		newGroup := sops.KeyGroup{}
		for _, key := range group {
			if ageMasterKey, ok := key.(*age.MasterKey); ok {
				if ageMasterKey.Recipient != ageKey {
					newGroup = append(newGroup, key)
				} else {
					found = true
				}
			} else {
				newGroup = append(newGroup, key)
			}
		}
		tree.Metadata.KeyGroups[i] = newGroup
	}

	if !found {
		// Key wasn't in this file, skip
		return nil
	}

	// Ensure at least one key remains
	hasKeys := false
	for _, group := range tree.Metadata.KeyGroups {
		if len(group) > 0 {
			hasKeys = true
			break
		}
	}
	if !hasKeys {
		return fmt.Errorf("cannot remove the last key from file")
	}

	// Get existing data key and update master keys
	dataKey, err := tree.Metadata.GetDataKey()
	if err != nil {
		return fmt.Errorf("failed to get data key: %w", err)
	}

	// Update metadata to rotate the data key (more secure when removing keys)
	errs := tree.Metadata.UpdateMasterKeysWithKeyServices(dataKey, []keyservice.KeyServiceClient{
		keyservice.NewLocalClient(),
	})
	if len(errs) > 0 {
		return fmt.Errorf("failed to update master keys after removal (%d errors)", len(errs))
	}

	// Emit the updated encrypted file
	encryptedFile, err := store.EmitEncryptedFile(tree)
	if err != nil {
		return fmt.Errorf("failed to emit encrypted file: %w", err)
	}

	// Ensure directory exists with restrictive permissions
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write back to file with restricted permissions
	if err := os.WriteFile(filePath, encryptedFile, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ExtractAgeKeys extracts age keys from parsed SOPS YAML metadata
func ExtractAgeKeys(yamlData map[string]interface{}) []string {
	keys := []string{}

	if sopsData, ok := yamlData["sops"].(map[string]interface{}); ok {
		if ageArray, ok := sopsData["age"].([]interface{}); ok {
			for _, ageEntry := range ageArray {
				if ageMap, ok := ageEntry.(map[string]interface{}); ok {
					if recipient, ok := ageMap["recipient"].(string); ok && recipient != "" {
						keys = append(keys, recipient)
					}
				}
			}
		}
	}

	return keys
}

// extractAgeKeys is a deprecated alias for ExtractAgeKeys
// Deprecated: Use ExtractAgeKeys instead
func extractAgeKeys(yamlData map[string]interface{}) []string {
	return ExtractAgeKeys(yamlData)
}
