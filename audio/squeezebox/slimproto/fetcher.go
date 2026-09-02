package slimproto

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync/atomic"
	"time"
)

// StreamMetadata encapsulates an active HTTP audio stream and its response headers.
type StreamMetadata struct {
	Headers    string
	Conn       net.Conn
	BodyReader io.Reader
}

// StreamFetcher defines the interface for dialing and establishing HTTP audio stream sessions.
type StreamFetcher interface {
	// Fetch dials the LMS streaming port, sends HTTP request headers, parses the response headers,
	// and returns an active stream handle with a byte-counting reader.
	Fetch(ctx context.Context, serverIP net.IP, serverPort uint16, fallbackHost string, httpHeader string, counter *atomic.Uint64) (*StreamMetadata, error)
}

// HTTPStreamer connects to LMS HTTP audio streaming ports (e.g. 9000).
type HTTPStreamer struct {
	dialer net.Dialer
}

// NewHTTPStreamer creates an initialized HTTPStreamer with a default connection timeout.
func NewHTTPStreamer(timeout time.Duration) *HTTPStreamer {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &HTTPStreamer{
		dialer: net.Dialer{Timeout: timeout},
	}
}

type countingReader struct {
	r       io.Reader
	counter *atomic.Uint64
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	if n > 0 && cr.counter != nil {
		cr.counter.Add(uint64(n))
	}
	return n, err
}

// Fetch establishes the streaming connection to LMS and reads HTTP response headers.
func (f *HTTPStreamer) Fetch(ctx context.Context, serverIP net.IP, serverPort uint16, fallbackHost string, httpHeader string, counter *atomic.Uint64) (*StreamMetadata, error) {
	targetIP := serverIP.String()
	if serverIP == nil || serverIP.IsUnspecified() || targetIP == "0.0.0.0" {
		targetIP = fallbackHost
	}
	if serverPort == 0 {
		serverPort = 9000
	}

	streamAddr := fmt.Sprintf("%s:%d", targetIP, serverPort)
	slog.Info("SlimProto connecting to audio stream", "addr", streamAddr)

	conn, err := f.dialer.DialContext(ctx, "tcp", streamAddr)
	if err != nil {
		return nil, fmt.Errorf("dial stream %s: %w", streamAddr, err)
	}

	// Send HTTP GET request
	if _, err := conn.Write([]byte(httpHeader)); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("write HTTP stream header: %w", err)
	}

	bufReader := bufio.NewReader(conn)

	// Read full raw HTTP response headers up to \r\n\r\n conforming to Squeezelite stream.c
	var headerBytes []byte
	for {
		line, err := bufReader.ReadString('\n')
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("reading HTTP response headers: %w", err)
		}
		headerBytes = append(headerBytes, []byte(line)...)
		if line == "\r\n" || line == "\n" {
			break
		}
		if len(headerBytes) > 8192 {
			_ = conn.Close()
			return nil, errors.New("HTTP response headers exceeded 8KB limit")
		}
	}

	countedReader := &countingReader{
		r:       bufReader,
		counter: counter,
	}

	return &StreamMetadata{
		Headers:    string(headerBytes),
		Conn:       conn,
		BodyReader: countedReader,
	}, nil
}
