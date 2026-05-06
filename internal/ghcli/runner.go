package ghcli

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
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
			return stdout.String(), fmt.Errorf("%s failed: %s", name, stderrText)
		}
		return stdout.String(), fmt.Errorf("%s failed: %w", name, err)
	}
	return stdout.String(), nil
}
