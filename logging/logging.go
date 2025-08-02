package logging

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// bufferingTeeWriter is a thread-safe writer that can buffer output and later
// flush it to a new destination. It can also tee output to a file.
type bufferingTeeWriter struct {
	mu          sync.Mutex
	buffer      *bytes.Buffer
	target      io.Writer
	file        *os.File
	isBuffering bool
}

func (w *bufferingTeeWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	var firstErr error

	// When buffering, we write to the buffer. bytes.Buffer.Write always returns a nil error.
	if w.isBuffering {
		w.buffer.Write(p)
	} else if w.target != nil {
		// When not buffering, write directly to the target.
		if _, err := w.target.Write(p); err != nil {
			firstErr = err
		}
	}

	// Always write to the file if it's configured.
	if w.file != nil {
		if _, err := w.file.Write(p); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return len(p), firstErr
}

var (
	defaultLogger *slog.Logger
	writer        *bufferingTeeWriter
)

// Init initializes the logging system.
func Init(bufferOutput bool, levelStr, formatStr string, logToFile bool, logFilePath string) error {
	writer = &bufferingTeeWriter{
		buffer:      &bytes.Buffer{},
		isBuffering: bufferOutput,
	}

	if logToFile {
		file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err != nil {
			return err
		}
		writer.file = file
	}

	var level slog.Level
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if strings.ToLower(formatStr) == "json" {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)

	return nil
}

// SetOutput flushes the buffer to the new writer and starts live logging.
func SetOutput(newTarget io.Writer) error {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	if writer.buffer.Len() > 0 {
		if _, err := newTarget.Write(writer.buffer.Bytes()); err != nil {
			return err // Return the error if flushing fails
		}
		writer.buffer.Reset()
	}

	writer.target = newTarget
	writer.isBuffering = false
	return nil
}

// BufferOutput stops live logging and starts buffering.
func BufferOutput() {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	writer.target = nil
	writer.isBuffering = true
}

// Close flushes any remaining logs and closes resources.
func Close() error {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	var firstErr error

	// If there's a file, ensure the buffer is flushed to it.
	if writer.file != nil {
		if writer.buffer.Len() > 0 {
			if _, err := writer.file.Write(writer.buffer.Bytes()); err != nil {
				firstErr = err
			}
		}
		if err := writer.file.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	} else if writer.target == nil {
		// If there's no file and no other target,
		// flush the buffer to stderr as a last resort.
		if writer.buffer.Len() > 0 {
			if _, err := os.Stderr.Write(writer.buffer.Bytes()); err != nil {
				firstErr = err
			}
		}
	}

	// Clear the buffer after flushing.
	writer.buffer.Reset()
	return firstErr
}
