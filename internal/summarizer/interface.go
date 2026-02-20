package summarizer

import "context"

// Summarizer reads SRT files and produces LLM-generated markdown summaries.
type Summarizer interface {
	SummarizeAll(ctx context.Context, srtDir, destDir string) error
}
