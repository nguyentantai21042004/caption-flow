package processor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// moveToArchived moves original video to archived folder after successful processing
func (p *implProcessor) moveToArchived(ctx context.Context, videoPath string) error {
	// Ensure archived folder exists
	if err := os.MkdirAll(p.cfg.Paths.Archived, 0755); err != nil {
		return fmt.Errorf("create archived folder: %w", err)
	}

	filename := filepath.Base(videoPath)
	destPath := filepath.Join(p.cfg.Paths.Archived, filename)
	p.logger.Info(ctx, "Moving original video to archived: %s -> %s", videoPath, destPath)

	if err := os.Rename(videoPath, destPath); err != nil {
		return fmt.Errorf("move to archived: %w", err)
	}

	return nil
}

// copySRT copies subtitle file to output folder
func (p *implProcessor) copySRT(ctx context.Context, srtPath, destPath string) error {
	p.logger.Info(ctx, "Copying SRT to output: %s -> %s", srtPath, destPath)

	data, err := os.ReadFile(srtPath)
	if err != nil {
		return fmt.Errorf("read SRT: %w", err)
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("write SRT to output: %w", err)
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
