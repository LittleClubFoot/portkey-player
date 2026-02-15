package player

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// MPVCommand is a JSON-encoded IPC command for mpv.
type MPVCommand struct {
	Command   []any  `json:"command"`
	RequestID int64  `json:"request_id,omitempty"`
}

// MPVResponse is a JSON-encoded IPC response from mpv.
type MPVResponse struct {
	Error     string `json:"error"`
	Data      any    `json:"data"`
	RequestID int64  `json:"request_id"`
}

// IPCClient communicates with mpv over a Unix domain socket.
type IPCClient struct {
	socketPath string
	conn       net.Conn
	mu         sync.Mutex
	reqID      atomic.Int64
}

// NewIPCClient creates a client for the given mpv socket path.
func NewIPCClient(socketPath string) *IPCClient {
	return &IPCClient{socketPath: socketPath}
}

// Connect establishes the IPC connection, retrying until the socket is available.
func (c *IPCClient) Connect(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("unix", c.socketPath, time.Second)
		if err == nil {
			c.mu.Lock()
			c.conn = conn
			c.mu.Unlock()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("connecting to mpv socket %s: timed out after %v", c.socketPath, timeout)
}

// SendCommand sends a command to mpv and returns the response.
func (c *IPCClient) SendCommand(args ...any) (*MPVResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil, fmt.Errorf("ipc: not connected")
	}

	id := c.reqID.Add(1)
	cmd := MPVCommand{
		Command:   args,
		RequestID: id,
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("ipc: marshaling command: %w", err)
	}
	data = append(data, '\n')

	if err := c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, fmt.Errorf("ipc: setting write deadline: %w", err)
	}
	if _, err := c.conn.Write(data); err != nil {
		return nil, fmt.Errorf("ipc: writing command: %w", err)
	}

	if err := c.conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, fmt.Errorf("ipc: setting read deadline: %w", err)
	}

	buf := make([]byte, 4096)
	n, err := c.conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("ipc: reading response: %w", err)
	}

	var resp MPVResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		return nil, fmt.Errorf("ipc: parsing response: %w", err)
	}

	if resp.Error != "" && resp.Error != "success" {
		return &resp, fmt.Errorf("ipc: mpv error: %s", resp.Error)
	}

	return &resp, nil
}

// Close terminates the IPC connection.
func (c *IPCClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
