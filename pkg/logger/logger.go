package logger

import (
	"log/slog"
	"os"
)

// Setup initializes the global logger.
// Currently it outputs to stdout using a TextHandler, which is human-readable.
// This can be changed to JSONHandler or other transports in the future.
func Setup() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// Fatal logs an error message and then exits the application.
// slog doesn't have a Fatal method by default.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

// CronLogger adapts slog to the cron.Logger interface
type CronLogger struct {
	Logger *slog.Logger
}

func (l *CronLogger) Info(msg string, keysAndValues ...interface{}) {
	l.Logger.Info(msg, keysAndValues...)
}

func (l *CronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.Logger.Error(msg, append(keysAndValues, "error", err)...)
}
