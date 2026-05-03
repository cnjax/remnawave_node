package process

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

// Manager manages the Xray process directly
type Manager struct {
	mu sync.Mutex

	cmd        *exec.Cmd
	cancelFunc context.CancelFunc
	done       chan struct{} // closed by the monitor goroutine when cmd.Wait() returns

	binaryPath string
	configURL  string
	isRunning  bool
}

// NewManager creates a new process manager
func NewManager(binaryPath, configURL string) *Manager {
	return &Manager{
		binaryPath: binaryPath,
		configURL:  configURL,
		isRunning:  false,
	}
}

// Start starts the Xray process
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Info().
		Str("binary", m.binaryPath).
		Str("config", m.configURL).
		Msg("Process manager Start() called")

	// Stop existing process if running
	if m.isRunning && m.cmd != nil {
		log.Info().Msg("Stopping existing process before starting new one")
		m.stopLocked()
	}

	// Check if binary exists
	if _, err := os.Stat(m.binaryPath); os.IsNotExist(err) {
		log.Error().Str("binary", m.binaryPath).Msg("Binary does not exist")
		return fmt.Errorf("binary not found: %s", m.binaryPath)
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel
	m.done = make(chan struct{})

	// Create command: /usr/local/bin/rw-core -config http://127.0.0.1:61001/internal/get-config -format json
	m.cmd = exec.CommandContext(ctx, m.binaryPath, "-config", m.configURL, "-format", "json")
	log.Info().
		Str("command", fmt.Sprintf("%s -config %s -format json", m.binaryPath, m.configURL)).
		Msg("Command created")

	// Get stdout and stderr pipes
	stdout, err := m.cmd.StdoutPipe()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get stdout pipe")
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := m.cmd.StderrPipe()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get stderr pipe")
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start the process
	log.Info().Msg("Starting xray process...")
	if err := m.cmd.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start xray process")
		return fmt.Errorf("failed to start xray process: %w", err)
	}

	m.isRunning = true
	log.Info().
		Str("binary", m.binaryPath).
		Int("pid", m.cmd.Process.Pid).
		Msg("Xray process started successfully")

	// Stream stdout in goroutine
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			log.Info().Str("source", "xray").Msg(scanner.Text())
		}
	}()

	// Stream stderr in goroutine
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Error().Str("source", "xray").Msg(scanner.Text())
		}
	}()

	// Monitor process exit — the only place that writes isRunning=false.
	done := m.done
	go func() {
		defer close(done)
		err := m.cmd.Wait()

		m.mu.Lock()
		m.isRunning = false
		m.mu.Unlock()

		if err != nil {
			log.Warn().Err(err).Msg("Xray process exited with error")
		} else {
			log.Info().Msg("Xray process exited")
		}
	}()

	return nil
}

// Stop stops the Xray process and waits until it has actually exited (up to 10 s).
func (m *Manager) Stop() error {
	m.mu.Lock()
	return m.stopLocked()
}

// stopLocked stops the process.
// It releases the mutex while waiting for the process to exit, then re-acquires it.
// Callers must hold m.mu before calling this.
func (m *Manager) stopLocked() error {
	if !m.isRunning || m.cmd == nil || m.cmd.Process == nil {
		m.mu.Unlock()
		return nil
	}

	log.Info().Int("pid", m.cmd.Process.Pid).Msg("Stopping Xray process")

	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	// Graceful SIGTERM first.
	if err := m.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		m.cmd.Process.Kill() //nolint:errcheck
	}

	// Capture the done channel before releasing the lock so we can wait on it.
	done := m.done
	m.mu.Unlock()

	// Wait for the monitor goroutine to confirm exit, with a hard-kill fallback.
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		m.mu.Lock()
		if m.cmd != nil && m.cmd.Process != nil {
			m.cmd.Process.Kill() //nolint:errcheck
		}
		m.mu.Unlock()
		// Wait for the goroutine to finish after the kill.
		<-done
	}

	m.mu.Lock()
	log.Info().Msg("Xray process stopped")
	return nil
}

// Restart restarts the Xray process
func (m *Manager) Restart() error {
	if err := m.Stop(); err != nil {
		log.Warn().Err(err).Msg("Error stopping xray process during restart")
	}

	return m.Start()
}

// IsRunning returns whether the process is running
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check by testing if process is still alive
	if m.isRunning && m.cmd != nil && m.cmd.Process != nil {
		// Sending signal 0 checks if process exists
		err := m.cmd.Process.Signal(syscall.Signal(0))
		if err != nil {
			m.isRunning = false
		}
	}

	return m.isRunning
}

// GetPID returns the process ID if running
func (m *Manager) GetPID() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil && m.cmd.Process != nil {
		return m.cmd.Process.Pid
	}
	return 0
}

// Cleanup ensures the process is stopped
func (m *Manager) Cleanup() {
	m.Stop()
}

// CheckBinary verifies the binary exists and is executable
func (m *Manager) CheckBinary() error {
	info, err := os.Stat(m.binaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("xray binary not found at %s", m.binaryPath)
		}
		return fmt.Errorf("failed to stat xray binary: %w", err)
	}

	// Check if executable
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("xray binary is not executable: %s", m.binaryPath)
	}

	return nil
}
