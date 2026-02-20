package summarizer

import (
	"github.com/nguyentantai21042004/caption-flow/internal/logger"
)

type implSummarizer struct {
	apiKeys    []string
	currentKey int
	logger     logger.Logger
	model      string
}

// New creates a Summarizer that rotates through the supplied Gemini API keys.
func New(apiKeys []string, log logger.Logger) Summarizer {
	return &implSummarizer{
		apiKeys: apiKeys,
		logger:  log,
		model:   "gemini-2.5-flash",
	}
}
