package logging

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ZerologEvent wraps zerolog.Event to implement our Event interface
type ZerologEvent struct {
	event *zerolog.Event
}

// Fields adds structured fields to the event
func (e *ZerologEvent) Fields(fields ...Field) Event {
	for _, field := range fields {
		e.event = e.event.Interface(field.Key, field.Value)
	}
	return e
}

// Field adds a single structured field to the event
func (e *ZerologEvent) Field(key string, value interface{}) Event {
	e.event = e.event.Interface(key, value)
	return e
}

// Err adds an error to the event
func (e *ZerologEvent) Err(err error) Event {
	e.event = e.event.Err(err)
	return e
}

// Msg logs the message
func (e *ZerologEvent) Msg(msg string) {
	e.event.Msg(msg)
}

// Msgf logs the formatted message
func (e *ZerologEvent) Msgf(format string, v ...interface{}) {
	e.event.Msgf(format, v...)
}

// ZerologAdapter implements LogAdapter using zerolog
type ZerologAdapter struct {
	logger zerolog.Logger
	level  Level
}

// NewZerologAdapter creates a new zerolog adapter with the global zerolog logger
func NewZerologAdapter() Adapter {
	return &ZerologAdapter{
		logger: log.Logger,
		level:  InfoLevel,
	}
}

// NewZerologAdapterWithLogger creates a new zerolog adapter with a custom logger
func NewZerologAdapterWithLogger(logger zerolog.Logger) Adapter {
	return &ZerologAdapter{
		logger: logger,
		level:  InfoLevel,
	}
}

// SetLevel sets the log level
func (z *ZerologAdapter) SetLevel(level Level) Adapter {
	z.level = level
	z.logger = z.logger.Level(z.convertLevel(level))
	return z
}

// GetLevel returns the current log level
func (z *ZerologAdapter) GetLevel() Level {
	return z.level
}

// convertLevel converts our Level to zerolog.Level
func (z *ZerologAdapter) convertLevel(level Level) zerolog.Level {
	switch level {
	case TraceLevel:
		return zerolog.TraceLevel
	case DebugLevel:
		return zerolog.DebugLevel
	case InfoLevel:
		return zerolog.InfoLevel
	case WarnLevel:
		return zerolog.WarnLevel
	case ErrorLevel:
		return zerolog.ErrorLevel
	case FatalLevel:
		return zerolog.FatalLevel
	case PanicLevel:
		return zerolog.PanicLevel
	case DisabledLevel:
		return zerolog.Disabled
	default:
		return zerolog.InfoLevel
	}
}

// Trace returns a trace level event
func (z *ZerologAdapter) Trace() Event {
	return &ZerologEvent{event: z.logger.Trace()}
}

// Debug returns a debug level event
func (z *ZerologAdapter) Debug() Event {
	return &ZerologEvent{event: z.logger.Debug()}
}

// Info returns an info level event
func (z *ZerologAdapter) Info() Event {
	return &ZerologEvent{event: z.logger.Info()}
}

// Warn returns a warning level event
func (z *ZerologAdapter) Warn() Event {
	return &ZerologEvent{event: z.logger.Warn()}
}

// Error returns an error level event
func (z *ZerologAdapter) Error() Event {
	return &ZerologEvent{event: z.logger.Error()}
}

// Fatal returns a fatal level event
func (z *ZerologAdapter) Fatal() Event {
	return &ZerologEvent{event: z.logger.Fatal()}
}

// Panic returns a panic level event
func (z *ZerologAdapter) Panic() Event {
	return &ZerologEvent{event: z.logger.Panic()}
}

func (z *ZerologAdapter) Printf(format string, v ...interface{}) {
	z.logger.Printf(format, v...)
}

// WithContext returns a new adapter with context
func (z *ZerologAdapter) WithContext(ctx context.Context) Adapter {
	return &ZerologAdapter{
		logger: z.logger.With().Logger(),
		level:  z.level,
	}
}

// WithFields returns a new adapter with additional fields
func (z *ZerologAdapter) WithFields(fields ...Field) Adapter {
	logger := z.logger.With()
	for _, field := range fields {
		logger = logger.Interface(field.Key, field.Value)
	}
	return &ZerologAdapter{
		logger: logger.Logger(),
		level:  z.level,
	}
}

// WithPackage returns a new adapter with package name field
func (z *ZerologAdapter) WithPackage(pkg string) Adapter {
	return &ZerologAdapter{
		logger: z.logger.With().Str("package", pkg).Logger(),
		level:  z.level,
	}
}
