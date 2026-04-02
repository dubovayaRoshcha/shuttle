package config

import (
	"log/slog"
	"os"
	"strings"
)

var logger *slog.Logger

// InitLogger настраивает slog логгер
func InitLogger(level string) {
	lvl := slog.LevelInfo
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "error":
		lvl = slog.LevelError
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	})
	logger = slog.New(handler)

	logger.Info("Logger initialized", "level", lvl)
}

// фильтрация по уровню //

func Debug(msg string) {
	if logger != nil {
		logger.Debug(msg)
	}
}

func Info(msg string) {
	if logger != nil {
		logger.Info(msg)
	}
}

func Error(msg string) {
	if logger != nil {
		logger.Error(msg)
	}
}
