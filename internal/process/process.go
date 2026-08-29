package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/dennismutuku2005/gpm/internal/config"
	"github.com/dennismutuku2005/gpm/internal/logs"
)

type Status string

const (
	StatusStarting   Status = "starting"
	StatusOnline     Status = "online"
	StatusStopping   Status = "stopping"
	StatusStopped    Status = "stopped"
	StatusFailed     Status = "failed"
	StatusRestarting Status = "restarting"
)

// Process represents a running instance of a managed process.
type Process struct {
	mu              sync.RWMutex
	Config          config.ProcessConfig
	Status          Status
	PID             int
	StartTime       time.Time
	Restarts        int
	cmd             *exec.Cmd
	stdoutWriter    *logs.RotateWriter
	stderrWriter    *logs.RotateWriter
	configDir       string
	exitChan        chan struct{} // Closed when the process exits
	exitErr         error
	intentionalStop bool
}

// ProcessStats holds runtime information about a process for display.
type ProcessStats struct {
	Name        string
	Status      Status
	PID         int
	Uptime      time.Duration
	Restarts    int
	AutoStart   bool
	AutoRestart bool
	Command     string
}

// NewProcess creates a process instance.
func NewProcess(cfg config.ProcessConfig, configDir string) *Process {
	return &Process{
		Config:    cfg,
		Status:    StatusStopped,
		configDir: configDir,
	}
}

// Start launches the command and starts capturing output and tracking lifecycle.
func (p *Process) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Status == StatusOnline || p.Status == StatusStarting {
		return fmt.Errorf("process is already running")
	}

	p.intentionalStop = false
	p.Status = StatusStarting

	// Open logs
	stdoutPath, stderrPath := logs.GetLogPaths(p.configDir, p.Config.Name)
	stdoutWriter, err := logs.NewRotateWriter(stdoutPath)
	if err != nil {
		p.Status = StatusFailed
		return fmt.Errorf("failed to open stdout log: %w", err)
	}
	stderrWriter, err := logs.NewRotateWriter(stderrPath)
	if err != nil {
		stdoutWriter.Close()
		p.Status = StatusFailed
		return fmt.Errorf("failed to open stderr log: %w", err)
	}

	p.stdoutWriter = stdoutWriter
	p.stderrWriter = stderrWriter

	// Prepare command
	cmd := exec.Command(p.Config.Command, p.Config.Args...)
	cmd.Stdout = p.stdoutWriter
	cmd.Stderr = p.stderrWriter

	// Set directory
	if p.Config.Directory != "" {
		cmd.Dir = p.Config.Directory
	} else {
		cmd.Dir = filepath.Dir(p.Config.Command)
	}

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range p.Config.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if err := cmd.Start(); err != nil {
		p.stdoutWriter.Close()
		p.stderrWriter.Close()
		p.Status = StatusFailed
		return fmt.Errorf("failed to start command: %w", err)
	}

	p.cmd = cmd
	p.PID = cmd.Process.Pid
	p.StartTime = time.Now()
	p.Status = StatusOnline
	p.exitChan = make(chan struct{})
	p.exitErr = nil

	// Start monitor
	go p.monitor()

	return nil
}

// Stop terminates the process, first gracefully with SIGTERM/Interrupt, then with Kill if needed.
func (p *Process) Stop() error {
	p.mu.Lock()
	if p.Status != StatusOnline && p.Status != StatusStarting && p.Status != StatusRestarting {
		p.mu.Unlock()
		return nil
	}

	p.intentionalStop = true
	p.Status = StatusStopping
	cmd := p.cmd
	exitChan := p.exitChan
	p.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		// Attempt graceful interrupt
		_ = interruptProcess(cmd.Process)

		// Wait for process exit or timeout
		select {
		case <-exitChan:
			// Process exited gracefully
		case <-time.After(3 * time.Second):
			// Timeout, force kill
			_ = cmd.Process.Kill()
		}
	}

	p.mu.Lock()
	p.Status = StatusStopped
	p.PID = 0
	p.mu.Unlock()

	return nil
}

// monitor runs in a separate goroutine waiting for the process to exit.
func (p *Process) monitor() {
	err := p.cmd.Wait()

	p.mu.Lock()
	p.exitErr = err
	if p.Status == StatusOnline || p.Status == StatusStarting {
		if err != nil {
			p.Status = StatusFailed
		} else {
			p.Status = StatusStopped
		}
		p.PID = 0
	}
	if p.stdoutWriter != nil {
		p.stdoutWriter.Close()
	}
	if p.stderrWriter != nil {
		p.stderrWriter.Close()
	}
	close(p.exitChan)
	p.mu.Unlock()
}

// Wait returns a channel that is closed when the process terminates.
func (p *Process) Wait() <-chan struct{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.exitChan
}

// ExitErr returns the exit error of the process, if any.
func (p *Process) ExitErr() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.exitErr
}

// GetStats gathers stats for representation.
func (p *Process) GetStats() ProcessStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var uptime time.Duration
	if p.Status == StatusOnline && !p.StartTime.IsZero() {
		uptime = time.Since(p.StartTime)
	}

	return ProcessStats{
		Name:        p.Config.Name,
		Status:      p.Status,
		PID:         p.PID,
		Uptime:      uptime,
		Restarts:    p.Restarts,
		AutoStart:   p.Config.AutoStart,
		AutoRestart: p.Config.AutoRestart,
		Command:     p.Config.Command,
	}
}

// IncrementRestarts increments the restart count.
func (p *Process) IncrementRestarts() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Restarts++
}

// ResetRestarts resets the restart count.
func (p *Process) ResetRestarts() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Restarts = 0
}

// SetStatus directly sets the status of the process.
func (p *Process) SetStatus(s Status) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Status = s
}

// Lock locks the process.
func (p *Process) Lock() {
	p.mu.Lock()
}

// Unlock unlocks the process.
func (p *Process) Unlock() {
	p.mu.Unlock()
}

// RLock read-locks the process.
func (p *Process) RLock() {
	p.mu.RLock()
}

// RUnlock read-unlocks the process.
func (p *Process) RUnlock() {
	p.mu.RUnlock()
}

// IsIntentionalStop returns true if the process was stopped intentionally via Stop().
func (p *Process) IsIntentionalStop() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.intentionalStop
}

