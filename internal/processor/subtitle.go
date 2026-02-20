package processor

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// convertToASS converts SRT subtitle to ASS format
// ASS format is required to avoid font rendering issues on macOS
func (p *implProcessor) convertToASS(ctx context.Context, srtPath string) (string, error) {
	assPath := strings.TrimSuffix(srtPath, filepath.Ext(srtPath)) + ".ass"

	p.logger.Info(ctx, "Converting SRT to ASS: %s", srtPath)

	args := []string{
		"-i", srtPath,
		"-y", // Overwrite output file if exists
		assPath,
	}

	if _, err := p.executor.Execute(ctx, "ffmpeg", args...); err != nil {
		return "", fmt.Errorf("ffmpeg convert to ASS: %w", err)
	}

	p.logger.Info(ctx, "Converted to ASS: %s", assPath)
	return assPath, nil
}

// burnSubtitle burns subtitle into video using hardware acceleration
// Optimized for M4 Pro with VideoToolbox encoder
func (p *implProcessor) burnSubtitle(ctx context.Context, videoPath, assPath string) (string, error) {
	// Generate output path
	filename := filepath.Base(videoPath)
	outputPath := filepath.Join(p.cfg.Paths.Output, "final_"+filename)

	p.logger.Info(ctx, "Burning subtitle into video (M4 Pro optimized): %s", videoPath)

	// FFmpeg arguments optimized for M4 Pro
	// -i: Input video
	// -vf ass=: Video filter to burn ASS subtitle
	// -c:v h264_videotoolbox: Use Apple Silicon hardware encoder
	// -b:v: Target video bitrate (8M for high quality)
	// -maxrate: Maximum bitrate (12M)
	// -bufsize: Buffer size for rate control (16M)
	// -profile:v: H.264 profile (high for better quality)
	// -level: H.264 level (4.2 supports up to 4K)
	// -c:a copy: Copy audio stream without re-encoding
	// -movflags +faststart: Optimize for streaming/web playback
	args := []string{
		"-i", videoPath,
		"-vf", fmt.Sprintf("ass=%s", assPath),
		"-c:v", p.cfg.FFmpeg.Encoder,
		"-b:v", p.cfg.FFmpeg.VideoBitrate,
	}

	// Add advanced encoding options if available
	if p.cfg.FFmpeg.MaxBitrate != "" {
		args = append(args, "-maxrate", p.cfg.FFmpeg.MaxBitrate)
	}
	if p.cfg.FFmpeg.BufSize != "" {
		args = append(args, "-bufsize", p.cfg.FFmpeg.BufSize)
	}

	// Add quality preset
	if p.cfg.FFmpeg.Preset != "" {
		// VideoToolbox doesn't use preset, but we can control quality
		// Use -q:v for quality (lower is better, 1-100)
		// For "medium" preset, use q:v 65 (good balance)
		switch p.cfg.FFmpeg.Preset {
		case "fast":
			args = append(args, "-q:v", "75")
		case "medium":
			args = append(args, "-q:v", "65")
		case "slow":
			args = append(args, "-q:v", "55")
		}
	}

	// H.264 profile and level for high quality
	args = append(args,
		"-profile:v", "high",
		"-level", "4.2",
		"-c:a", p.cfg.FFmpeg.AudioCodec,
		"-movflags", "+faststart", // Optimize for streaming
		"-y", // Overwrite output file if exists
		outputPath,
	)

	p.logger.Debug(ctx, "FFmpeg command: ffmpeg %v", strings.Join(args, " "))

	if _, err := p.executor.Execute(ctx, "ffmpeg", args...); err != nil {
		return "", fmt.Errorf("ffmpeg burn subtitle: %w", err)
	}

	p.logger.Info(ctx, "Subtitle burned successfully: %s", outputPath)
	return outputPath, nil
}
