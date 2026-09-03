package control

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"strings"
	"sync"
	"time"
)

// AutoSyncConfig holds configuration parameters for the AutoSyncManager.
type AutoSyncConfig struct {
	OurMAC         string        // MAC address of our GoLEDs Squeezebox client
	OurName        string        // Player name of our GoLEDs client
	IgnoredPlayers []string      // List of player names or MACs to ignore
	PollInterval   time.Duration // Polling interval (e.g. 1500ms)
}

// AutoSyncManager continuously discovers active LMS players and synchronizes our client.
type AutoSyncManager struct {
	client LMSClientInterface
	cfg    AutoSyncConfig

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu         sync.Mutex
	syncedWith string // Currently synced player MAC
	syncedName string // Currently synced player Name
}

// LMSClientInterface defines the subset of LMSClient methods used by AutoSyncManager.
type LMSClientInterface interface {
	GetPlayers(ctx context.Context) ([]PlayerInfo, error)
	GetPlayerStatus(ctx context.Context, playerMAC string) (*PlayerStatus, error)
	SyncPlayer(ctx context.Context, ourMAC, targetMAC string) error
	UnsyncPlayer(ctx context.Context, ourMAC string) error
	SetPlayerPref(ctx context.Context, playerMAC, pref, value string) error
}

// NewAutoSyncManager creates a new AutoSyncManager.
func NewAutoSyncManager(client LMSClientInterface, cfg AutoSyncConfig) *AutoSyncManager {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 1500 * time.Millisecond
	}
	return &AutoSyncManager{
		client: client,
		cfg:    cfg,
	}
}

// Start begins the background monitoring goroutine.
func (m *AutoSyncManager) Start() {
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.wg.Add(1)
	go m.monitorLoop()
}

// Stop stops the background monitor and unsyncs our player from any active group.
func (m *AutoSyncManager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()

	// Unsync on shutdown with dedicated short timeout (500ms)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = m.client.UnsyncPlayer(ctx, m.cfg.OurMAC)

	m.mu.Lock()
	m.syncedWith = ""
	m.syncedName = ""
	m.mu.Unlock()

	slog.Info("AutoSyncManager stopped and cleanly unsynced from LMS")
}

// SyncedWith returns the MAC and Name of the player currently synced with.
func (m *AutoSyncManager) SyncedWith() (mac, name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.syncedWith, m.syncedName
}

func (m *AutoSyncManager) isIgnored(p PlayerInfo) bool {
	// Always ignore ourselves
	if strings.EqualFold(p.PlayerID, m.cfg.OurMAC) || strings.EqualFold(p.Name, m.cfg.OurName) {
		return true
	}
	for _, ign := range m.cfg.IgnoredPlayers {
		if strings.EqualFold(p.PlayerID, ign) || strings.EqualFold(p.Name, ign) {
			return true
		}
	}
	return false
}

func (m *AutoSyncManager) monitorLoop() {
	defer m.wg.Done()
	ticker := time.NewTicker(m.cfg.PollInterval)
	defer ticker.Stop()

	// Configure our player preferences on LMS so LMS knows GoLEDs is a passive visualizer:
	// maintainSync=0 tells LMS StreamingController::_CheckSync to NEVER enforce sync or adjust master playback for us.
	initCtx, initCancel := context.WithTimeout(m.ctx, 3*time.Second)
	_ = m.client.SetPlayerPref(initCtx, m.cfg.OurMAC, "maintainSync", "0")
	_ = m.client.SetPlayerPref(initCtx, m.cfg.OurMAC, "minSyncAdjust", "5000")
	initCancel()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.evaluatePlayers()
		}
	}
}

func (m *AutoSyncManager) evaluatePlayers() {
	ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
	defer cancel()

	// 1. Check our own player's status on LMS to see if we are already synced to a group
	currentMaster := ""
	ourStatus, err := m.client.GetPlayerStatus(ctx, m.cfg.OurMAC)
	if err == nil && ourStatus != nil && ourStatus.SyncMaster != "" {
		currentMaster = ourStatus.SyncMaster
	}
	if currentMaster == "" {
		m.mu.Lock()
		currentMaster = m.syncedWith
		m.mu.Unlock()
	}

	// 2. If currently synced to a player, check if it is actively playing
	if currentMaster != "" && !strings.EqualFold(currentMaster, m.cfg.OurMAC) {
		status, err := m.client.GetPlayerStatus(ctx, currentMaster)
		if err == nil && status != nil {
			if status.Mode == "play" {
				// Sticky lock: Current synced master is actively playing music
				m.mu.Lock()
				m.syncedWith = currentMaster
				if status.Name != "" {
					m.syncedName = status.Name
				}
				m.mu.Unlock()
				return
			}
			slog.Debug("AutoSyncManager: Synced master is not actively playing", "master", currentMaster, "mode", status.Mode)
		}
	}

	// 3. Discover available players that are actively in "play" mode
	players, err := m.client.GetPlayers(ctx)
	if err != nil {
		slog.Debug("AutoSyncManager: failed to fetch players", "error", err)
		return
	}

	type activeCandidate struct {
		player    PlayerInfo
		masterMAC string
	}
	var activeCandidates []activeCandidate

	for _, p := range players {
		if m.isIgnored(p) {
			continue
		}
		status, err := m.client.GetPlayerStatus(ctx, p.PlayerID)
		if err != nil || status == nil {
			continue
		}
		if status.Mode == "play" {
			targetMaster := p.PlayerID
			if status.SyncMaster != "" {
				targetMaster = status.SyncMaster
			}
			activeCandidates = append(activeCandidates, activeCandidate{
				player:    p,
				masterMAC: targetMaster,
			})
		}
	}

	if len(activeCandidates) == 0 {
		// No players are actively playing
		if currentMaster != "" {
			// If our current master was stopped/powered off, cleanly unsync
			m.mu.Lock()
			m.syncedWith = ""
			m.syncedName = ""
			m.mu.Unlock()
		}
		return
	}

	// 4. Pick an active candidate
	selected := activeCandidates[rand.IntN(len(activeCandidates))]
	targetMAC := selected.masterMAC
	targetName := selected.player.Name

	// If already synced to this target, nothing to do
	if strings.EqualFold(currentMaster, targetMAC) {
		m.mu.Lock()
		m.syncedWith = targetMAC
		m.syncedName = targetName
		m.mu.Unlock()
		return
	}

	slog.Info("AutoSyncManager: Automatically syncing to active player", "targetName", targetName, "targetMAC", targetMAC)

	if err := m.client.SyncPlayer(ctx, m.cfg.OurMAC, targetMAC); err != nil {
		slog.Warn("AutoSyncManager: Failed to sync player", "target", targetMAC, "error", err)
		return
	}

	// Ensure maintainSync=0 is asserted after joining the sync group
	_ = m.client.SetPlayerPref(ctx, m.cfg.OurMAC, "maintainSync", "0")
	_ = m.client.SetPlayerPref(ctx, m.cfg.OurMAC, "minSyncAdjust", "5000")

	m.mu.Lock()
	m.syncedWith = targetMAC
	m.syncedName = targetName
	m.mu.Unlock()
}
