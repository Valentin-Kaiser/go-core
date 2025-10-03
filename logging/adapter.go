package logging

import (
	"context"
)

// Level represents log levels
type Level int

const (
	TraceLevel    Level = -1
	DebugLevel    Level = 0
	InfoLevel     Level = 1
	WarnLevel     Level = 2
	ErrorLevel    Level = 3
	FatalLevel    Level = 4
	PanicLevel    Level = 5
	DisabledLevel Level = 6
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case TraceLevel:
		return "trace"
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	case PanicLevel:
		return "panic"
	case DisabledLevel:
		return "disabled"
	default:
		return "unknown"
	}
}

// Adapter defines the interface for internal logging
type Adapter interface {
	// Level control
	SetLevel(level Level) Adapter
	GetLevel() Level

	Trace() Event
	Debug() Event
	Info() Event
	Warn() Event
	Error() Event
	Fatal() Event
	Panic() Event

	Printf(format string, v ...interface{})

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

// Event represents a log event with a fluent interface
type Event interface {
	// Add fields to the event
	Fields(fields ...Field) Event

	Field(key string, value interface{}) Event

	// Add an error to the event
	Err(err error) Event

	// Log the message
	Msg(msg string)

	// Log the formatted message
	Msgf(format string, v ...interface{})
}
