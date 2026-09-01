package control

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// JSONRPCRequest represents a standard LMS JSON-RPC payload.
type JSONRPCRequest struct {
	ID     int           `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

// JSONRPCResponse represents an LMS JSON-RPC response envelope.
type JSONRPCResponse struct {
	ID     int             `json:"id"`
	Method string          `json:"method"`
	Params []interface{}   `json:"params"`
	Result json.RawMessage `json:"result"`
	Error  interface{}     `json:"error"`
}

// PlayerInfo holds summary information for a player reported by LMS.
type PlayerInfo struct {
	PlayerID  string `json:"playerid"`
	Name      string `json:"name"`
	Model     string `json:"model"`
	IsPlayer  int    `json:"isplayer"`
	Connected int    `json:"connected"`
	Power     int    `json:"power"`
}

// PlayerStatus holds the playback state of a player.
type PlayerStatus struct {
	PlayerID   string  `json:"playerid"`
	Name       string  `json:"player_name"`
	Mode       string  `json:"mode"`        // "play", "pause", "stop"
	Power      int     `json:"power"`       // 1 = on, 0 = off
	Connected  int     `json:"player_connected"`
	SyncMaster string  `json:"sync_master"` // Master MAC if synced as slave
	SyncSlaves string  `json:"sync_slaves"` // Comma-separated slave MACs if master
	Time       float64 `json:"time"`        // Current track playback position in seconds
	Duration   float64 `json:"duration"`    // Track duration in seconds
}

// LMSClient provides methods to query and control LMS via JSON-RPC.
type LMSClient struct {
	endpoint   string
	httpClient *http.Client
}

// NewLMSClient creates an LMS JSON-RPC client.
func NewLMSClient(host string, port int) *LMSClient {
	if port <= 0 {
		port = 9000
	}
	return &LMSClient{
		endpoint: fmt.Sprintf("http://%s:%d/jsonrpc.js", host, port),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *LMSClient) call(ctx context.Context, playerID string, command []interface{}, target interface{}) error {
	reqBody := JSONRPCRequest{
		ID:     1,
		Method: "slim.request",
		Params: []interface{}{playerID, command},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal json-rpc: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create http req: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do json-rpc req: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("json-rpc error HTTP %d: %s", resp.StatusCode, string(body))
	}

	var rpcResp JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return fmt.Errorf("decode json-rpc resp: %w", err)
	}

	if target != nil && len(rpcResp.Result) > 0 {
		if err := json.Unmarshal(rpcResp.Result, target); err != nil {
			return fmt.Errorf("unmarshal result into target: %w", err)
		}
	}

	return nil
}

// SetPlayerName sets the name of the player on LMS.
func (c *LMSClient) SetPlayerName(ctx context.Context, playerMAC, name string) error {
	cmd := []interface{}{"name", name}
	return c.call(ctx, playerMAC, cmd, nil)
}

// SetPlayerPref sets a player-specific preference on LMS.
func (c *LMSClient) SetPlayerPref(ctx context.Context, playerMAC, pref, value string) error {
	cmd := []interface{}{"playerpref", pref, value}
	return c.call(ctx, playerMAC, cmd, nil)
}

// GetPlayers retrieves the list of all connected players on LMS.
func (c *LMSClient) GetPlayers(ctx context.Context) ([]PlayerInfo, error) {
	var result struct {
		PlayersLoop []PlayerInfo `json:"players_loop"`
	}

	cmd := []interface{}{"players", 0, 100}
	if err := c.call(ctx, "", cmd, &result); err != nil {
		return nil, err
	}
	return result.PlayersLoop, nil
}

// GetPlayerStatus gets the current status ("play", "pause", "stop") of a player by MAC.
func (c *LMSClient) GetPlayerStatus(ctx context.Context, playerMAC string) (*PlayerStatus, error) {
	var result PlayerStatus
	cmd := []interface{}{"status", "-", 1, "tags:uB"}
	if err := c.call(ctx, playerMAC, cmd, &result); err != nil {
		return nil, err
	}
	if result.PlayerID == "" {
		result.PlayerID = playerMAC
	}
	return &result, nil
}

// SyncPlayer synchronizes our player (ourMAC, the slave) with a master player (targetMAC).
// In LMS JSON-RPC, the active master must be the target player in the request,
// with the slave (ourMAC) passed as the parameter: targetMAC ["sync", ourMAC].
func (c *LMSClient) SyncPlayer(ctx context.Context, ourMAC, targetMAC string) error {
	cmd := []interface{}{"sync", ourMAC}
	return c.call(ctx, targetMAC, cmd, nil)
}

// SyncPlayers is an alias for SyncPlayer.
func (c *LMSClient) SyncPlayers(ctx context.Context, masterMAC, slaveMAC string) error {
	return c.SyncPlayer(ctx, slaveMAC, masterMAC)
}

// UnsyncPlayer removes a player from any sync group.
func (c *LMSClient) UnsyncPlayer(ctx context.Context, playerMAC string) error {
	cmd := []interface{}{"sync", "-"}
	return c.call(ctx, playerMAC, cmd, nil)
}
