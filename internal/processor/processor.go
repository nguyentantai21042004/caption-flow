package processor

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
)

// Process orchestrates the entire video processing pipeline
func (p *implProcessor) Process(ctx context.Context, videoPath string) error {
	startTime := time.Now()
	originalFilename := filepath.Base(videoPath)

	p.logger.Info(ctx, "========================================")
	p.logger.Info(ctx, "Starting video processing: %s", videoPath)
	p.logger.Info(ctx, "========================================")

	// Step 1: Extract audio
	audioPath, err := p.extractAudio(ctx, videoPath)
	if err != nil {
		return fmt.Errorf("extract audio: %w", err)
	}
	defer p.cleanupTempFile(ctx, audioPath)

	// Step 2: Transcribe audio to subtitle
	srtPath, err := p.transcribe(ctx, audioPath)
	if err != nil {
		return fmt.Errorf("transcribe: %w", err)
	}
	defer p.cleanupTempFile(ctx, srtPath)

	// Step 3: Burn subtitle into video (keeps original filename)
	outputPath, err := p.burnSubtitle(ctx, videoPath, srtPath)
	if err != nil {
		return fmt.Errorf("burn subtitle: %w", err)
	}

	// Step 4: Copy SRT to output folder (with original name)
	srtOutputPath := filepath.Join(p.cfg.Paths.Output, originalFilename[:len(originalFilename)-len(filepath.Ext(originalFilename))]+".srt")
	if err := p.copySRT(ctx, srtPath, srtOutputPath); err != nil {
		p.logger.Warn(ctx, "Failed to copy SRT to output: %v", err)
	}

	// Step 5: Move original video to archived folder
	if err := p.moveToArchived(ctx, videoPath); err != nil {
		p.logger.Warn(ctx, "Failed to move original to archived folder: %v", err)
	}

	duration := time.Since(startTime)
	p.logger.Info(ctx, "========================================")
	p.logger.Info(ctx, "Processing completed successfully!")
	p.logger.Info(ctx, "Output video: %s", outputPath)
	p.logger.Info(ctx, "Output subtitle: %s", srtOutputPath)
	p.logger.Info(ctx, "Processing time: %s", duration)
	p.logger.Info(ctx, "========================================")

	return nil
}
