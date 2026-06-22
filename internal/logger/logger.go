package logger

import (
	"log/slog"
	"os"
)

// Setup initializes the global slog logger.
// If verbose is true, uses Debug level; otherwise Info.
func Setup(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	slog.SetDefault(slog.New(handler))
}
