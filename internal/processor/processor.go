package processor

import (
	"context"
	"fmt"
	"time"
)

// Process orchestrates the entire video processing pipeline
func (p *implProcessor) Process(ctx context.Context, videoPath string) error {
	startTime := time.Now()
	p.logger.Info(ctx, "========================================")
	p.logger.Info(ctx, "Starting video processing: %s", videoPath)
	p.logger.Info(ctx, "========================================")

	// Step 1: Move to processing folder
	processingPath, err := p.moveToProcessing(ctx, videoPath)
	if err != nil {
		return fmt.Errorf("move to processing: %w", err)
	}

	// Step 2: Extract audio
	audioPath, err := p.extractAudio(ctx, processingPath)
	if err != nil {
		return fmt.Errorf("extract audio: %w", err)
	}
	defer p.cleanupTempFile(ctx, audioPath)

	// Step 3: Transcribe audio to subtitle
	srtPath, err := p.transcribe(ctx, audioPath)
	if err != nil {
		return fmt.Errorf("transcribe: %w", err)
	}

	// Step 4: Convert SRT to ASS
	assPath, err := p.convertToASS(ctx, srtPath)
	if err != nil {
		return fmt.Errorf("convert to ASS: %w", err)
	}
	defer p.cleanupTempFile(ctx, assPath)

	// Step 5: Burn subtitle into video
	outputPath, err := p.burnSubtitle(ctx, processingPath, assPath)
	if err != nil {
		return fmt.Errorf("burn subtitle: %w", err)
	}

	// Step 6: Move SRT to output folder
	if err := p.moveToOutput(ctx, srtPath); err != nil {
		p.logger.Warn(ctx, "Failed to move SRT to output: %v", err)
		// Not a critical error, continue
	}

	// Step 7: Cleanup original video from processing
	if err := p.cleanup(ctx, processingPath); err != nil {
		p.logger.Warn(ctx, "Cleanup failed: %v", err)
		// Not a critical error, continue
	}

	duration := time.Since(startTime)
	p.logger.Info(ctx, "========================================")
	p.logger.Info(ctx, "Processing completed successfully!")
	p.logger.Info(ctx, "Output video: %s", outputPath)
	p.logger.Info(ctx, "Processing time: %s", duration)
	p.logger.Info(ctx, "========================================")

	return nil
}
