package processor

import "context"

// Processor defines the interface for video processing operations
type Processor interface {
	Process(ctx context.Context, videoPath string) error
}
