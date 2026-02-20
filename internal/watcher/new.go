package watcher

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/nguyentantai21042004/caption-flow/internal/logger"
)

// New creates a new Watcher instance with concurrency control
func New(inputDir string, handler EventHandler, log logger.Logger, maxConcurrent int) (Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}

	if err := watcher.Add(inputDir); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("add watch path: %w", err)
	}

	// Default to 2 concurrent if not specified
	if maxConcurrent <= 0 {
		maxConcurrent = 2
	}

	return &implWatcher{
		inputDir:      inputDir,
		handler:       handler,
		logger:        log,
		watcher:       watcher,
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
	}, nil
}
