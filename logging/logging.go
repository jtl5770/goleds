package logging

import (
	"bytes"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"syscall"
)

// logWriter is a thread-safe writer that only writes to the central log buffer.
type logWriter struct{}

// Write handles writing bytes to the central logBuffer.
func (w *logWriter) Write(p []byte) (n int, err error) {
	logBufferMutex.Lock()
	n, err = logBuffer.Write(p)
	logBufferMutex.Unlock()

	// Signal the flusher only if we are not in buffering mode.
	if !isBuffering {
		select {
		case dataAvailableCh <- struct{}{}: // Signal that there is data
		default: // Do not block if the channel is full
		}
	}

	return
}

var (
	logBuffer       bytes.Buffer // Unified buffer
	logBufferMutex  sync.Mutex   // Mutex for logBuffer
	originalStdout  int          = -1
	originalStderr  int          = -1
	initOnce        sync.Once
	captureOnce     sync.Once
	flusherStopCh   chan struct{} // Channel to stop the flusher goroutine
	fileOutput      *os.File      // The file to write logs to
	isBuffering     bool          // Global buffering flag
	tuiOutput       io.Writer     // The TUI writer
	dataAvailableCh chan struct{} // Channel to signal new data
)

// InitialSetup redirects the default stdout and stderr to an internal pipe.
func InitialSetup() {
	captureOnce.Do(func() {
		var err error
		originalStdout, err = syscall.Dup(int(os.Stdout.Fd()))
		if err != nil {
			log.Fatalf("Failed to duplicate stdout: %v", err)
		}
		originalStderr, err = syscall.Dup(int(os.Stderr.Fd()))
		if err != nil {
			log.Fatalf("Failed to duplicate stderr: %v", err)
		}

		r, w, err := os.Pipe()
		if err != nil {
			log.Fatalf("Failed to create pipe: %v", err)
		}

		if err := syscall.Dup2(int(w.Fd()), int(os.Stdout.Fd())); err != nil {
			log.Fatalf("Failed to redirect stdout: %v", err)
		}
		if err := syscall.Dup2(int(w.Fd()), int(os.Stderr.Fd())); err != nil {
			log.Fatalf("Failed to redirect stderr: %v", err)
		}

		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := r.Read(buf)
				if n > 0 {
					logBufferMutex.Lock()
					logBuffer.Write(buf[:n])
					logBufferMutex.Unlock()

					// Signal the flusher only if we are not in buffering mode.
					if !isBuffering {
						select {
						case dataAvailableCh <- struct{}{}:
						default:
						}
					}
				}
				if err != nil {
					return
				}
			}
		}()
	})
}

// Configure initializes the logging system.
func Configure(bufferOutput bool, levelStr, formatStr string, logToFile bool, logFilePath string) error {
	initOnce.Do(func() {
		dataAvailableCh = make(chan struct{}, 1) // Buffer of 1 is important
		flusherStopCh = make(chan struct{})
		go startFlusher()
	})

	logBufferMutex.Lock()
	defer logBufferMutex.Unlock()

	isBuffering = bufferOutput

	if !bufferOutput {
		tuiOutput = os.NewFile(uintptr(originalStdout), "/dev/stdout")
	} else {
		tuiOutput = nil
	}

	if fileOutput != nil {
		fileOutput.Close()
		fileOutput = nil
	}
	if logToFile {
		file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err != nil {
			return err
		}
		fileOutput = file
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

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if strings.ToLower(formatStr) == "json" {
		handler = slog.NewJSONHandler(&logWriter{}, opts)
	} else {
		handler = slog.NewTextHandler(&logWriter{}, opts)
	}

	slog.SetDefault(slog.New(handler))

	return nil
}

// SetOutput sets the TUI writer and disables buffering mode.
func SetOutput(newTarget io.Writer) {
	logBufferMutex.Lock()
	defer logBufferMutex.Unlock()
	tuiOutput = newTarget
	isBuffering = false

	// Kickstart the flusher to process any buffered logs.
	select {
	case dataAvailableCh <- struct{}{}:
	default:
	}
}

// BufferOutput clears the TUI writer and enables buffering mode.
func BufferOutput() {
	logBufferMutex.Lock()
	defer logBufferMutex.Unlock()
	tuiOutput = nil
	isBuffering = true
}

// Close stops the flusher, performs a final flush, and restores stdout/stderr.
func Close() error {
	// Stop the periodic flusher
	if flusherStopCh != nil {
		close(flusherStopCh)
		flusherStopCh = nil
	}

	logBufferMutex.Lock()
	defer logBufferMutex.Unlock()

	// Restore stdio first, so the final flush goes to the console.
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

	// Perform one final, unconditional flush to the console and/or file.
	if logBuffer.Len() > 0 {
		bytes := logBuffer.Bytes()
		// Write to the restored os.Stdout
		os.Stdout.Write(bytes)

		if fileOutput != nil {
			fileOutput.Write(bytes)
		}
		logBuffer.Reset()
	}

	// Close the file handle
	var firstErr error
	if fileOutput != nil {
		if err := fileOutput.Close(); err != nil {
			firstErr = err
		}
		fileOutput = nil // prevent re-closing
	}

	return firstErr
}

// startFlusher starts a goroutine that periodically flushes the log buffer.
func startFlusher() {
	for {
		select {
		case <-dataAvailableCh:
			flushBuffer()
		case <-flusherStopCh:
			return
		}
	}
}

// flushBuffer writes the buffer content to the target(s) if not buffering.
func flushBuffer() {
	logBufferMutex.Lock()
	defer logBufferMutex.Unlock()

	if isBuffering || logBuffer.Len() == 0 {
		return
	}

	bytes := logBuffer.Bytes()
	if tuiOutput != nil {
		tuiOutput.Write(bytes)
	}
	if fileOutput != nil {
		fileOutput.Write(bytes)
	}

	logBuffer.Reset()
}
