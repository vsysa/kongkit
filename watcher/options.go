package watcher

import (
	"log"
	"time"
)

type ErrorHandler func(err error)
type Logger interface {
	Printf(format string, v ...interface{})
}
type NoOpLogger struct{}

func (n *NoOpLogger) Printf(format string, v ...interface{}) {}

type Options struct {
	errorHandler     ErrorHandler
	debounceDuration time.Duration
	logChanges       bool
	logger           Logger
}

func defaultWatcherOptions() *Options {
	return &Options{
		errorHandler: func(err error) {
			log.Printf("Watcher error: %v", err)
		},
		debounceDuration: 10 * time.Millisecond,
		logger:           &NoOpLogger{},
	}
}

// Option defines a function signature for setting WatcherOptions.
type Option func(*Options)

// WithErrorHandler
// This option allows setting a custom error handler for the watcher.
// The provided handler will be called whenever an error occurs during file monitoring.
// By default, errors are logged using the standard library's log.Printf.
func WithErrorHandler(handler ErrorHandler) Option {
	return func(o *Options) {
		o.errorHandler = handler
	}
}

// WithFileChangeLogging
// This option enables logging of file change events.
// When enabled, all detected file changes will be logged using the configured logger.
// This is useful for debugging or monitoring purposes.
func WithFileChangeLogging() Option {
	return func(o *Options) {
		o.logChanges = true
	}
}

// WithDebounce
// This option sets a debounce duration for file change events.
// When multiple rapid file changes occur, only the final change after the specified duration will trigger an event.
// This prevents excessive processing caused by frequent updates.
func WithDebounce(duration time.Duration) Option {
	return func(o *Options) {
		o.debounceDuration = duration
	}
}

// WithLogger
// This option allows injecting a custom logger for the watcher.
// The logger must implement the Logger interface, which includes the Printf method.
// By default, a NoOpLogger is used, which suppresses all log output.
func WithLogger(logger Logger) Option {
	return func(o *Options) {
		o.logger = logger
	}
}
