package discovery

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
)

// DefaultPort is the standard UDP and TCP port for Squeezebox / SlimProto discovery and control.
const DefaultPort = 3483

// DefaultJSONRPCPort is the standard HTTP / JSON-RPC port for Logitech Media Server.
const DefaultJSONRPCPort = 9000

// ServerInfo contains the details of a discovered Squeezebox server.
type ServerInfo struct {
	Host          string
	SlimProtoPort int
	JSONRPCPort   int
	Name          string
	Version       string
	UUID          string
}

// EncodeRequest creates a modern TLV discovery request packet starting with 'e'.
// Each requested tag is followed by a 0x00 length byte.
func EncodeRequest(tags ...string) []byte {
	if len(tags) == 0 {
		tags = []string{"NAME", "IPAD", "JSON", "VERS", "UUID"}
	}
	var buf bytes.Buffer
	buf.WriteByte('e')
	for _, tag := range tags {
		// Ensure tag is 4 bytes
		if len(tag) == 4 {
			buf.WriteString(tag)
			buf.WriteByte(0x00) // 0-length value indicates query
		}
	}
	return buf.Bytes()
}

// ParseResponse parses a modern TLV discovery response packet starting with 'E' (or 'e').
// If IPAD is not in the payload or empty, it falls back to remoteIP.
func ParseResponse(data []byte, remoteIP net.IP) (*ServerInfo, error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("empty discovery response")
	}

	if data[0] != 'E' && data[0] != 'e' {
		return nil, fmt.Errorf("invalid discovery response prefix: 0x%02x (%c)", data[0], data[0])
	}

	info := &ServerInfo{
		SlimProtoPort: DefaultPort,
		JSONRPCPort:   DefaultJSONRPCPort,
	}

	if remoteIP != nil {
		info.Host = remoteIP.String()
	}

	idx := 1
	for idx < len(data) {
		// Need at least 4 bytes for tag and 1 byte for length
		if idx+5 > len(data) {
			break
		}
		tag := string(data[idx : idx+4])
		valLen := int(data[idx+4])
		idx += 5

		if idx+valLen > len(data) {
			break
		}
		val := string(data[idx : idx+valLen])
		idx += valLen

		switch tag {
		case "NAME":
			info.Name = val
		case "IPAD":
			if val != "" {
				info.Host = val
			}
		case "JSON":
			if port, err := strconv.Atoi(val); err == nil && port > 0 && port <= 65535 {
				info.JSONRPCPort = port
			}
		case "VERS":
			info.Version = val
		case "UUID":
			info.UUID = val
		case "CLIP":
			// CLI port (e.g. 9090) if needed
		}
	}

	if info.Host == "" {
		return nil, fmt.Errorf("could not determine server host from discovery response")
	}

	return info, nil
}
