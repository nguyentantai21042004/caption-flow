package processor

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// transcribe uses Whisper to convert audio to subtitle file (SRT format)
// Optimized for M4 Pro with Metal acceleration and multi-threading
func (p *implProcessor) transcribe(ctx context.Context, audioPath string) (string, error) {
	// Generate output prefix (Whisper will append .srt)
	outputPrefix := strings.TrimSuffix(audioPath, filepath.Ext(audioPath))

	p.logger.Info(ctx, "Starting transcription with %d threads (Metal GPU enabled): %s",
		p.cfg.Whisper.Threads, audioPath)

	// Whisper arguments optimized for M4 Pro
	// -m: Model path
	// -f: Input audio file
	// -osrt: Output SRT format
	// -l: Force language (prevents hallucination)
	// --prompt: Domain-specific keywords to improve accuracy
	// --output-file: Output file prefix
	// -t: Number of threads (8 for M4 Pro)
	// -ml: Max segment length (0 = no limit, better for long videos)
	// -mc: Max context (0 = no limit)
	// -bo: Best of (5 = better accuracy)
	args := []string{
		"-m", p.cfg.Whisper.ModelPath,
		"-f", audioPath,
		"-osrt",
		"-l", p.cfg.Whisper.Language,
		"-t", strconv.Itoa(p.cfg.Whisper.Threads),
		"-ml", "0", // No max length limit
		"-mc", "0", // No max context limit
		"-bo", "5", // Best of 5 for better accuracy
		"--prompt", p.cfg.Whisper.Prompt,
		"--output-file", outputPrefix,
	}

	// Add GPU flag if enabled (Metal acceleration on Apple Silicon)
	if p.cfg.Whisper.UseGPU {
		// Metal is enabled by default in whisper.cpp on macOS
		p.logger.Debug(ctx, "Metal GPU acceleration enabled")
	}

	if _, err := p.executor.Execute(ctx, p.cfg.Whisper.BinaryPath, args...); err != nil {
		return "", fmt.Errorf("whisper transcribe: %w", err)
	}

	srtPath := outputPrefix + ".srt"
	p.logger.Info(ctx, "Transcription completed: %s", srtPath)
	return srtPath, nil
}
