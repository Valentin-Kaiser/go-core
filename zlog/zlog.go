// zlog provides a simple and flexible logging utility built around the zerolog library.
//
// It offers structured, leveled logging with support for console output, file logging with rotation,
// and custom output targets. This package simplifies logger setup and management by wrapping zerolog
// and integrating with the lumberjack package for efficient log file rotation.
//
// Key Features:
//   - Structured logging using zerolog
//   - Optional console output with human-readable formatting
//   - File logging with automatic rotation (size, age, and backup limits)
//   - Singleton design for easy initialization and global logger access
//   - Runtime log level adjustments and custom writer support
//
// Example:
//
//	package main
//
//	import (
//		"github.com/Valentin-Kaiser/go-core/zlog"
//		"github.com/rs/zerolog"
//		"github.com/rs/zerolog/log"
//	)
//
//	func main() {
//		zlog.Logger().
//			WithConsole().
//			WithLogFile().
//			Init("example", zerolog.InfoLevel)
//
//		log.Info().Msg("This is an info message")
//
//		zlog.SetLevel(zerolog.DebugLevel)
//		log.Debug().Msg("This is a debug message")
//	}
package zlog

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Valentin-Kaiser/go-core/flag"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

type logger struct {
	level   zerolog.Level
	file    *lumberjack.Logger
	outputs []io.Writer
}

var instance = &logger{
	outputs: []io.Writer{},
}

// init initializes the logger with default settings.
func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
}

// Logger returns the singleton instance of the logger.
func Logger() *logger {
	return instance
}

// Init initializes the logger with the specified log name and log level.
// It sets the log file path and the global log level.
// If the log name does not end with ".log", it appends ".log" to the name.
func (l *logger) Init(logname string, loglevel zerolog.Level) {
	if !strings.HasSuffix(logname, ".log") {
		logname += ".log"
	}

	if l.file != nil {
		l.file.Filename = filepath.Join(flag.Path, logname)
	}

	zerolog.SetGlobalLevel(loglevel)
	log.Logger = log.Output(io.MultiWriter(l.outputs...))
}

// WithConsole adds a console writer to the logger outputs.
// It uses the zerolog.ConsoleWriter to format the log output for the console.
func (l *logger) WithConsole() *logger {
	l.outputs = append(l.outputs, zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	return l
}

// WithLogFile adds a file writer to the logger outputs.
// It uses the lumberjack package to handle log rotation and file management.
func (l *logger) WithLogFile() *logger {
	l.file = &lumberjack.Logger{
		MaxSize:    10, // megabytes
		MaxAge:     28, // days
		MaxBackups: 10, // number of backups
		Compress:   true,
	}
	l.outputs = append(l.outputs, l.file)
	return l
}

// With adds additional writers to the logger outputs.
// It can be used to add custom writers, such as file writers or network writers.
func (l *logger) With(writers ...io.Writer) *logger {
	l.outputs = append(l.outputs, writers...)
	return l
}

// Stop closes the log file.
// It should be called when the application is shutting down to ensure that all log entries are flushed to the file.
func (l *logger) Stop() {
	if l.file == nil {
		return
	}

	err := l.file.Close()
	if err != nil {
		log.Error().Err(err).Msgf("[Logger] failed to close log file")
	}
}

// Rotate rotates the log file manually.
// It creates a new log file and closes the old one.
func (l *logger) Rotate() {
	err := l.file.Rotate()
	if err != nil {
		log.Error().Err(err).Msgf("[Logger] failed to rotate log file")
	}
}

// SetLevel sets the global log level.
// It should be used to change the log level at runtime.
func (l *logger) SetLevel(level zerolog.Level) *logger {
	l.level = level
	zerolog.SetGlobalLevel(level)
	return l
}

// GetLevel returns the current global log level.
func (l *logger) GetLevel() zerolog.Level {
	return l.level
}

// WithLevel creates a new logger instance with the specified log level.
// It does not modify the global logger or the current logger instance.
// The returned logger should be used as a io.Writer to log messages at the specified level.
func (l *logger) WithLevel(level zerolog.Level) *logger {
	return &logger{
		level:   level,
		file:    l.file,
		outputs: l.outputs,
	}
}

// SetMaxSize sets the maximum size of the log file in megabytes.
// It should be used to limit the size of the log file and prevent it from growing indefinitely.
func (l *logger) SetMaxSize(size int) *logger {
	if l.file == nil {
		return l
	}
	l.file.MaxSize = size
	return l
}

// SetMaxAge sets the maximum age of the log file in days.
func (l *logger) SetMaxAge(age int) *logger {
	if l.file == nil {
		return l
	}
	l.file.MaxAge = age
	return l
}

// SetMaxBackups sets the maximum number of backup log files to keep.
func (l *logger) SetMaxBackups(backups int) *logger {
	if l.file == nil {
		return l
	}
	l.file.MaxBackups = backups
	return l
}

// SetCompress sets whether to compress the log files that are no longer needed.
func (l *logger) SetCompress(compress bool) *logger {
	if l.file == nil {
		return l
	}
	l.file.Compress = compress
	return l
}

// GetPath returns the path of the log file.
// It can be used to access the log file directly if needed.
func (l *logger) GetPath() string {
	if l.file == nil {
		return ""
	}
	return l.file.Filename
}

func (l *logger) Write(p []byte) (n int, err error) {
	log.WithLevel(l.level).Msg(strings.TrimSpace(string(p)))
	return len(p), nil
}
