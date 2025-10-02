package logging

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// StandardAdapter implements LogAdapter using Go's standard log package
type StandardAdapter struct {
	logger *log.Logger
	level  Level
}

// NewStandardAdapter creates a new standard log adapter with the default logger
func NewStandardAdapter() Adapter {
	return &StandardAdapter{
		logger: log.Default(),
		level:  InfoLevel,
	}
}

// NewStandardAdapterWithLogger creates a new standard log adapter with a custom logger
func NewStandardAdapterWithLogger(logger *log.Logger) Adapter {
	return &StandardAdapter{
		logger: logger,
		level:  InfoLevel,
	}
}

// StandardEvent wraps standard log functionality to implement our Event interface
type StandardEvent struct {
	adapter *StandardAdapter
	level   Level
	fields  []Field
	err     error
	caller  string
}

// Fields adds structured fields to the event
func (e *StandardEvent) Fields(fields ...Field) Event {
	e.fields = append(e.fields, fields...)
	return e
}

// Field adds a single field to the event
func (e *StandardEvent) Field(key string, value interface{}) Event {
	e.fields = append(e.fields, Field{Key: key, Value: value})
	return e
}

// Err adds an error to the event
func (e *StandardEvent) Err(err error) Event {
	e.err = err
	return e
}

// Msg logs the message with all accumulated fields
func (e *StandardEvent) Msg(msg string) {
	if !e.shouldLog() {
		return
	}

	logMsg := e.formatMessage(msg)

	switch e.level {
	case FatalLevel:
		e.adapter.logger.Fatal(logMsg)
	case PanicLevel:
		e.adapter.logger.Panic(logMsg)
	default:
		e.adapter.logger.Print(logMsg)
	}
}

// Msgf logs the formatted message with all accumulated fields
func (e *StandardEvent) Msgf(format string, v ...interface{}) {
	e.Msg(fmt.Sprintf(format, v...))
}

// shouldLog checks if the event should be logged based on the level
func (e *StandardEvent) shouldLog() bool {
	return e.level >= e.adapter.level
}

// formatMessage formats the message with level, fields, and error
func (e *StandardEvent) formatMessage(msg string) string {
	var parts []string

	// Add level prefix
	parts = append(parts, fmt.Sprintf("[%s]", strings.ToUpper(e.level.String())))

	if e.caller != "" {
		parts = append(parts, fmt.Sprintf("%s > ", e.caller))
	}

	// Add the main message
	parts = append(parts, msg)

	// Add fields
	for _, field := range e.fields {
		parts = append(parts, fmt.Sprintf("%s=%v", field.Key, field.Value))
	}

	// Add error if present
	if e.err != nil {
		parts = append(parts, fmt.Sprintf("error=%v", e.err))
	}

	return strings.Join(parts, " ")
}

// SetLevel sets the log level
func (s *StandardAdapter) SetLevel(level Level) Adapter {
	s.level = level
	return s
}

// GetLevel returns the current log level
func (s *StandardAdapter) GetLevel() Level {
	return s.level
}

// Trace returns a trace level event
func (s *StandardAdapter) Trace() Event {
	e := &StandardEvent{adapter: s, level: TraceLevel}
	if debug {
		e.caller = track()
	}
	return e
}

// Debug returns a debug level event
func (s *StandardAdapter) Debug() Event {
	e := &StandardEvent{adapter: s, level: DebugLevel}
	if debug {
		e.caller = track()
	}
	return e
}

// Info returns an info level event
func (s *StandardAdapter) Info() Event {
	e := &StandardEvent{adapter: s, level: InfoLevel}
	if debug {
		e.caller = track()
	}
	return e
}

// Warn returns a warning level event
func (s *StandardAdapter) Warn() Event {
	e := &StandardEvent{adapter: s, level: WarnLevel}
	if debug {
		e.caller = track()
	}
	return e
}

// Error returns an error level event
func (s *StandardAdapter) Error() Event {
	e := &StandardEvent{adapter: s, level: ErrorLevel}
	if debug {
		e.caller = track()
	}
	return e
}

// Fatal returns a fatal level event
func (s *StandardAdapter) Fatal() Event {
	e := &StandardEvent{adapter: s, level: FatalLevel}
	if debug {
		e.caller = track()
	}
	return e
}

// Panic returns a panic level event
func (s *StandardAdapter) Panic() Event {
	e := &StandardEvent{adapter: s, level: PanicLevel}
	if debug {
		e.caller = track()
	}
	return e
}

// Printf prints a formatted message
func (s *StandardAdapter) Printf(format string, v ...interface{}) {
	s.logger.Printf(format, v...)
}

// WithContext returns a new adapter with context (no-op for standard log)
func (s *StandardAdapter) WithContext(ctx context.Context) Adapter {
	return &StandardAdapter{
		logger: s.logger,
		level:  s.level,
	}
}

// WithFields returns a new adapter with additional fields
func (s *StandardAdapter) WithFields(fields ...Field) Adapter {
	// For standard log, we can't pre-add fields, so we return the same adapter
	// Fields will be handled at the event level
	return &StandardAdapter{
		logger: s.logger,
		level:  s.level,
	}
}

// WithPackage returns a new adapter with package name field
func (s *StandardAdapter) WithPackage(pkg string) Adapter {
	return &StandardAdapter{
		logger: s.logger,
		level:  s.level,
	}
}
