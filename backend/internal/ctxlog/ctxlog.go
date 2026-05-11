// Package ctxlog resolves a request-scoped zerolog logger from context (middleware.RequestID)
// and falls back to the global logger when no request context is attached.
package ctxlog

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// From returns the logger stored in ctx via zerolog.WithContext, or the global logger.
func From(ctx context.Context) *zerolog.Logger {
	if ctx == nil {
		return &log.Logger
	}
	zl := zerolog.Ctx(ctx)
	if zl.GetLevel() == zerolog.Disabled {
		return &log.Logger
	}
	return zl
}
