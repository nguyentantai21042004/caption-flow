package processor

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// extractAudio extracts audio from video file and converts to 16kHz mono WAV
// This format is optimal for Whisper processing
// Optimized for M4 Pro with faster processing
func (p *implProcessor) extractAudio(ctx context.Context, videoPath string) (string, error) {
	// Generate output audio path
	audioPath := strings.TrimSuffix(videoPath, filepath.Ext(videoPath)) + "_temp.wav"

	p.logger.Info(ctx, "Extracting audio (optimized for M4 Pro): %s", videoPath)

	// FFmpeg arguments for audio extraction
	// -i: Input video
	// -vn: No video (audio only)
	// -ar 16000: Sample rate 16kHz (optimal for Whisper)
	// -ac 1: Mono channel (Whisper works best with mono)
	// -c:a pcm_s16le: PCM 16-bit little-endian format (uncompressed, best quality)
	// -threads 0: Use all available CPU threads
	// -y: Overwrite output file if exists
	args := []string{
		"-i", videoPath,
		"-vn",          // No video
		"-ar", "16000", // 16kHz sample rate
		"-ac", "1", // Mono
		"-c:a", "pcm_s16le",
		"-threads", "0", // Use all available threads
		"-y",
		audioPath,
	}

	if _, err := p.executor.Execute(ctx, "ffmpeg", args...); err != nil {
		return "", fmt.Errorf("ffmpeg extract audio: %w", err)
	}

	p.logger.Info(ctx, "Audio extracted successfully: %s", audioPath)
	return audioPath, nil
}
