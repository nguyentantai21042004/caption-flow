package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type implExecutor struct{}

// New creates a new Executor instance
func New() Executor {
	return &implExecutor{}
}

// Execute runs an external command with the given arguments
func (e *implExecutor) Execute(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Include stderr in error message for debugging
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return "", fmt.Errorf("command '%s' failed: %w\nstderr: %s", name, err, stderrStr)
		}
		return "", fmt.Errorf("command '%s' failed: %w", name, err)
	}

	return stdout.String(), nil
}

// ExecuteInDir runs an external command in a specific working directory
func (e *implExecutor) ExecuteInDir(ctx context.Context, dir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir // Set working directory

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Include stderr in error message for debugging
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return "", fmt.Errorf("command '%s' failed: %w\nstderr: %s", name, err, stderrStr)
		}
		return "", fmt.Errorf("command '%s' failed: %w", name, err)
	}

	return stdout.String(), nil
}
