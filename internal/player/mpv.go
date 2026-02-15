package player

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"
)

const defaultSocketPath = "/tmp/kidsmedia-mpv.sock"

// MPVPlayer controls media playback through the mpv binary and IPC socket.
type MPVPlayer struct {
	socketPath string
	args       []string
	cmd        *exec.Cmd
	ipc        *IPCClient
	playing    bool
	mu         sync.Mutex
	logger     *slog.Logger
}

// NewMPVPlayer creates a new mpv-based player with the given extra arguments.
func NewMPVPlayer(extraArgs []string, logger *slog.Logger) *MPVPlayer {
	return &MPVPlayer{
		socketPath: defaultSocketPath,
		args:       extraArgs,
		logger:     logger,
	}
}

func (m *MPVPlayer) Play(mediaPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop any existing playback.
	if m.cmd != nil {
		m.stopLocked()
	}

	// Remove stale socket.
	os.Remove(m.socketPath)

	args := []string{
		"--input-ipc-server=" + m.socketPath,
		"--idle=no",
	}
	args = append(args, m.args...)
	args = append(args, mediaPath)

	m.cmd = exec.Command("mpv", args...)
	m.cmd.Stdout = os.Stdout
	m.cmd.Stderr = os.Stderr

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("starting mpv: %w", err)
	}

	m.logger.Info("mpv started", "pid", m.cmd.Process.Pid, "media", mediaPath)

	// Connect to the IPC socket.
	m.ipc = NewIPCClient(m.socketPath)
	if err := m.ipc.Connect(5 * time.Second); err != nil {
		m.stopLocked()
		return fmt.Errorf("connecting to mpv IPC: %w", err)
	}

	m.playing = true
	return nil
}

func (m *MPVPlayer) Pause() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ipc == nil {
		return fmt.Errorf("player not running")
	}

	_, err := m.ipc.SendCommand("set_property", "pause", true)
	if err != nil {
		return fmt.Errorf("pausing: %w", err)
	}
	m.playing = false
	return nil
}

func (m *MPVPlayer) Resume() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ipc == nil {
		return fmt.Errorf("player not running")
	}

	_, err := m.ipc.SendCommand("set_property", "pause", false)
	if err != nil {
		return fmt.Errorf("resuming: %w", err)
	}
	m.playing = true
	return nil
}

func (m *MPVPlayer) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopLocked()
}

func (m *MPVPlayer) stopLocked() error {
	if m.ipc != nil {
		m.ipc.SendCommand("quit")
		m.ipc.Close()
		m.ipc = nil
	}

	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
		m.cmd.Wait()
		m.cmd = nil
	}

	m.playing = false
	os.Remove(m.socketPath)
	m.logger.Info("mpv stopped")
	return nil
}

func (m *MPVPlayer) Seek(seconds int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ipc == nil {
		return fmt.Errorf("player not running")
	}

	_, err := m.ipc.SendCommand("seek", seconds, "relative")
	if err != nil {
		return fmt.Errorf("seeking: %w", err)
	}
	return nil
}

func (m *MPVPlayer) SetVolume(level int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ipc == nil {
		return fmt.Errorf("player not running")
	}

	_, err := m.ipc.SendCommand("set_property", "volume", level)
	if err != nil {
		return fmt.Errorf("setting volume: %w", err)
	}
	return nil
}

func (m *MPVPlayer) IsPlaying() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.playing
}

func (m *MPVPlayer) GetPosition() (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ipc == nil {
		return 0, fmt.Errorf("player not running")
	}

	resp, err := m.ipc.SendCommand("get_property", "time-pos")
	if err != nil {
		return 0, fmt.Errorf("getting position: %w", err)
	}

	switch v := resp.Data.(type) {
	case float64:
		return v, nil
	default:
		return 0, fmt.Errorf("unexpected position type: %T", resp.Data)
	}
}

// WaitForEnd blocks until the mpv process exits.
func (m *MPVPlayer) WaitForEnd() error {
	m.mu.Lock()
	cmd := m.cmd
	m.mu.Unlock()

	if cmd == nil {
		return nil
	}

	err := cmd.Wait()

	m.mu.Lock()
	m.playing = false
	if m.ipc != nil {
		m.ipc.Close()
		m.ipc = nil
	}
	m.cmd = nil
	os.Remove(m.socketPath)
	m.mu.Unlock()

	if err != nil {
		// mpv returns non-zero on quit command; not always an error.
		m.logger.Debug("mpv exited", "error", err)
	}
	return nil
}
