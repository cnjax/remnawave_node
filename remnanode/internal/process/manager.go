package process

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
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

	lastPID       int
	lastExitError string
	lastExitTime  time.Time
	stdoutTail    []string
	stderrTail    []string
}

const (
	outputTailLimit       = 40
	startExitProbeTimeout = 500 * time.Millisecond
)

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
		m.mu.Unlock()
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
		m.mu.Unlock()
		log.Error().Err(err).Msg("Failed to get stdout pipe")
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := m.cmd.StderrPipe()
	if err != nil {
		m.mu.Unlock()
		log.Error().Err(err).Msg("Failed to get stderr pipe")
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start the process
	log.Info().Msg("Starting xray process...")
	if err := m.cmd.Start(); err != nil {
		m.mu.Unlock()
		log.Error().Err(err).Msg("Failed to start xray process")
		return fmt.Errorf("failed to start xray process: %w", err)
	}

	m.isRunning = true
	m.lastPID = m.cmd.Process.Pid
	m.lastExitError = ""
	m.lastExitTime = time.Time{}
	m.stdoutTail = nil
	m.stderrTail = nil
	log.Info().
		Str("binary", m.binaryPath).
		Int("pid", m.cmd.Process.Pid).
		Msg("Xray process started successfully")

	done := m.done
	cmd := m.cmd
	m.mu.Unlock()

	go m.streamOutput("stdout", stdout)
	go m.streamOutput("stderr", stderr)

	// Monitor process exit — the only place that writes isRunning=false.
	go func() {
		err := cmd.Wait()

		m.mu.Lock()
		if m.cmd == cmd {
			m.isRunning = false
			m.lastExitTime = time.Now()
			if err != nil {
				m.lastExitError = err.Error()
			} else {
				m.lastExitError = ""
			}
		}
		m.mu.Unlock()
		close(done)

		if err != nil {
			log.Warn().Err(err).Msg("Xray process exited with error")
		} else {
			log.Info().Msg("Xray process exited")
		}
	}()

	select {
	case <-done:
		time.Sleep(50 * time.Millisecond)
		if diagnostics := m.Diagnostics(); diagnostics != "" {
			return fmt.Errorf("xray process exited immediately after start: %s", diagnostics)
		}
		return fmt.Errorf("xray process exited immediately after start")
	case <-time.After(startExitProbeTimeout):
	}

	return nil
}

func (m *Manager) streamOutput(source string, reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if source == "stderr" {
			log.Error().Str("source", "xray").Msg(line)
		} else {
			log.Info().Str("source", "xray").Msg(line)
		}
		m.appendOutput(source, line)
	}
	if err := scanner.Err(); err != nil {
		log.Warn().Err(err).Str("source", source).Msg("Failed to read xray output")
	}
}

func (m *Manager) appendOutput(source, line string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if source == "stderr" {
		m.stderrTail = appendTail(m.stderrTail, line)
		return
	}
	m.stdoutTail = appendTail(m.stdoutTail, line)
}

func appendTail(lines []string, line string) []string {
	lines = append(lines, line)
	if len(lines) > outputTailLimit {
		return lines[len(lines)-outputTailLimit:]
	}
	return lines
}

// Stop stops the Xray process and waits until it has actually exited (up to 10 s).
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopLocked()
}

// stopLocked stops the process.
// Callers must hold m.mu. This method releases the mutex while waiting for the
// process to exit, then re-acquires it before returning so the caller's lock
// invariant (and any defer-unlock) remains consistent.
func (m *Manager) stopLocked() error {
	if !m.isRunning || m.cmd == nil || m.cmd.Process == nil {
		return nil // caller holds the lock; let caller (or its defer) unlock
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

// Diagnostics returns recent process exit/output details for startup failures.
func (m *Manager) Diagnostics() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.diagnosticsLocked()
}

func (m *Manager) diagnosticsLocked() string {
	parts := make([]string, 0, 4)
	if m.lastPID != 0 {
		parts = append(parts, fmt.Sprintf("pid=%d", m.lastPID))
	}
	if !m.lastExitTime.IsZero() {
		parts = append(parts, "exitTime="+m.lastExitTime.Format(time.RFC3339Nano))
	}
	if m.lastExitError != "" {
		parts = append(parts, "exitError="+m.lastExitError)
	}
	if len(m.stderrTail) > 0 {
		parts = append(parts, "stderr="+strings.Join(m.stderrTail, "\n"))
	} else if len(m.stdoutTail) > 0 {
		parts = append(parts, "stdout="+strings.Join(m.stdoutTail, "\n"))
	}
	return strings.Join(parts, "; ")
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
