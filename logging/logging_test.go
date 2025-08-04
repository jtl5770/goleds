package logging

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// resetGlobalState resets all global variables to their initial state for test isolation.
func resetGlobalState() {
	logBufferMutex.Lock()
	defer logBufferMutex.Unlock()

	// Stop the flusher if it's running
	if flusherStopCh != nil {
		close(flusherStopCh)
		flusherStopCh = nil
	}

	// Close any open file handles
	if fileOutput != nil {
		fileOutput.Close()
		fileOutput = nil
	}

	// Reset writers and flags
	tuiOutput = nil
	isBuffering = true // Default state

	// Clear the buffer
	logBuffer.Reset()

	// Reset sync.Once variables
	initOnce = sync.Once{}
	captureOnce = sync.Once{}
	captureOnce = sync.Once{}

	// Restore original stdio if they were captured
	if originalStdout != -1 {
		syscall.Dup2(originalStdout, int(os.Stdout.Fd()))
		syscall.Close(originalStdout)
		originalStdout = -1
	}
	if originalStderr != -1 {
		syscall.Dup2(originalStderr, int(os.Stderr.Fd()))
		syscall.Close(originalStderr)
		originalStderr = -1
	}
}

func TestBufferingAndFlushing(t *testing.T) {
	resetGlobalState()
	InitialSetup() // Capture stdout/stderr

	// 1. Initial state: Buffering enabled
	if err := Configure(true, "DEBUG", "text", false, ""); err != nil {
		t.Fatalf("Configure failed: %v", err)
	}

	slog.Info("First message, should be buffered.")

	// Give the flusher a moment to run (it shouldn't do anything)
	time.Sleep(150 * time.Millisecond)

	logBufferMutex.Lock()
	if !strings.Contains(logBuffer.String(), "First message") {
		t.Fatal("Log message was not written to the buffer in buffering mode.")
	}
	logBufferMutex.Unlock()

	// 2. Switch to TUI output
	var tuiPane bytes.Buffer
	SetOutput(&tuiPane)
	slog.Info("Switching to TUI output, should flush the buffer.")
	// The flusher should now activate and flush the buffer.
	time.Sleep(150 * time.Millisecond) // Wait for the flusher

	if !strings.Contains(tuiPane.String(), "First message") {
		t.Fatalf("Expected initial buffered log to be flushed to TUI, but it wasn't. Got: %s", tuiPane.String())
	}

	// 3. Live logging
	slog.Info("Second message, should be live.")
	time.Sleep(150 * time.Millisecond) // Wait for the flusher

	if !strings.Contains(tuiPane.String(), "Second message") {
		t.Fatalf("Expected live log to be written to TUI, but it wasn't. Got: %s", tuiPane.String())
	}

	// 4. Switch back to buffering
	BufferOutput()
	slog.Info("Third message, should be buffered again.")
	time.Sleep(150 * time.Millisecond)

	if strings.Contains(tuiPane.String(), "Third message") {
		t.Fatalf("Log was written to TUI pane while in buffering mode. Got: %s", tuiPane.String())
	}

	// 5. Finalize
	if err := Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestFileLogging(t *testing.T) {
	resetGlobalState()
	InitialSetup()

	tempFile, err := os.CreateTemp("", "test.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Configure for non-buffered (like RPI mode) with file output
	if err := Configure(false, "INFO", "json", true, tempFile.Name()); err != nil {
		t.Fatalf("Configure failed: %v", err)
	}

	slog.Info("RPI log", "key", "value")

	time.Sleep(150 * time.Millisecond) // Wait for flusher

	// Check file content after flush
	content, err := os.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), `"msg":"RPI log"`) || !strings.Contains(string(content), `"key":"value"`) {
		t.Errorf("Expected log to be written to file in JSON format, but it wasn't. Got: %s", string(content))
	}

	if err := Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestFinalFlushOnClose(t *testing.T) {
	resetGlobalState()
	InitialSetup()

	tempFile, err := os.CreateTemp("", "test.log.*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Configure for buffering mode but with a file, so we can check the final flush.
	if err := Configure(true, "INFO", "text", true, tempFile.Name()); err != nil {
		t.Fatalf("Configure failed: %v", err)
	}

	slog.Info("This log should be in the buffer.")

	// Close should perform a final flush to the file.
	if err := Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	content, err := os.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "This log should be in the buffer.") {
		t.Errorf("Expected final flush to write buffer to file on Close, but it didn't. Got: %s", string(content))
	}
}
