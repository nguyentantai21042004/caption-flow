package executor

import "context"

// Executor defines the interface for executing external commands
type Executor interface {
	Execute(ctx context.Context, name string, args ...string) (string, error)
	ExecuteInDir(ctx context.Context, dir string, name string, args ...string) (string, error)
}
