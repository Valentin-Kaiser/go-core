package log_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Valentin-Kaiser/go-core/logging"
	"github.com/Valentin-Kaiser/go-core/logging/log"
)

func TestLoggerSingleton(t *testing.T) {
	// Test basic logging functions exist and can be called
	t.Run("basic logging functions", func(t *testing.T) {
		// These should not panic
		log.Trace().Msg("trace message")
		log.Debug().Msg("debug message")
		log.Info().Msg("info message")
		log.Warn().Msg("warn message")
		log.Error().Msg("error message")
	})

	t.Run("error logging with fluent interface", func(t *testing.T) {
		err := errors.New("test error")
		// This should work like the requested usage: log.Info().Err(err).Msg("test")
		log.Info().Err(err).Msg("test")
	})

	t.Run("field logging", func(t *testing.T) {
		log.Info().Field("key", "value").Msg("message with field")
		log.Info().Fields(log.F("user", "john"), log.F("action", "login")).Msg("message with fields")
	})

	t.Run("level management", func(t *testing.T) {
		originalLevel := log.GetLevel()

		log.SetLevel(logging.ErrorLevel)
		if log.GetLevel() != logging.ErrorLevel {
			t.Errorf("Expected level %v, got %v", logging.ErrorLevel, log.GetLevel())
		}

		// Restore original level
		log.SetLevel(originalLevel)
	})

	t.Run("printf logging", func(t *testing.T) {
		log.Printf("formatted message: %s, number: %d", "test", 42)
	})

	t.Run("context and fields", func(t *testing.T) {
		// Test WithContext and WithFields return proper adapters
		ctxLogger := log.WithContext(context.TODO())
		if ctxLogger == nil {
			t.Error("WithContext should return a non-nil adapter")
		}

		fieldLogger := log.WithFields(log.F("component", "test"))
		if fieldLogger == nil {
			t.Error("WithFields should return a non-nil adapter")
		}
	})
}
