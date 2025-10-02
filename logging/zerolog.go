package logging

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ZerologAdapter implements LogAdapter using zerolog
type ZerologAdapter struct {
	logger zerolog.Logger
	level  Level
}

// NewZerologAdapter creates a new zerolog adapter with the global zerolog logger
func NewZerologAdapter() Adapter {
	return &ZerologAdapter{
		logger: log.Logger,
		level:  LevelInfo,
	}
}

// NewZerologAdapterWithLogger creates a new zerolog adapter with a custom logger
func NewZerologAdapterWithLogger(logger zerolog.Logger) Adapter {
	return &ZerologAdapter{
		logger: logger,
		level:  LevelInfo,
	}
}

// SetLevel sets the log level
func (z *ZerologAdapter) SetLevel(level Level) {
	z.level = level
	z.logger = z.logger.Level(z.convertLevel(level))
}

// GetLevel returns the current log level
func (z *ZerologAdapter) GetLevel() Level {
	return z.level
}

// convertLevel converts our Level to zerolog.Level
func (z *ZerologAdapter) convertLevel(level Level) zerolog.Level {
	switch level {
	case LevelTrace:
		return zerolog.TraceLevel
	case LevelDebug:
		return zerolog.DebugLevel
	case LevelInfo:
		return zerolog.InfoLevel
	case LevelWarn:
		return zerolog.WarnLevel
	case LevelError:
		return zerolog.ErrorLevel
	case LevelFatal:
		return zerolog.FatalLevel
	case LevelPanic:
		return zerolog.PanicLevel
	case LevelDisabled:
		return zerolog.Disabled
	default:
		return zerolog.InfoLevel
	}
}

// addFields adds structured fields to a zerolog event
func (z *ZerologAdapter) addFields(event *zerolog.Event, fields []Field) *zerolog.Event {
	for _, field := range fields {
		event = event.Interface(field.Key, field.Value)
	}
	return event
}

// Trace logs a trace message
func (z *ZerologAdapter) Trace(msg string, fields ...Field) {
	z.addFields(z.logger.Trace(), fields).Msg(msg)
}

// Debug logs a debug message
func (z *ZerologAdapter) Debug(msg string, fields ...Field) {
	z.addFields(z.logger.Debug(), fields).Msg(msg)
}

// Info logs an info message
func (z *ZerologAdapter) Info(msg string, fields ...Field) {
	z.addFields(z.logger.Info(), fields).Msg(msg)
}

// Warn logs a warning message
func (z *ZerologAdapter) Warn(msg string, fields ...Field) {
	z.addFields(z.logger.Warn(), fields).Msg(msg)
}

// Error logs an error message
func (z *ZerologAdapter) Error(msg string, fields ...Field) {
	z.addFields(z.logger.Error(), fields).Msg(msg)
}

// Fatal logs a fatal message
func (z *ZerologAdapter) Fatal(msg string, fields ...Field) {
	z.addFields(z.logger.Fatal(), fields).Msg(msg)
}

// Panic logs a panic message
func (z *ZerologAdapter) Panic(msg string, fields ...Field) {
	z.addFields(z.logger.Panic(), fields).Msg(msg)
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
