package templating

import (
	"testing"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name      string
		values    map[string]interface{}
		expected  map[string]interface{}
		expectErr bool
	}{
		{
			name: "simple substitution",
			values: map[string]interface{}{
				"BASE":  "value",
				"FULL":  "${BASE}/path",
			},
			expected: map[string]interface{}{
				"BASE": "value",
				"FULL": "value/path",
			},
			expectErr: false,
		},
		{
			name: "multiple substitutions",
			values: map[string]interface{}{
				"A": "hello",
				"B": "world",
				"C": "${A} ${B}",
			},
			expected: map[string]interface{}{
				"A": "hello",
				"B": "world",
				"C": "hello world",
			},
			expectErr: false,
		},
		{
			name: "nested substitution",
			values: map[string]interface{}{
				"A": "base",
				"B": "${A}/middle",
				"C": "${B}/end",
			},
			expected: map[string]interface{}{
				"A": "base",
				"B": "base/middle",
				"C": "base/middle/end",
			},
			expectErr: false,
		},
		{
			name: "undefined variable",
			values: map[string]interface{}{
				"A": "${UNDEFINED}",
			},
			expectErr: true,
		},
		{
			name: "circular dependency",
			values: map[string]interface{}{
				"A": "${B}",
				"B": "${A}",
			},
			expectErr: true,
		},
		{
			name: "no substitution needed",
			values: map[string]interface{}{
				"A": "plain",
				"B": "text",
			},
			expected: map[string]interface{}{
				"A": "plain",
				"B": "text",
			},
			expectErr: false,
		},
		{
			name: "underscore-prefixed variables",
			values: map[string]interface{}{
				"_INTERNAL": "secret",
				"PUBLIC":    "uses ${_INTERNAL}",
			},
			expected: map[string]interface{}{
				"_INTERNAL": "secret",
				"PUBLIC":    "uses secret",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewResolver(tt.values)
			resolved, err := resolver.Resolve()

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			for key, expectedValue := range tt.expected {
				if actualValue, ok := resolved[key]; !ok {
					t.Errorf("Key %s not found in resolved values", key)
				} else if actualValue != expectedValue {
					t.Errorf("Key %s: expected %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestResolveString(t *testing.T) {
	resolver := NewResolver(map[string]interface{}{
		"NAME": "Alice",
		"AGE":  "30",
	})

	tests := []struct {
		input    string
		expected string
	}{
		{"Hello ${NAME}", "Hello Alice"},
		{"${NAME} is ${AGE}", "Alice is 30"},
		{"No substitution", "No substitution"},
		{"${NAME}${NAME}", "AliceAlice"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := resolver.ResolveString(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
