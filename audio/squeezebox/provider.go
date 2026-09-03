package squeezebox

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"lautenbacher.net/goleds/audio"
	"lautenbacher.net/goleds/audio/squeezebox/control"
	"lautenbacher.net/goleds/audio/squeezebox/discovery"
	"lautenbacher.net/goleds/audio/squeezebox/slimproto"
)

// SqueezeboxAudioProvider implements audio.AudioProvider using SlimProto and LMS JSON-RPC.
type SqueezeboxAudioProvider struct {
	levels   *audio.AtomicLevels
	proto    *slimproto.Client
	autoSync *control.AutoSyncManager
}

// Config holds configuration parameters for the Squeezebox audio provider.
type Config struct {
	Server         string        `yaml:"Server"`
	SlimProtoPort  int           `yaml:"SlimProtoPort"`
	JSONRPCPort    int           `yaml:"JSONRPCPort"`
	PlayerMAC      string        `yaml:"PlayerMAC"`
	PlayerName     string        `yaml:"PlayerName"`
	IgnoredPlayers []string      `yaml:"IgnoredPlayers"`
	AutoSync       bool          `yaml:"AutoSync"`
	PollInterval   time.Duration `yaml:"PollInterval"`
}

// GeneratePlayerMAC derives a Squeezebox MAC address in the format 00:04:20:ee:Y:Z
// where Y and Z are derived from the host's primary network interface hardware address.
func GeneratePlayerMAC() net.HardwareAddr {
	var hostY, hostZ byte = 0x12, 0x34

	if ifaces, err := net.Interfaces(); err == nil {
		for _, iface := range ifaces {
			// Find first active, non-loopback interface with a valid MAC
			if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 && len(iface.HardwareAddr) >= 6 {
				hostY = iface.HardwareAddr[len(iface.HardwareAddr)-2]
				hostZ = iface.HardwareAddr[len(iface.HardwareAddr)-1]
				break
			}
		}
	} else if hostname, err := os.Hostname(); err == nil {
		hash := sha256.Sum256([]byte(hostname))
		hostY = hash[0]
		hostZ = hash[1]
	}

	return net.HardwareAddr{0x00, 0x04, 0x20, 0xee, hostY, hostZ}
}

// NewSqueezeboxAudioProvider creates an initialized SqueezeboxAudioProvider.
// If cfg.Server is empty, it uses modern UDP auto-discovery to locate LMS, ignoring any configured ports.
// If cfg.Server is specified, it connects to that host and uses the given ports or defaults.
func NewSqueezeboxAudioProvider(cfg Config) (*SqueezeboxAudioProvider, error) {
	var serverHost string
	var slimProtoPort int
	var jsonrpcPort int

	cleanServer := strings.TrimSpace(cfg.Server)
	if cleanServer == "" {
		slog.Info("Squeezebox Server not configured, initiating UDP auto-discovery...")
		discCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		discovered, err := discovery.DiscoverServer(discCtx, 3*time.Second)
		if err != nil {
			return nil, fmt.Errorf("squeezebox auto-discovery: %w", err)
		}
		serverHost = discovered.Host
		slimProtoPort = discovered.SlimProtoPort
		jsonrpcPort = discovered.JSONRPCPort
		slog.Info("Using auto-discovered LMS server",
			"name", discovered.Name,
			"host", serverHost,
			"slimproto_port", slimProtoPort,
			"jsonrpc_port", jsonrpcPort,
			"version", discovered.Version)
	} else {
		serverHost = cleanServer
		slimProtoPort = cfg.SlimProtoPort
		if slimProtoPort <= 0 {
			slimProtoPort = discovery.DefaultPort
		}
		jsonrpcPort = cfg.JSONRPCPort
		if jsonrpcPort <= 0 {
			jsonrpcPort = discovery.DefaultJSONRPCPort
		}
		slog.Info("Using statically configured LMS server",
			"host", serverHost,
			"slimproto_port", slimProtoPort,
			"jsonrpc_port", jsonrpcPort)
	}

	if cfg.PlayerName == "" {
		cfg.PlayerName = "GoLEDs VU"
	}
	pollInterval := cfg.PollInterval
	if pollInterval <= 0 {
		pollInterval = 1500 * time.Millisecond
	}

	var mac net.HardwareAddr
	cleanMAC := strings.TrimSpace(cfg.PlayerMAC)
	if cleanMAC == "" || strings.EqualFold(cleanMAC, "auto") {
		mac = GeneratePlayerMAC()
	} else {
		var err error
		mac, err = net.ParseMAC(cleanMAC)
		if err != nil {
			mac = GeneratePlayerMAC()
		}
	}

	levels := audio.NewAtomicLevels()
	serverSlim := fmt.Sprintf("%s:%d", serverHost, slimProtoPort)
	helo := slimproto.HeloConfig{
		MAC:        mac,
		DeviceID:   12, // SqueezePlay / SqueezeSlave
		Revision:   1,
		PlayerName: cfg.PlayerName,
	}

	protoClient := slimproto.NewClient(serverSlim, helo, levels)

	var autoSyncMgr *control.AutoSyncManager
	if cfg.AutoSync {
		lmsClient := control.NewLMSClient(serverHost, jsonrpcPort)
		syncConfig := control.AutoSyncConfig{
			OurMAC:         mac.String(),
			OurName:        cfg.PlayerName,
			IgnoredPlayers: cfg.IgnoredPlayers,
			PollInterval:   pollInterval,
		}
		autoSyncMgr = control.NewAutoSyncManager(lmsClient, syncConfig)
	}

	return &SqueezeboxAudioProvider{
		levels:   levels,
		proto:    protoClient,
		autoSync: autoSyncMgr,
	}, nil
}

// GetLevels returns the latest left/right dB levels atomically (0 allocs).
func (s *SqueezeboxAudioProvider) GetLevels() (leftDB, rightDB float64, playing bool) {
	return s.levels.Get()
}

// Start starts the SlimProto client and AutoSyncManager.
func (s *SqueezeboxAudioProvider) Start() error {
	if err := s.proto.Start(); err != nil {
		return fmt.Errorf("start slimproto client: %w", err)
	}
	if s.autoSync != nil {
		s.autoSync.Start()
	}
	return nil
}

// Stop stops the AutoSyncManager and closes SlimProto connection.
func (s *SqueezeboxAudioProvider) Stop() error {
	if s.autoSync != nil {
		s.autoSync.Stop()
	}
	return s.proto.Stop()
}
