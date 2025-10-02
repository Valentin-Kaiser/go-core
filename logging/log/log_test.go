package log

import (
	"errors"
	"testing"

	"github.com/Valentin-Kaiser/go-core/logging"
)

func TestLoggerSingleton(t *testing.T) {
	// Test basic logging functions exist and can be called
	t.Run("basic logging functions", func(t *testing.T) {
		// These should not panic
		Trace().Msg("trace message")
		Debug().Msg("debug message")
		Info().Msg("info message")
		Warn().Msg("warn message")
		Error().Msg("error message")
	})

	t.Run("error logging with fluent interface", func(t *testing.T) {
		err := errors.New("test error")
		// This should work like the requested usage: log.Info().Err(err).Msg("test")
		Info().Err(err).Msg("test")
	})

	t.Run("field logging", func(t *testing.T) {
		Info().Field("key", "value").Msg("message with field")
		Info().Fields(F("user", "john"), F("action", "login")).Msg("message with fields")
	})

	t.Run("level management", func(t *testing.T) {
		originalLevel := GetLevel()

		SetLevel(logging.ErrorLevel)
		if GetLevel() != logging.ErrorLevel {
			t.Errorf("Expected level %v, got %v", logging.ErrorLevel, GetLevel())
		}

		// Restore original level
		SetLevel(originalLevel)
	})

	t.Run("printf logging", func(t *testing.T) {
		Printf("formatted message: %s, number: %d", "test", 42)
	})

	t.Run("context and fields", func(t *testing.T) {
		// Test WithContext and WithFields return proper adapters
		ctxLogger := WithContext(nil)
		if ctxLogger == nil {
			t.Error("WithContext should return a non-nil adapter")
		}

		fieldLogger := WithFields(F("component", "test"))
		if fieldLogger == nil {
			t.Error("WithFields should return a non-nil adapter")
		}
	})
}
