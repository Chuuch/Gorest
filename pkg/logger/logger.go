package logger

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

func Setup(level, format string, addSource bool) error {
	logLevel, err := parseLevel(level)
	if err != nil {
		return err
	}

	options := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: addSource,
	}

	handler, err := newHandler(format, options)
	if err != nil {
		return err
	}

	slog.SetDefault(slog.New(handler))
	return nil
}

func parseLevel(level string) (slog.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unknown log level %q", level)
	}
}

func newHandler(format string, options *slog.HandlerOptions) (slog.Handler, error) {
	switch strings.ToLower(format) {
	case "json":
		return slog.NewJSONHandler(os.Stdout, options), nil
	case "text":
		return slog.NewTextHandler(os.Stdout, options), nil
	default:
		return nil, fmt.Errorf("unknown log format %q", format)
	}
}
