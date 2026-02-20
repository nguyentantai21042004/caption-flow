package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/nguyentantai21042004/caption-flow/internal/config"
	"github.com/nguyentantai21042004/caption-flow/internal/logger"
	"github.com/nguyentantai21042004/caption-flow/internal/processor"
	"github.com/nguyentantai21042004/caption-flow/internal/watcher"
	"github.com/nguyentantai21042004/caption-flow/pkg/executor"
)

func main() {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log := logger.New(cfg.Logging.Level)
	log.Info(ctx, "========================================")
	log.Info(ctx, "Video Processing Pipeline (M4 Pro Optimized)")
	log.Info(ctx, "========================================")
	log.Info(ctx, "System: %s/%s", runtime.GOOS, runtime.GOARCH)
	log.Info(ctx, "CPU Cores: %d", runtime.NumCPU())
	log.Info(ctx, "Max Concurrent Processing: %d", cfg.Performance.MaxConcurrent)
	log.Info(ctx, "Configuration loaded successfully")

	// Verify required directories exist
	if err := ensureDirectories(cfg); err != nil {
		log.Error(ctx, "Failed to create directories: %v", err)
		os.Exit(1)
	}

	// Initialize dependencies
	exec := executor.New()
	proc := processor.New(cfg, exec, log)

	// Create watcher with processor as handler and concurrency control
	w, err := watcher.New(cfg.Paths.Input, proc.Process, log, cfg.Performance.MaxConcurrent)
	if err != nil {
		log.Error(ctx, "Failed to create watcher: %v", err)
		os.Exit(1)
	}
	defer w.Stop()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start watcher in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := w.Start(ctx); err != nil && err != context.Canceled {
			errChan <- err
		}
	}()

	log.Info(ctx, "========================================")
	log.Info(ctx, "Video Pipeline is ready!")
	log.Info(ctx, "Monitoring: %s", cfg.Paths.Input)
	log.Info(ctx, "Output: %s", cfg.Paths.Output)
	log.Info(ctx, "")
	log.Info(ctx, "Optimizations:")
	log.Info(ctx, "  - Whisper: %d threads, Metal GPU", cfg.Whisper.Threads)
	log.Info(ctx, "  - FFmpeg: %s encoder, %s bitrate", cfg.FFmpeg.Encoder, cfg.FFmpeg.VideoBitrate)
	log.Info(ctx, "  - Concurrent: %d videos at once", cfg.Performance.MaxConcurrent)
	log.Info(ctx, "")
	log.Info(ctx, "Press Ctrl+C to stop")
	log.Info(ctx, "========================================")

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		log.Info(ctx, "Shutdown signal received")
	case err := <-errChan:
		log.Error(ctx, "Watcher error: %v", err)
	}

	// Graceful shutdown
	log.Info(ctx, "Shutting down gracefully...")
	cancel()

	log.Info(ctx, "Video Pipeline stopped")
}

// ensureDirectories creates required directories if they don't exist
func ensureDirectories(cfg *config.Config) error {
	dirs := []string{
		cfg.Paths.Input,
		cfg.Paths.Processing,
		cfg.Paths.Output,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	return nil
}
