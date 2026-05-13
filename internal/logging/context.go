package logging

import (
	"context"
)

type key int

const loggerKey key = 0

// FromContext extracts a Logger from the context, creating a default one if absent.
func FromContext(ctx context.Context) (logger Logger, created bool) {
	logger, ok := ctx.Value(loggerKey).(Logger)

	if !ok {
		logger = NewDefaultLogger()
		created = true
	}

	return
}

// NewContext returns a context with the given Logger attached.
func NewContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}
