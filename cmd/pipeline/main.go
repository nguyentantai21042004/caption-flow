package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/nguyentantai21042004/caption-flow/internal/config"
	"github.com/nguyentantai21042004/caption-flow/internal/logger"
	"github.com/nguyentantai21042004/caption-flow/internal/processor"
	"github.com/nguyentantai21042004/caption-flow/internal/watcher"
	"github.com/nguyentantai21042004/caption-flow/pkg/executor"
)

func main() {
	// Parse command line flags
	target := flag.String("target", "", "Target video file(s) to process (comma-separated or single file)")
	watchMode := flag.Bool("watch", false, "Run in watch mode (monitor input folder)")
	flag.Parse()

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

	// Verify required directories exist
	if err := ensureDirectories(cfg); err != nil {
		log.Error(ctx, "Failed to create directories: %v", err)
		os.Exit(1)
	}

	// Initialize dependencies
	exec := executor.New()
	proc := processor.New(cfg, exec, log)

	// Determine mode: target or watch
	if *target != "" {
		// Target mode: process specific files
		runTargetMode(ctx, cfg, proc, log, *target)
	} else if *watchMode {
		// Watch mode: monitor input folder
		runWatchMode(ctx, cfg, proc, log)
	} else {
		// Default: list available files and show usage
		showUsage(ctx, cfg, log)
	}
}

// runTargetMode processes specific target files
func runTargetMode(ctx context.Context, cfg *config.Config, proc processor.Processor, log logger.Logger, target string) {
	log.Info(ctx, "Running in TARGET mode")
	log.Info(ctx, "Target: %s", target)
	log.Info(ctx, "========================================")

	// Parse targets (comma-separated)
	targets := strings.Split(target, ",")

	successCount := 0
	failCount := 0

	for _, t := range targets {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}

		// Check if file exists in input folder
		videoPath := filepath.Join(cfg.Paths.Input, t)
		if _, err := os.Stat(videoPath); os.IsNotExist(err) {
			log.Error(ctx, "File not found: %s", videoPath)
			failCount++
			continue
		}

		log.Info(ctx, "Processing: %s", t)
		if err := proc.Process(ctx, videoPath); err != nil {
			log.Error(ctx, "Failed to process %s: %v", t, err)
			failCount++
		} else {
			successCount++
		}
	}

	log.Info(ctx, "========================================")
	log.Info(ctx, "Processing completed!")
	log.Info(ctx, "Success: %d, Failed: %d", successCount, failCount)
	log.Info(ctx, "========================================")
}

// runWatchMode monitors input folder for new files
func runWatchMode(ctx context.Context, cfg *config.Config, proc processor.Processor, log logger.Logger) {
	log.Info(ctx, "Running in WATCH mode")
	log.Info(ctx, "Max Concurrent Processing: %d", cfg.Performance.MaxConcurrent)
	log.Info(ctx, "========================================")

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

// showUsage displays available files and usage instructions
func showUsage(ctx context.Context, cfg *config.Config, log logger.Logger) {
	log.Info(ctx, "Usage:")
	log.Info(ctx, "  ./vid-pipeline -target <filename>     # Process specific file(s)")
	log.Info(ctx, "  ./vid-pipeline -watch                 # Watch mode (monitor folder)")
	log.Info(ctx, "")
	log.Info(ctx, "Available files in %s:", cfg.Paths.Input)

	files, err := os.ReadDir(cfg.Paths.Input)
	if err != nil {
		log.Error(ctx, "Failed to read input directory: %v", err)
		return
	}

	videoCount := 0
	for _, file := range files {
		if file.IsDir() || strings.HasPrefix(file.Name(), ".") {
			continue
		}

		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext == ".mp4" || ext == ".mov" || ext == ".avi" || ext == ".mkv" || ext == ".webm" {
			info, _ := file.Info()
			log.Info(ctx, "  - %s (%.2f MB)", file.Name(), float64(info.Size())/1024/1024)
			videoCount++
		}
	}

	if videoCount == 0 {
		log.Info(ctx, "  (no video files found)")
	}

	log.Info(ctx, "")
	log.Info(ctx, "Examples:")
	log.Info(ctx, "  ./vid-pipeline -target \"video.mp4\"")
	log.Info(ctx, "  ./vid-pipeline -target \"video1.mp4,video2.mp4\"")
	log.Info(ctx, "  ./vid-pipeline -watch")
}

// ensureDirectories creates required directories if they don't exist
func ensureDirectories(cfg *config.Config) error {
	dirs := []string{
		cfg.Paths.Input,
		cfg.Paths.Output,
		cfg.Paths.Archived,
		cfg.Paths.Temp,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	return nil
}
