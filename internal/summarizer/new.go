package summarizer

import (
	"github.com/nguyentantai21042004/caption-flow/internal/logger"
)

type implSummarizer struct {
	apiKeys    []string
	currentKey int
	logger     logger.Logger
	model      string
	prompt     string
}

func New(apiKeys []string, model, prompt string, log logger.Logger) Summarizer {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return &implSummarizer{
		apiKeys: apiKeys,
		logger:  log,
		model:   model,
		prompt:  prompt,
	}
}
