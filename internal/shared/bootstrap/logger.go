package bootstrap

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"internal/shared/settings"
)

// BuildLogger creates a structured *slog.Logger from the given LogConfig.
// The runtime name is added as a default field to every log line,
// enabling log aggregation and filtering across multiple binaries.
func BuildLogger(cfg settings.LogConfig, runtime string) *slog.Logger {
	return newLogger(cfg, os.Stdout, runtime)
}

func newLogger(cfg settings.LogConfig, writer io.Writer, runtime string) *slog.Logger {
	var level slog.Level
	if err := level.UnmarshalText([]byte(strings.ToLower(string(cfg.Level)))); err != nil {
		level = slog.LevelInfo
	}

	options := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if strings.EqualFold(string(cfg.Format), string(settings.LogFormatJSON)) {
		handler = slog.NewJSONHandler(writer, options)
	} else {
		handler = slog.NewTextHandler(writer, options)
	}

	if runtime != "" {
		return slog.New(handler).With("runtime", runtime)
	}
	return slog.New(handler)
}
