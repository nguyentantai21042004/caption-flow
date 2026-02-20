package summarizer

import "context"

// Summarizer reads SRT files and produces transcript + summary DOCX files.
type Summarizer interface {
	// SummarizeAll discovers SRTs in outputDir, generates:
	//   outputDir/transcripts/*.docx  (raw SRT content)
	//   outputDir/summaries/*.docx    (LLM-generated summary)
	//   outputDir/archived/*.srt      (processed SRTs moved here)
	SummarizeAll(ctx context.Context, outputDir string) error
}
