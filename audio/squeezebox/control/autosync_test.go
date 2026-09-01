package control

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestAutoSyncManager_SelfIgnoreAndSync(t *testing.T) {
	var mu sync.Mutex
	syncCalled := false
	unsyncCalled := false
	ourMAC := "00:04:20:ee:12:34"
	targetMAC := "00:04:20:99:88:77"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		mu.Lock()
		defer mu.Unlock()

		player := ""
		if len(req.Params) > 0 {
			if p, ok := req.Params[0].(string); ok {
				player = p
			}
		}

		var cmd string
		if len(req.Params) > 1 {
			if cmds, ok := req.Params[1].([]interface{}); ok && len(cmds) > 0 {
				cmd = cmds[0].(string)
			}
		}

		switch cmd {
		case "players":
			resp := map[string]interface{}{
				"players_loop": []map[string]interface{}{
					{
						"playerid": ourMAC,
						"name":     "GoLEDs VU",
					},
					{
						"playerid": targetMAC,
						"name":     "Living Room",
					},
				},
			}
			data, _ := json.Marshal(resp)
			_ = json.NewEncoder(w).Encode(JSONRPCResponse{Result: data})

		case "status":
			mode := "stop"
			if player == targetMAC {
				mode = "play"
			}
			resp := map[string]interface{}{
				"player_name": player,
				"mode":        mode,
			}
			data, _ := json.Marshal(resp)
			_ = json.NewEncoder(w).Encode(JSONRPCResponse{Result: data})

		case "sync":
			if len(req.Params) > 1 {
				cmds := req.Params[1].([]interface{})
				if len(cmds) > 1 {
					arg := cmds[1].(string)
					if arg == ourMAC && player == targetMAC {
						syncCalled = true
					}
					if arg == "-" && player == ourMAC {
						unsyncCalled = true
					}
				}
			}
			_ = json.NewEncoder(w).Encode(JSONRPCResponse{})
		}
	}))
	defer ts.Close()

	client := &LMSClient{
		endpoint:   ts.URL,
		httpClient: ts.Client(),
	}

	cfg := AutoSyncConfig{
		OurMAC:         ourMAC,
		OurName:        "GoLEDs VU",
		IgnoredPlayers: []string{},
		PollInterval:   20 * time.Millisecond,
	}

	mgr := NewAutoSyncManager(client, cfg)
	mgr.Start()

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if !syncCalled {
		t.Errorf("Expected SyncPlayer to be called on targetMAC with ourMAC")
	}
	mu.Unlock()

	mgr.Stop()

	mu.Lock()
	if !unsyncCalled {
		t.Errorf("Expected UnsyncPlayer to be called on Stop()")
	}
	mu.Unlock()
}

func TestAutoSyncManager_SwitchFromPausedToNewPlayer(t *testing.T) {
	var mu sync.Mutex
	ourMAC := "00:04:20:ee:12:34"
	player1MAC := "00:04:20:11:11:11"
	player2MAC := "00:04:20:22:22:22"

	player1Mode := "play"
	player2Mode := "stop"

	syncedTargets := make([]string, 0)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		mu.Lock()
		defer mu.Unlock()

		player := ""
		if len(req.Params) > 0 {
			if p, ok := req.Params[0].(string); ok {
				player = p
			}
		}

		var cmd string
		if len(req.Params) > 1 {
			if cmds, ok := req.Params[1].([]interface{}); ok && len(cmds) > 0 {
				cmd = cmds[0].(string)
			}
		}

		switch cmd {
		case "players":
			resp := map[string]interface{}{
				"players_loop": []map[string]interface{}{
					{"playerid": ourMAC, "name": "GoLEDs VU"},
					{"playerid": player1MAC, "name": "Living Room"},
					{"playerid": player2MAC, "name": "Kitchen"},
				},
			}
			data, _ := json.Marshal(resp)
			_ = json.NewEncoder(w).Encode(JSONRPCResponse{Result: data})

		case "status":
			mode := "stop"
			if player == player1MAC {
				mode = player1Mode
			} else if player == player2MAC {
				mode = player2Mode
			}
			resp := map[string]interface{}{
				"playerid":    player,
				"player_name": player,
				"mode":        mode,
			}
			data, _ := json.Marshal(resp)
			_ = json.NewEncoder(w).Encode(JSONRPCResponse{Result: data})

		case "sync":
			if len(req.Params) > 1 {
				cmds := req.Params[1].([]interface{})
				if len(cmds) > 1 {
					arg := cmds[1].(string)
					if arg == ourMAC {
						syncedTargets = append(syncedTargets, player)
					}
				}
			}
			_ = json.NewEncoder(w).Encode(JSONRPCResponse{})
		}
	}))
	defer ts.Close()

	client := &LMSClient{
		endpoint:   ts.URL,
		httpClient: ts.Client(),
	}

	cfg := AutoSyncConfig{
		OurMAC:         ourMAC,
		OurName:        "GoLEDs VU",
		IgnoredPlayers: []string{},
		PollInterval:   20 * time.Millisecond,
	}

	mgr := NewAutoSyncManager(client, cfg)
	mgr.Start()

	time.Sleep(60 * time.Millisecond)

	// Step 1: Verify it synced to Player 1 initially
	mac, _ := mgr.SyncedWith()
	if mac != player1MAC {
		t.Fatalf("Expected initial sync with Player 1 (%s), got: %s", player1MAC, mac)
	}

	// Step 2: Player 1 pauses (stops active playback) and Player 2 starts playing
	mu.Lock()
	player1Mode = "pause"
	player2Mode = "play"
	mu.Unlock()

	time.Sleep(100 * time.Millisecond)

	// Step 3: Verify AutoSync switched to Player 2
	mac, _ = mgr.SyncedWith()
	if mac != player2MAC {
		t.Fatalf("Expected auto-sync to switch to Player 2 (%s) after Player 1 paused, got: %s", player2MAC, mac)
	}

	mgr.Stop()
}
