package watcher

import "context"

// Watcher defines the interface for file system monitoring
type Watcher interface {
	Start(ctx context.Context) error
	Stop() error
}

// EventHandler is a function that handles file events
type EventHandler func(ctx context.Context, filePath string) error
