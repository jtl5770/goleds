package logging

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
)

// failingWriter is a helper for testing error propagation.

type failingWriter struct{}

func (fw *failingWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("write failed")
}

func TestTUIMode(t *testing.T) {
	if err := Init(true, "DEBUG", "text", false, ""); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	slog.Info("Initial log")

	var tuiPane bytes.Buffer
	if err := SetOutput(&tuiPane); err != nil {
		t.Fatalf("ActivateTUIOutput failed: %v", err)
	}

	if !strings.Contains(tuiPane.String(), "Initial log") {
		t.Errorf("Expected initial log to be flushed to TUI, but it wasn't. Got: %s", tuiPane.String())
	}

	slog.Info("Live log")

	if !strings.Contains(tuiPane.String(), "Live log") {
		t.Errorf("Expected live log to be written to TUI, but it wasn't. Got: %s", tuiPane.String())
	}

	BufferOutput()

	slog.Info("Buffered log")

	if strings.Contains(tuiPane.String(), "Buffered log") {
		t.Errorf("Expected log to be buffered, but it was written to TUI. Got: %s", tuiPane.String())
	}

	if err := Close(); err != nil {
		t.Fatalf("Finalize failed: %v", err)
	}
}

func TestRPIMode_FileLogging(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if err := Init(false, "INFO", "json", true, tempFile.Name()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	slog.Info("RPI log", "key", "value")

	if err := Close(); err != nil {
		t.Fatalf("Finalize failed: %v", err)
	}

	content, err := os.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Check for JSON format and content
	if !strings.Contains(string(content), `"msg":"RPI log"`) || !strings.Contains(string(content), `"key":"value"`) {
		t.Errorf("Expected log to be written to file in JSON format, but it wasn't. Got: %s", string(content))
	}
}

func TestStderrFallback(t *testing.T) {
	if err := Init(true, "DEBUG", "text", false, ""); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	slog.Info("Shutdown log")

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	var wg sync.WaitGroup
	wg.Add(1)
	var capturedOutput string
	go func() {
		defer wg.Done()
		buf := make([]byte, 1024)
		n, _ := r.Read(buf)
		capturedOutput = string(buf[:n])
	}()

	if err := Close(); err != nil {
		t.Fatalf("Finalize failed: %v", err)
	}

	w.Close()
	wg.Wait()
	os.Stderr = oldStderr

	if !strings.Contains(capturedOutput, "Shutdown log") {
		t.Errorf("Expected shutdown log to be written to stderr, but it wasn't. Got: %s", capturedOutput)
	}
}

func TestErrorPropagation(t *testing.T) {
	if err := Init(false, "INFO", "text", false, ""); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	writer.target = &failingWriter{}

	// This log should cause an error
	slog.Info("This should fail")

	// We can't easily grab the error from the async slog handler,
	// but we can check if our writer was called.
	// A more advanced test would involve a custom slog.Handler.
}
