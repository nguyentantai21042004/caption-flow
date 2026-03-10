package summarizer

import (
	"net/http"
	"time"

	"github.com/nguyentantai21042004/caption-flow/internal/logger"
)

type implSummarizer struct {
	deepSeekKeys       []string
	currentDeepSeekKey int
	geminiKeys         []string
	currentGeminiKey   int
	logger             logger.Logger
	deepSeekModel      string
	deepSeekBaseURL    string
	geminiModel        string
	prompt             string
	httpClient         *http.Client
}

func New(deepSeekKeys, geminiKeys []string, deepSeekModel, deepSeekBaseURL, geminiModel, prompt string, log logger.Logger) Summarizer {
	if deepSeekModel == "" {
		deepSeekModel = "deepseek-chat"
	}
	if deepSeekBaseURL == "" {
		deepSeekBaseURL = "https://api.deepseek.com/v1"
	}
	if geminiModel == "" {
		geminiModel = "gemini-2.5-flash"
	}
	return &implSummarizer{
		deepSeekKeys:    deepSeekKeys,
		geminiKeys:      geminiKeys,
		logger:          log,
		deepSeekModel:   deepSeekModel,
		deepSeekBaseURL: deepSeekBaseURL,
		geminiModel:     geminiModel,
		prompt:          prompt,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}
