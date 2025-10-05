package logging

// NoOpEvent implements Event interface but does nothing
type NoOpEvent struct{}

// Fields does nothing and returns itself for chaining
func (e *NoOpEvent) Fields(fields ...Field) Event {
	return e
}

// Field does nothing and returns itself for chaining
func (e *NoOpEvent) Field(key string, value interface{}) Event {
	return e
}

// Err does nothing and returns itself for chaining
func (e *NoOpEvent) Err(err error) Event {
	return e
}

// Msg does nothing
func (e *NoOpEvent) Msg(msg string) {}

// Msgf does nothing
func (e *NoOpEvent) Msgf(format string, v ...interface{}) {}

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
func (n *NoOpAdapter) SetLevel(level Level) Adapter {
	n.level = level
	return n
}

// GetLevel returns the current log level
func (n *NoOpAdapter) GetLevel() Level {
	return n.level
}

// Trace returns a no-op event
func (n *NoOpAdapter) Trace() Event {
	return &NoOpEvent{}
}

// Debug returns a no-op event
func (n *NoOpAdapter) Debug() Event {
	return &NoOpEvent{}
}

// Info returns a no-op event
func (n *NoOpAdapter) Info() Event {
	return &NoOpEvent{}
}

// Warn returns a no-op event
func (n *NoOpAdapter) Warn() Event {
	return &NoOpEvent{}
}

// Error returns a no-op event
func (n *NoOpAdapter) Error() Event {
	return &NoOpEvent{}
}

// Fatal returns a no-op event
func (n *NoOpAdapter) Fatal() Event {
	return &NoOpEvent{}
}

// Panic returns a no-op event
func (n *NoOpAdapter) Panic() Event {
	return &NoOpEvent{}
}

func (n *NoOpAdapter) Printf(format string, v ...interface{}) {}

// WithPackage returns the same no-op adapter
func (n *NoOpAdapter) WithPackage(pkg string) Adapter {
	return n
}
