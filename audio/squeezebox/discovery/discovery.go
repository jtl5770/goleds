package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"
)

// DiscoverServer broadcasts a modern TLV discovery request and waits for the first valid LMS response.
func DiscoverServer(ctx context.Context, timeout time.Duration) (*ServerInfo, error) {
	return DiscoverServerOnPort(ctx, timeout, DefaultPort)
}

// DiscoverServerOnPort broadcasts a modern TLV discovery request to a specific port and waits for the first valid response.
func DiscoverServerOnPort(ctx context.Context, timeout time.Duration, port int) (*ServerInfo, error) {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	if port <= 0 {
		port = DefaultPort
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, fmt.Errorf("listen UDP for discovery: %w", err)
	}
	defer conn.Close()

	reqBytes := EncodeRequest("NAME", "IPAD", "JSON", "VERS", "UUID")

	broadcastAddrs := getBroadcastAddresses()
	for _, dst := range broadcastAddrs {
		target := &net.UDPAddr{IP: dst, Port: port}
		if _, err := conn.WriteToUDP(reqBytes, target); err != nil {
			slog.Debug("Failed to send discovery broadcast to", "addr", target.String(), "error", err)
		}
	}

	// Read loop until valid response or timeout
	buf := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("squeezebox auto-discovery timed out after %v: %w", timeout, ctx.Err())
		default:
		}

		deadline, ok := ctx.Deadline()
		if ok {
			_ = conn.SetReadDeadline(deadline)
		} else {
			_ = conn.SetReadDeadline(time.Now().Add(timeout))
		}

		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil, fmt.Errorf("squeezebox auto-discovery timed out: %w", ctx.Err())
			}
			return nil, fmt.Errorf("read discovery response: %w", err)
		}

		if n > 0 {
			var remoteIP net.IP
			if remoteAddr != nil {
				remoteIP = remoteAddr.IP
			}
			info, err := ParseResponse(buf[:n], remoteIP)
			if err != nil {
				slog.Debug("Ignoring invalid discovery response", "from", remoteAddr, "error", err)
				continue
			}

			slog.Info("Discovered LMS server",
				"name", info.Name,
				"host", info.Host,
				"slimproto_port", info.SlimProtoPort,
				"jsonrpc_port", info.JSONRPCPort,
				"version", info.Version)
			return info, nil
		}
	}
}

// getBroadcastAddresses returns global broadcast 255.255.255.255, 127.0.0.1, and all active subnet broadcast addresses.
func getBroadcastAddresses() []net.IP {
	addrs := []net.IP{net.IPv4bcast, net.IPv4(127, 0, 0, 1)}

	ifaces, err := net.Interfaces()
	if err != nil {
		return addrs
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		ifAddrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range ifAddrs {
			ipNet, ok := a.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil {
				continue
			}
			// Calculate broadcast IP
			ip := ipNet.IP.To4()
			mask := ipNet.Mask
			if len(mask) == 4 {
				bcast := net.IPv4(
					ip[0]|^mask[0],
					ip[1]|^mask[1],
					ip[2]|^mask[2],
					ip[3]|^mask[3],
				)
				addrs = append(addrs, bcast)
			}
		}
	}

	return addrs
}
