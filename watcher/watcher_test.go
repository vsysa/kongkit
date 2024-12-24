package watcher

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to write to a file.
func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err, "Failed to write to file")
}

// Helper function to create a temporary file.
func createTempFile(t *testing.T, initialContent string) string {
	t.Helper()
	file, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err, "Failed to create temp file")
	defer file.Close()
	writeFile(t, file.Name(), initialContent)
	return file.Name()
}

// TestControlFileChanges_Basic
// This test verifies the core functionality of the ControlFileChanges function.
// It ensures that file modifications are correctly detected and that the old and new configurations are accurately reported.
// A temporary file is created, updated, and monitored for changes, with results validated against expected values.
func TestControlFileChanges_Basic(t *testing.T) {
	tempFile := createTempFile(t, "initial")
	defer os.Remove(tempFile)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	updates, err := ControlFileChanges(ctx, tempFile, func() string {
		data, _ := os.ReadFile(tempFile)
		return string(data)
	})
	require.NoError(t, err, "Failed to start watcher")

	// Trigger file change
	writeFile(t, tempFile, "updated")

	select {
	case event := <-updates:
		assert.Equal(t, "initial", event.OldConfig, "Old config should match initial value")
		assert.Equal(t, "updated", event.NewConfig, "New config should match updated value")
	case <-ctx.Done():
		t.Fatal("Timeout waiting for file change event")
	}
}

// TestControlFileChanges_WithDebounce
// This test evaluates the debounce behavior of ControlFileChanges.
// When multiple rapid updates are made to a file, only the final state after the debounce interval should trigger an update event.
// The test ensures intermediate changes are ignored and the last valid update is processed correctly.
func TestControlFileChanges_WithDebounce(t *testing.T) {
	tempFile := createTempFile(t, "initial")
	defer os.Remove(tempFile)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	updates, err := ControlFileChanges(ctx, tempFile, func() string {
		data, _ := os.ReadFile(tempFile)
		return string(data)
	}, WithDebounce(500*time.Millisecond))
	require.NoError(t, err, "Failed to start watcher with debounce")

	// Trigger multiple rapid changes
	writeFile(t, tempFile, "update1")
	writeFile(t, tempFile, "update2")
	writeFile(t, tempFile, "update3")

	// Wait for debounce period
	time.Sleep(1 * time.Second)

	select {
	case event := <-updates:
		assert.Equal(t, "initial", event.OldConfig, "Old config should match initial value")
		assert.Equal(t, "update3", event.NewConfig, "New config should match last update after debounce")
	case <-ctx.Done():
		t.Fatal("Timeout waiting for debounce event")
	}
}

// TestControlFileChanges_ErrorHandling
// This test ensures robust error handling in ControlFileChanges.
// It attempts to monitor an invalid file path and verifies that the function returns an appropriate error without crashing.
// This guarantees resilience against incorrect inputs or unexpected file access issues.
func TestControlFileChanges_ErrorHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := ControlFileChanges(ctx, "/invalid/path", func() string {
		return ""
	})
	assert.Error(t, err, "Expected an error for invalid file path")
}

// TestControlFileChanges_GracefulShutdownDuringLongConfigRead
// This test validates that ControlFileChanges handles context cancellation gracefully during long-running configuration reads.
// If the context is canceled while the configuration is being read, the watcher should exit cleanly without panicking.
// The test checks that the channel is properly closed after cancellation.
func TestControlFileChanges_GracefulShutdownDuringLongConfigRead(t *testing.T) {
	tempFile := createTempFile(t, "initial")
	defer os.Remove(tempFile)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

	updates, err := ControlFileChanges(ctx, tempFile, func() string {
		// Simulate long-running config read
		time.Sleep(1 * time.Second)
		data, _ := os.ReadFile(tempFile)
		return string(data)
	})
	require.NoError(t, err, "Failed to start watcher")

	// Trigger file change
	writeFile(t, tempFile, "updated")

	// Wait briefly and then cancel the context during config read
	time.Sleep(500 * time.Millisecond)
	cancel()

	// Ensure no panic and graceful shutdown
	select {
	case _, ok := <-updates:
		assert.False(t, ok, "Channel should be closed after context cancellation")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for watcher to close channel after context cancellation")
	}
}

// TestControlFileChanges_PanicRecoveryInConfigRead
// This test examines the panic recovery mechanism in ControlFileChanges.
// If the getCurrentConfigFn function panics during execution, the watcher must handle the panic gracefully and resume normal operation.
// The test ensures that a single failure does not disrupt ongoing file monitoring.
func TestControlFileChanges_PanicRecoveryInConfigRead(t *testing.T) {
	tempFile := createTempFile(t, "initial")
	defer os.Remove(tempFile)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	readCounter := 0

	updates, err := ControlFileChanges(ctx, tempFile, func() string {
		readCounter++
		// The first read is performed by the library to initialize the initial configuration value.
		if readCounter == 2 {
			panic("simulated panic in getCurrentConfigFn")
		}
		data, _ := os.ReadFile(tempFile)
		return string(data)
	}, WithDebounce(0))
	require.NoError(t, err, "Failed to start watcher")

	// Trigger file change
	writeFile(t, tempFile, "updatedWithPanic")
	writeFile(t, tempFile, "updated")

	// Ensure watcher recovers from panic and continues working
	select {
	case event := <-updates:
		assert.Equal(t, "initial", event.OldConfig, "Old config should match initial value")
		assert.Equal(t, "updated", event.NewConfig, "New config should match updated value after panic recovery")
	case <-ctx.Done():
		t.Fatal("Timeout waiting for watcher event after panic recovery")
	}
}
