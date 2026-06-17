package enricher

import (
	"github.com/nguyentantai21042004/caption-flow/internal/config"
	"github.com/nguyentantai21042004/caption-flow/internal/logger"
	"github.com/nguyentantai21042004/caption-flow/pkg/executor"
)

type implEnricher struct {
	cfg  config.EnrichConfig
	exec executor.Executor
	log  logger.Logger
}

// New creates an Enricher from config + shared executor/logger dependencies.
func New(cfg config.EnrichConfig, exec executor.Executor, log logger.Logger) Enricher {
	return &implEnricher{cfg: cfg, exec: exec, log: log}
}
