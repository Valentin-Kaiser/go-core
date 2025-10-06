// Package log provides a convenient global logger interface that wraps
// the core logging functionality. It offers simplified access to logging
// operations without requiring explicit logger instantiation.
package log

import (
	"github.com/valentin-kaiser/go-core/logging"
)

// Logger returns the current global logger adapter
func Logger() logging.Adapter {
	return logging.GetGlobalAdapter()
}

// SetLevel sets the log level for the global logger
func SetLevel(level logging.Level) {
	logging.GetGlobalAdapter().SetLevel(level)
}

// GetLevel returns the current log level
func GetLevel() logging.Level {
	return logging.GetGlobalAdapter().GetLevel()
}

// Trace returns a trace level event
func Trace() logging.Event {
	return logging.GetGlobalAdapter().Trace()
}

// Debug returns a debug level event
func Debug() logging.Event {
	return logging.GetGlobalAdapter().Debug()
}

// Info returns an info level event
func Info() logging.Event {
	return logging.GetGlobalAdapter().Info()
}

// Warn returns a warning level event
func Warn() logging.Event {
	return logging.GetGlobalAdapter().Warn()
}

// Error returns an error level event
func Error() logging.Event {
	return logging.GetGlobalAdapter().Error()
}

// Fatal returns a fatal level event
func Fatal() logging.Event {
	return logging.GetGlobalAdapter().Fatal()
}

// Panic returns a panic level event
func Panic() logging.Event {
	return logging.GetGlobalAdapter().Panic()
}

// Printf logs a formatted message
func Printf(format string, v ...interface{}) {
	logging.GetGlobalAdapter().Printf(format, v...)
}

// F is a helper function to create fields
func F(key string, value interface{}) logging.Field {
	return logging.F(key, value)
}
