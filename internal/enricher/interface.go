package enricher

import "context"

// Enricher turns a YouTube URL into a "Claude-ready" bundle on disk: the
// caption transcript (timestamped segments) plus scene-change keyframes mapped
// to the transcript timeline, so a multimodal model can read image + speech
// together and write course documents.
type Enricher interface {
	Enrich(ctx context.Context, videoURL string) (*Bundle, error)
}
