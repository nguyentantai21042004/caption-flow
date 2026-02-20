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
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/nguyentantai21042004/caption-flow/internal/config"
	"github.com/nguyentantai21042004/caption-flow/internal/logger"
	"github.com/nguyentantai21042004/caption-flow/internal/processor"
	"github.com/nguyentantai21042004/caption-flow/internal/summarizer"
	"github.com/nguyentantai21042004/caption-flow/internal/watcher"
	"github.com/nguyentantai21042004/caption-flow/pkg/executor"
)

func main() {
	// Parse command line flags
	target := flag.String("target", "", "Target video file(s) to process (comma-separated or single file)")
	targetAll := flag.Bool("target-all", false, "Process all video files in input folder")
	watchMode := flag.Bool("watch", false, "Run in watch mode (monitor input folder)")
	summarizeMode := flag.Bool("summarize", false, "Summarize all SRT files in output folder via Gemini")
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

	// Determine mode
	if *summarizeMode {
		runSummarize(ctx, cfg, log)
		return
	}

	if *targetAll {
		targets := discoverVideoFiles(ctx, cfg, log)
		if len(targets) == 0 {
			log.Info(ctx, "No video files found in %s", cfg.Paths.Input)
			return
		}
		runTargetMode(ctx, cfg, proc, log, strings.Join(targets, ","))
	} else if *target != "" {
		runTargetMode(ctx, cfg, proc, log, *target)
	} else if *watchMode {
		runWatchMode(ctx, cfg, proc, log)
	} else {
		showUsage(ctx, cfg, log)
	}
}

// runTargetMode processes target files concurrently using goroutines
func runTargetMode(ctx context.Context, cfg *config.Config, proc processor.Processor, log logger.Logger, target string) {
	startTime := time.Now()

	// Parse and validate targets
	var validPaths []string
	for _, t := range strings.Split(target, ",") {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		videoPath := filepath.Join(cfg.Paths.Input, t)
		if _, err := os.Stat(videoPath); os.IsNotExist(err) {
			log.Error(ctx, "File not found, skipping: %s", videoPath)
			continue
		}
		validPaths = append(validPaths, videoPath)
	}

	if len(validPaths) == 0 {
		log.Error(ctx, "No valid files to process")
		return
	}

	maxConcurrent := cfg.Performance.MaxConcurrent
	log.Info(ctx, "Running in TARGET mode (concurrent: %d goroutines)", maxConcurrent)
	log.Info(ctx, "Files to process: %d", len(validPaths))
	for i, p := range validPaths {
		log.Info(ctx, "  [%d] %s", i+1, filepath.Base(p))
	}
	log.Info(ctx, "========================================")

	// Concurrent processing with semaphore
	semaphore := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var successCount, failCount int64

	for _, videoPath := range validPaths {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire slot (blocks if full)

		go func(path string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release slot

			name := filepath.Base(path)
			log.Info(ctx, "[START] %s", name)
			if err := proc.Process(ctx, path); err != nil {
				log.Error(ctx, "[FAIL]  %s: %v", name, err)
				atomic.AddInt64(&failCount, 1)
			} else {
				log.Info(ctx, "[DONE]  %s", name)
				atomic.AddInt64(&successCount, 1)
			}
		}(videoPath)
	}

	wg.Wait()

	log.Info(ctx, "========================================")
	log.Info(ctx, "All processing completed!")
	log.Info(ctx, "Success: %d, Failed: %d, Total time: %s", successCount, failCount, time.Since(startTime).Round(time.Millisecond))
	log.Info(ctx, "========================================")
}

// runSummarize reads SRT files from output and generates a markdown summary via Gemini
func runSummarize(ctx context.Context, cfg *config.Config, log logger.Logger) {
	keysEnv := os.Getenv("GEMINI_API_KEYS")
	if keysEnv == "" {
		log.Error(ctx, "GEMINI_API_KEYS environment variable is not set")
		log.Error(ctx, "Usage: export GEMINI_API_KEYS=\"key1,key2,key3\"")
		os.Exit(1)
	}

	var keys []string
	for _, k := range strings.Split(keysEnv, ",") {
		k = strings.TrimSpace(k)
		if k != "" {
			keys = append(keys, k)
		}
	}

	if len(keys) == 0 {
		log.Error(ctx, "No valid API keys found in GEMINI_API_KEYS")
		os.Exit(1)
	}

	log.Info(ctx, "Running in SUMMARIZE mode")
	log.Info(ctx, "API keys loaded: %d", len(keys))
	log.Info(ctx, "Source: %s/*.srt", cfg.Paths.Output)
	log.Info(ctx, "========================================")

	destDir := filepath.Join(cfg.Paths.Output, "summaries")
	sum := summarizer.New(keys, log)

	startTime := time.Now()
	if err := sum.SummarizeAll(ctx, cfg.Paths.Output, destDir); err != nil {
		log.Error(ctx, "Summarization failed: %v", err)
		os.Exit(1)
	}

	log.Info(ctx, "========================================")
	log.Info(ctx, "Summarization completed in %s", time.Since(startTime).Round(time.Millisecond))
	log.Info(ctx, "Output: %s/", destDir)
	log.Info(ctx, "========================================")
}

// discoverVideoFiles scans the input directory for all video files
func discoverVideoFiles(ctx context.Context, cfg *config.Config, log logger.Logger) []string {
	supportedExts := map[string]bool{
		".mp4": true, ".mov": true, ".avi": true,
		".mkv": true, ".webm": true, ".m4v": true, ".flv": true,
	}

	files, err := os.ReadDir(cfg.Paths.Input)
	if err != nil {
		log.Error(ctx, "Failed to read input directory: %v", err)
		return nil
	}

	var videos []string
	for _, f := range files {
		if f.IsDir() || strings.HasPrefix(f.Name(), ".") {
			continue
		}
		if supportedExts[strings.ToLower(filepath.Ext(f.Name()))] {
			videos = append(videos, f.Name())
		}
	}
	return videos
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
	log.Info(ctx, "  ./vid-pipeline -target-all            # Process ALL video files in input")
	log.Info(ctx, "  ./vid-pipeline -target <filename>     # Process specific file(s)")
	log.Info(ctx, "  ./vid-pipeline -watch                 # Watch mode (monitor folder)")
	log.Info(ctx, "  ./vid-pipeline -summarize             # Summarize SRTs via Gemini")
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
