package processor

import (
	"github.com/nguyentantai21042004/caption-flow/internal/config"
	"github.com/nguyentantai21042004/caption-flow/internal/logger"
	"github.com/nguyentantai21042004/caption-flow/pkg/executor"
)

type implProcessor struct {
	cfg      *config.Config
	executor executor.Executor
	logger   logger.Logger
}

// New creates a new Processor instance
func New(cfg *config.Config, exec executor.Executor, log logger.Logger) Processor {
	return &implProcessor{
		cfg:      cfg,
		executor: exec,
		logger:   log,
	}
}
