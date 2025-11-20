package templating

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// templateVarRegex matches ${VAR_NAME} patterns
	templateVarRegex = regexp.MustCompile(`\$\{([^}]+)\}`)
)

// Resolver handles template variable resolution
type Resolver struct {
	values map[string]interface{}
}

// NewResolver creates a new template resolver with the given values
func NewResolver(values map[string]interface{}) *Resolver {
	return &Resolver{
		values: values,
	}
}

// Resolve resolves all template variables in the given values map
// Returns a new map with resolved values
func (r *Resolver) Resolve() (map[string]interface{}, error) {
	resolved := make(map[string]interface{})
	resolving := make(map[string]bool) // Track variables currently being resolved to detect cycles

	// Resolve each value
	for key, value := range r.values {
		resolvedValue, err := r.resolveValue(key, value, resolving)
		if err != nil {
			return nil, err
		}
		resolved[key] = resolvedValue
	}

	return resolved, nil
}

// resolveValue resolves template variables in a single value
func (r *Resolver) resolveValue(key string, value interface{}, resolving map[string]bool) (interface{}, error) {
	// Check for circular dependency
	if resolving[key] {
		return nil, fmt.Errorf("circular dependency detected for variable: %s", key)
	}

	// Only resolve strings
	strValue, ok := value.(string)
	if !ok {
		return value, nil
	}

	// Mark this variable as being resolved
	resolving[key] = true
	defer delete(resolving, key)

	// Find all template variables in the string
	matches := templateVarRegex.FindAllStringSubmatch(strValue, -1)
	if len(matches) == 0 {
		return strValue, nil
	}

	result := strValue
	for _, match := range matches {
		fullMatch := match[0]  // ${VAR_NAME}
		varName := match[1]    // VAR_NAME

		// Look up the variable value
		varValue, exists := r.values[varName]
		if !exists {
			return nil, fmt.Errorf("undefined variable referenced: %s (in %s)", varName, key)
		}

		// Recursively resolve the referenced variable
		resolvedVarValue, err := r.resolveValue(varName, varValue, resolving)
		if err != nil {
			return nil, err
		}

		// Convert to string for substitution
		varStr := fmt.Sprintf("%v", resolvedVarValue)

		// Replace the template variable with its value
		result = strings.ReplaceAll(result, fullMatch, varStr)
	}

	return result, nil
}

// ResolveString resolves template variables in a single string value
func (r *Resolver) ResolveString(value string) (string, error) {
	resolved, err := r.resolveValue("", value, make(map[string]bool))
	if err != nil {
		return "", err
	}
	return resolved.(string), nil
}
