package partybus

import (
	"io"
	"log/slog"
)

var logger *slog.Logger

func SetLogger(log *slog.Logger) {
	logger = log
}

func Logger() *slog.Logger {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
	return logger
}
