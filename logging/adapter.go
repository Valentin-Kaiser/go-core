package logging

import (
	"context"
)

// Level represents log levels
type Level int

const (
	LevelTrace Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
	LevelPanic
	LevelDisabled
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case LevelTrace:
		return "trace"
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	case LevelFatal:
		return "fatal"
	case LevelPanic:
		return "panic"
	case LevelDisabled:
		return "disabled"
	default:
		return "unknown"
	}
}

// Adapter defines the interface for internal logging
type Adapter interface {
	// Level control
	SetLevel(level Level)
	GetLevel() Level

	// Basic logging methods
	Trace(msg string, fields ...Field)
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	Panic(msg string, fields ...Field)

	// Context-aware logging
	WithContext(ctx context.Context) Adapter
	WithFields(fields ...Field) Adapter

	// Package-specific logger
	WithPackage(pkg string) Adapter
}

// Field represents a structured log field
type Field struct {
	Key   string
	Value interface{}
}

// F is a helper function to create fields
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}
