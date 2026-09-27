// Package logging configures process-wide structured logging.
package logging

import (
	"io"
	"log/slog"
)

func Init(out io.Writer, debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
	return logger
}
