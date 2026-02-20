package watcher

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nguyentantai21042004/caption-flow/internal/logger"
)

type implWatcher struct {
	inputDir      string
	handler       EventHandler
	logger        logger.Logger
	watcher       *fsnotify.Watcher
	maxConcurrent int
	semaphore     chan struct{}
	wg            sync.WaitGroup
}

// Start begins monitoring the input directory for new video files
// Optimized for M4 Pro with concurrent processing support
func (w *implWatcher) Start(ctx context.Context) error {
	w.logger.Info(ctx, "File watcher started (max concurrent: %d). Monitoring: %s", w.maxConcurrent, w.inputDir)
	w.logger.Info(ctx, "Supported formats: .mp4, .mov, .avi, .mkv, .webm, .m4v, .flv")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info(ctx, "Waiting for ongoing processing to complete...")
			w.wg.Wait()
			w.logger.Info(ctx, "File watcher stopped")
			return ctx.Err()

		case event, ok := <-w.watcher.Events:
			if !ok {
				return fmt.Errorf("watcher events channel closed")
			}

			// Only process CREATE events
			if event.Op&fsnotify.Create == fsnotify.Create {
				if w.isVideoFile(event.Name) {
					w.logger.Info(ctx, "New video detected: %s", event.Name)

					// Small delay to ensure file is fully written
					time.Sleep(500 * time.Millisecond)

					// Acquire semaphore slot (blocks if max concurrent reached)
					select {
					case w.semaphore <- struct{}{}:
						w.wg.Add(1)
						// Handle the file in a goroutine with concurrency control
						go func(filePath string) {
							defer w.wg.Done()
							defer func() { <-w.semaphore }() // Release semaphore

							if err := w.handler(ctx, filePath); err != nil {
								w.logger.Error(ctx, "Failed to process %s: %v", filePath, err)
							}
						}(event.Name)
					case <-ctx.Done():
						return ctx.Err()
					}
				} else {
					w.logger.Debug(ctx, "Ignoring non-video file: %s", event.Name)
				}
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher errors channel closed")
			}
			w.logger.Error(ctx, "Watcher error: %v", err)
		}
	}
}

// Stop closes the file watcher
func (w *implWatcher) Stop() error {
	return w.watcher.Close()
}

// isVideoFile checks if the file has a supported video extension
func (w *implWatcher) isVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	supportedFormats := []string{".mp4", ".mov", ".avi", ".mkv", ".webm", ".m4v", ".flv"}

	for _, format := range supportedFormats {
		if ext == format {
			return true
		}
	}

	return false
}
