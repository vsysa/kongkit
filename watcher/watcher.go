package watcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ChangeEvent represents the old and new configuration states.
type ChangeEvent[T any] struct {
	OldConfig T
	NewConfig T
}

// ControlFileChanges monitors changes to a specified file and sends detected updates through a channel.
// It supports debounce behavior, context-based graceful shutdown, and customizable error handling and logging.
//
// Parameters:
//   - ctx: Context for managing cancellation and timeout.
//   - pathToFile: The full path to the file being monitored.
//   - getCurrentConfigFn: A callback function to fetch the current configuration from the file.
//   - opts: Variadic options to customize behavior (e.g., debounce duration, error handler, logger).
//
// Returns:
//   - A read-only channel of ChangeEvent[T], which sends updates whenever the file changes.
//   - An error if the file watcher fails to initialize or encounters setup issues.
//
// The function ensures safe concurrent access, supports panic recovery within the configuration reader,
// and avoids excessive notifications using debounce logic.
func ControlFileChanges[T any](ctx context.Context, pathToFile string, getCurrentConfigFn func() T, opts ...Option) (<-chan ChangeEvent[T], error) {
	updates := make(chan ChangeEvent[T])
	var mutex sync.Mutex
	var debounceTimer *time.Timer

	options := defaultWatcherOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Initialize the configuration with the current state of the file.
	oldConfig := getCurrentConfigFn()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	err = watcher.Add(pathToFile)
	if err != nil {
		return nil, fmt.Errorf("failed to watch file %s: %w", pathToFile, err)
	}

	go func() {
		defer close(updates)
		defer func() {
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			mutex.Lock()
			defer mutex.Unlock()
			watcher.Close()
		}()

		eventChannel := make(chan fsnotify.Event, 1)
		defer close(eventChannel)

		// Goroutine for processing aggregated events with debounce logic
		// This ensures that rapid consecutive file changes trigger only one update after the debounce duration.
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case event := <-eventChannel:
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					debounceTimer = time.AfterFunc(options.debounceDuration, func() {
						defer func() {
							if r := recover(); r != nil {
								options.errorHandler(fmt.Errorf("panic in getCurrentConfigFn: %v", r))
							}
						}()
						mutex.Lock()
						defer mutex.Unlock()

						newConfig := getCurrentConfigFn()
						select {
						case <-ctx.Done():
							return
						default:
							updates <- ChangeEvent[T]{OldConfig: oldConfig, NewConfig: newConfig}
							oldConfig = newConfig
							if options.logger != nil {
								options.logger.Printf("File changed: %s", event.Name)
							}
						}
					})
				}
			}
		}()

		// Main watcher loop
		for {
			select {
			case <-ctx.Done():
				options.logger.Printf("Watcher stopped by context cancellation")
				return

			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Process only relevant file events (write or create)
				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					select {
					case eventChannel <- event:
					default:
						// Skip event if the event channel is full to avoid blocking
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				options.errorHandler(err)
			}
		}
	}()

	return updates, nil
}
