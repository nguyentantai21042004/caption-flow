package processor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// moveToProcessing moves video file from input to processing folder
func (p *implProcessor) moveToProcessing(ctx context.Context, videoPath string) (string, error) {
	filename := filepath.Base(videoPath)
	destPath := filepath.Join(p.cfg.Paths.Processing, filename)

	p.logger.Info(ctx, "Moving to processing folder: %s -> %s", videoPath, destPath)

	if err := os.Rename(videoPath, destPath); err != nil {
		return "", fmt.Errorf("move to processing: %w", err)
	}

	return destPath, nil
}

// moveToOutput moves subtitle file to output folder
func (p *implProcessor) moveToOutput(ctx context.Context, srtPath string) error {
	srtFilename := filepath.Base(srtPath)
	destSRT := filepath.Join(p.cfg.Paths.Output, srtFilename)

	p.logger.Info(ctx, "Moving SRT to output: %s -> %s", srtPath, destSRT)

	if err := os.Rename(srtPath, destSRT); err != nil {
		return fmt.Errorf("move SRT to output: %w", err)
	}

	return nil
}

// cleanup removes temporary files and original video from processing folder
func (p *implProcessor) cleanup(ctx context.Context, videoPath string) error {
	p.logger.Info(ctx, "Cleaning up: %s", videoPath)

	if err := os.Remove(videoPath); err != nil {
		return fmt.Errorf("remove video from processing: %w", err)
	}

	return nil
}

// cleanupTempFile removes a temporary file, logs warning if fails
func (p *implProcessor) cleanupTempFile(ctx context.Context, filePath string) {
	if err := os.Remove(filePath); err != nil {
		p.logger.Warn(ctx, "Failed to cleanup temp file %s: %v", filePath, err)
	} else {
		p.logger.Debug(ctx, "Cleaned up temp file: %s", filePath)
	}
}
