package ghcli

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type Runner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		stderrText := strings.TrimSpace(stderr.String())
		if stderrText != "" {
			errMsg := fmt.Sprintf("%s failed: %s", name, stderrText)
			if hint := getScopeHint(stderrText); hint != "" {
				errMsg += "\n" + hint
			}
			return stdout.String(), fmt.Errorf("%s", errMsg)
		}
		return stdout.String(), fmt.Errorf("%s failed: %w", name, err)
	}
	return stdout.String(), nil
}

// getScopeHint checks if the error message indicates a missing scope and returns a helpful hint.
func getScopeHint(stderrText string) string {
	// Pattern: "The 'X' field requires one of the following scopes: ['Y'], but your token has only been granted the: ['Z'] scopes."
	scopePattern := regexp.MustCompile(`The '(\w+)' field requires one of the following scopes: \[([^\]]+)\], but your token has only been granted the: \[([^\]]+)\] scopes`)
	if matches := scopePattern.FindStringSubmatch(stderrText); len(matches) > 0 {
		missingScopes := matches[2]
		// Extract individual scopes from the quoted list
		scopes := strings.Split(missingScopes, ", ")
		if len(scopes) > 0 {
			// Clean up scope names (remove quotes)
			firstScope := strings.Trim(scopes[0], "'")
			return fmt.Sprintf("Hint: Run 'gh auth refresh -h github.com -s %s' to grant the required scope.", firstScope)
		}
	}
	return ""
}
