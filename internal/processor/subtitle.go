package processor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// burnSubtitle burns subtitle into video using hardware acceleration
// Uses relative path with working directory to avoid FFmpeg filter parsing issues
func (p *implProcessor) burnSubtitle(ctx context.Context, videoPath, srtPath string) (string, error) {
	filename := filepath.Base(videoPath)
	videosDir := filepath.Join(p.cfg.Paths.Output, "videos")
	if err := os.MkdirAll(videosDir, 0755); err != nil {
		return "", fmt.Errorf("create videos dir: %w", err)
	}
	outputPath := filepath.Join(videosDir, filename)

	p.logger.Info(ctx, "Burning subtitle into video (M4 Pro optimized): %s", videoPath)

	// Convert SRT to ASS first for better styling
	assPath := srtPath[:len(srtPath)-4] + ".ass"
	argsConvert := []string{
		"-i", srtPath,
		"-y",
		assPath,
	}

	if _, err := p.executor.Execute(ctx, "ffmpeg", argsConvert...); err != nil {
		p.logger.Warn(ctx, "Failed to convert SRT to ASS, using SRT: %v", err)
		assPath = srtPath // Use SRT if conversion fails
	}
	defer os.Remove(assPath)

	// Create isolated temp dir per video to avoid race conditions
	tempDir, err := os.MkdirTemp(p.cfg.Paths.Temp, "burn-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	tempSubtitle := filepath.Join(tempDir, "subtitle.ass")
	tempOutput := filepath.Join(tempDir, "output.mp4")

	// Copy subtitle to temp location
	if err := p.copyFile(assPath, tempSubtitle); err != nil {
		return "", fmt.Errorf("copy subtitle to temp: %w", err)
	}

	// Get absolute paths for input/output
	absVideoPath, _ := filepath.Abs(videoPath)
	absTempOutput, _ := filepath.Abs(tempOutput)

	workDir := tempDir
	subFilename := filepath.Base(tempSubtitle)

	// Clean filename (trim spaces)
	subFilename = strings.TrimSpace(subFilename)

	// Use subtitles filter with RELATIVE path (no quotes needed!)
	args := []string{
		"-y",
		"-i", absVideoPath,
		"-vf", fmt.Sprintf("subtitles=%s", subFilename), // No quotes!
		"-c:v", p.cfg.FFmpeg.Encoder,
		"-b:v", p.cfg.FFmpeg.VideoBitrate,
		"-c:a", p.cfg.FFmpeg.AudioCodec,
		absTempOutput,
	}

	p.logger.Debug(ctx, "FFmpeg command in dir %s: ffmpeg -vf subtitles=%s ...", workDir, subFilename)

	// Execute FFmpeg in the temp directory (this is the key!)
	if _, err := p.executor.ExecuteInDir(ctx, workDir, "ffmpeg", args...); err != nil {
		// If hardware encoder fails, try software encoder
		p.logger.Warn(ctx, "Hardware encoder failed, trying software encoder...")
		if err := p.burnSubtitleSoftware(ctx, workDir, absVideoPath, subFilename, absTempOutput); err != nil {
			return "", fmt.Errorf("both hardware and software encoders failed: %w", err)
		}
	}

	// Move temp output to final location
	if err := os.Rename(tempOutput, outputPath); err != nil {
		// If rename fails, copy instead
		if err := p.copyFile(tempOutput, outputPath); err != nil {
			return "", fmt.Errorf("move output to final location: %w", err)
		}
	}

	p.logger.Info(ctx, "Subtitle burned successfully: %s", outputPath)
	return outputPath, nil
}

// copyFile copies a file from src to dst
func (p *implProcessor) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("write destination: %w", err)
	}
	return nil
}

// burnSubtitleSoftware uses software encoder as fallback
func (p *implProcessor) burnSubtitleSoftware(ctx context.Context, workDir, videoPath, subFilename, outputPath string) error {
	args := []string{
		"-y",
		"-i", videoPath,
		"-vf", fmt.Sprintf("subtitles=%s", subFilename), // No quotes!
		"-c:v", "libx264",
		"-preset", p.cfg.FFmpeg.Preset,
		"-crf", "23",
		"-c:a", "copy",
		outputPath,
	}

	if _, err := p.executor.ExecuteInDir(ctx, workDir, "ffmpeg", args...); err != nil {
		return fmt.Errorf("software encoder failed: %w", err)
	}

	p.logger.Info(ctx, "Subtitle burned successfully with software encoder")
	return nil
}
