package ghcli

import (
	"testing"
)

func TestGetScopeHint(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "single scope error",
			input: `Your token has not been granted the required scopes to execute this query. The 'title' field requires one of the following scopes: ['read:project'], but your token has only been granted the: ['gist', 'read:org', 'repo', 'workflow'] scopes.`,
			expected: "Hint: Run 'gh auth refresh -h github.com -s read:project' to grant the required scope.",
		},
		{
			name: "multiple scopes error",
			input: `Your token has not been granted the required scopes to execute this query. The 'content' field requires one of the following scopes: ['read:project', 'write:project'], but your token has only been granted the: ['repo'] scopes.`,
			expected: "Hint: Run 'gh auth refresh -h github.com -s read:project' to grant the required scope.",
		},
		{
			name:     "no scope error",
			input:    "gh failed: some other error",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "partial match",
			input:    "The 'title' field requires scopes",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getScopeHint(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
