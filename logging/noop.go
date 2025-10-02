package logging

import "context"

// NoOpAdapter implements LogAdapter but does nothing
// This is the default implementation for minimal overhead when logging is disabled
type NoOpAdapter struct {
	level Level
}

// NewNoOpAdapter creates a new no-op adapter
func NewNoOpAdapter() Adapter {
	return &NoOpAdapter{level: DisabledLevel}
}

// SetLevel sets the log level (no-op)
func (n *NoOpAdapter) SetLevel(level Level) {
	n.level = level
}

// GetLevel returns the current log level
func (n *NoOpAdapter) GetLevel() Level {
	return n.level
}

// Trace does nothing
func (n *NoOpAdapter) Trace(msg string, fields ...Field) {}

// Debug does nothing
func (n *NoOpAdapter) Debug(msg string, fields ...Field) {}

// Info does nothing
func (n *NoOpAdapter) Info(msg string, fields ...Field) {}

// Warn does nothing
func (n *NoOpAdapter) Warn(msg string, fields ...Field) {}

// Error does nothing
func (n *NoOpAdapter) Error(msg string, fields ...Field) {}

// Fatal does nothing
func (n *NoOpAdapter) Fatal(msg string, fields ...Field) {}

// Panic does nothing
func (n *NoOpAdapter) Panic(msg string, fields ...Field) {}

func (n *NoOpAdapter) Printf(format string, v ...interface{}) {}

// WithContext returns the same no-op adapter
func (n *NoOpAdapter) WithContext(ctx context.Context) Adapter {
	return n
}

// WithFields returns the same no-op adapter
func (n *NoOpAdapter) WithFields(fields ...Field) Adapter {
	return n
}

// WithPackage returns the same no-op adapter
func (n *NoOpAdapter) WithPackage(pkg string) Adapter {
	return n
}
