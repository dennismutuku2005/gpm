package manager

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dennismutuku2005/gpm/internal/config"
	"github.com/dennismutuku2005/gpm/internal/ipc"
	"github.com/dennismutuku2005/gpm/internal/logs"
	"github.com/dennismutuku2005/gpm/internal/process"
)

// Manager is the background daemon that orchestrates process execution,
// monitoring, crash recovery, and IPC request servicing.
type Manager struct {
	mu           sync.Mutex
	configStore  *config.ConfigStore
	processes    map[string]*process.Process
	ipcListener  net.Listener
	shutdownChan chan struct{}
	running      bool
}

// NewManager creates a new Manager instance.
func NewManager(cs *config.ConfigStore) *Manager {
	return &Manager{
		configStore:  cs,
		processes:    make(map[string]*process.Process),
		shutdownChan: make(chan struct{}),
	}
}

// Start boots the manager: it loads configs, autostarts designated applications,
// and opens the IPC listener socket/pipe.
func (m *Manager) Start() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = true
	m.mu.Unlock()

	// Load saved configurations
	cfgs, err := m.configStore.Load()
	if err != nil {
		return fmt.Errorf("failed to load configs: %w", err)
	}

	// Initialize process instances
	m.mu.Lock()
	for _, cfg := range cfgs {
		p := process.NewProcess(cfg, m.configStore.GetConfigDir())
		m.processes[cfg.Name] = p

		// Auto-start designated processes
		if cfg.AutoStart {
			go func(proc *process.Process) {
				if err := proc.Start(); err != nil {
					fmt.Fprintf(os.Stderr, "Daemon autostart failed for process %q: %v\n", proc.Config.Name, err)
				} else {
					go m.watchProcess(proc)
				}
			}(p)
		}
	}
	m.mu.Unlock()

	// Start IPC listener
	pipeName := ipc.GetIPCPipe(m.configStore.GetConfigDir())
	l, err := ipc.ListenIPC(pipeName)
	if err != nil {
		return fmt.Errorf("failed to listen on IPC: %w", err)
	}
	m.ipcListener = l

	// Serve connections
	go m.serveIPC()

	return nil
}

// Stop shuts down the manager, terminating all processes and closing the IPC listener.
func (m *Manager) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	close(m.shutdownChan)
	if m.ipcListener != nil {
		m.ipcListener.Close()
	}
	// Snapshot the process list while holding the lock to avoid
	// a race between the two lock acquisitions in the original code.
	procs := make([]*process.Process, 0, len(m.processes))
	for _, p := range m.processes {
		procs = append(procs, p)
	}
	m.mu.Unlock()

	// Stop all processes in parallel
	var wg sync.WaitGroup
	for _, p := range procs {
		wg.Add(1)
		go func(proc *process.Process) {
			defer wg.Done()
			_ = proc.Stop()
		}(p)
	}
	wg.Wait()
}

// Wait blocks until the manager receives a shutdown command.
func (m *Manager) Wait() {
	<-m.shutdownChan
}

// watchProcess monitors a running process and executes the restart policy if it crashes.
func (m *Manager) watchProcess(p *process.Process) {
	for {
		<-p.Wait()

		// If stopped intentionally, exit monitoring.
		if p.IsIntentionalStop() {
			return
		}

		if !p.Config.AutoRestart {
			p.SetStatus(process.StatusFailed)
			return
		}

		// Increment restart counter
		p.IncrementRestarts()
		stats := p.GetStats()

		// Check max restarts limit
		if p.Config.MaxRestarts > 0 && stats.Restarts > p.Config.MaxRestarts {
			p.SetStatus(process.StatusFailed)
			stderrPath := filepath.Join(m.configStore.GetConfigDir(), "logs", p.Config.Name, "stderr.log")
			if f, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
				_, _ = f.WriteString(fmt.Sprintf("\n[GPM] Process hit max restarts limit (%d). Marking as failed.\n", p.Config.MaxRestarts))
				f.Close()
			}
			return
		}

		p.SetStatus(process.StatusRestarting)

		// Wait restart delay
		delay := time.Duration(p.Config.RestartDelay) * time.Second
		if delay <= 0 {
			delay = 5 * time.Second
		}

		select {
		case <-time.After(delay):
			// Proceed to restart
		case <-m.shutdownChan:
			return
		}

		// Recheck if stop became intentional
		if p.IsIntentionalStop() {
			return
		}

		// Restart
		if err := p.Start(); err != nil {
			stderrPath := filepath.Join(m.configStore.GetConfigDir(), "logs", p.Config.Name, "stderr.log")
			if f, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
				_, _ = f.WriteString(fmt.Sprintf("\n[GPM] Crash restart attempt failed: %v. Retrying...\n", err))
				f.Close()
			}
		} else {
			// Process successfully started again. Wait on the new wait channel (next loop).
		}
	}
}

func (m *Manager) serveIPC() {
	for {
		conn, err := m.ipcListener.Accept()
		if err != nil {
			// If stopped, listener closed
			select {
			case <-m.shutdownChan:
				return
			default:
				continue
			}
		}
		go m.handleConnection(conn)
	}
}

func (m *Manager) handleConnection(conn net.Conn) {
	defer conn.Close()

	req, err := ipc.ReadRequest(conn)
	if err != nil {
		return
	}

	resp := m.dispatchRequest(req)
	_ = ipc.WriteResponse(conn, resp)
}

func (m *Manager) dispatchRequest(req *ipc.Request) *ipc.Response {
	switch req.Command {
	case "start":
		return m.handleStart(req)
	case "stop":
		return m.handleStop(req.Name)
	case "restart":
		return m.handleRestart(req.Name)
	case "delete":
		return m.handleDelete(req.Name)
	case "list":
		return m.handleList()
	case "status":
		return m.handleStatus()
	case "logs":
		return m.handleLogs(req)
	case "enable":
		return m.handleAutostart(req.Name, true)
	case "disable":
		return m.handleAutostart(req.Name, false)
	case "shutdown":
		go func() {
			time.Sleep(500 * time.Millisecond)
			m.Stop()
		}()
		return &ipc.Response{Success: true}
	default:
		return &ipc.Response{Success: false, Error: "unknown command"}
	}
}

func (m *Manager) handleStart(req *ipc.Request) *ipc.Response {
	m.mu.Lock()
	p, exists := m.processes[req.Name]
	m.mu.Unlock()

	var err error
	if !exists {
		// New process. Expect config in payload.
		if req.TargetConfig == nil {
			return &ipc.Response{Success: false, Error: fmt.Sprintf("Process %q is not registered and no config was provided.", req.Name)}
		}
		p = process.NewProcess(*req.TargetConfig, m.configStore.GetConfigDir())

		m.mu.Lock()
		m.processes[req.Name] = p
		m.mu.Unlock()

		// Persist configuration
		if err := m.saveConfigs(); err != nil {
			return &ipc.Response{Success: false, Error: fmt.Sprintf("failed to save config: %v", err)}
		}
	} else if req.TargetConfig != nil {
		// Update existing process config if supplied
		p.Lock()
		p.Config = *req.TargetConfig
		p.Unlock()

		if err := m.saveConfigs(); err != nil {
			return &ipc.Response{Success: false, Error: fmt.Sprintf("failed to save updated config: %v", err)}
		}
	}

	// Reset restarts counter on manual starting
	p.ResetRestarts()
	err = p.Start()
	if err != nil {
		return &ipc.Response{Success: false, Error: err.Error()}
	}

	// Watch lifecycle
	go m.watchProcess(p)

	return &ipc.Response{Success: true}
}

func (m *Manager) handleStop(name string) *ipc.Response {
	m.mu.Lock()
	p, exists := m.processes[name]
	m.mu.Unlock()

	if !exists {
		return &ipc.Response{Success: false, Error: "process not found"}
	}

	err := p.Stop()
	if err != nil {
		return &ipc.Response{Success: false, Error: err.Error()}
	}

	return &ipc.Response{Success: true}
}

func (m *Manager) handleRestart(name string) *ipc.Response {
	m.mu.Lock()
	p, exists := m.processes[name]
	m.mu.Unlock()

	if !exists {
		return &ipc.Response{Success: false, Error: "process not found"}
	}

	_ = p.Stop()
	p.ResetRestarts()
	err := p.Start()
	if err != nil {
		return &ipc.Response{Success: false, Error: err.Error()}
	}

	go m.watchProcess(p)

	return &ipc.Response{Success: true}
}

func (m *Manager) handleDelete(name string) *ipc.Response {
	m.mu.Lock()
	p, exists := m.processes[name]
	m.mu.Unlock()

	if !exists {
		return &ipc.Response{Success: false, Error: "process not found"}
	}

	// Stop it
	_ = p.Stop()

	// Delete config
	m.mu.Lock()
	delete(m.processes, name)
	m.mu.Unlock()

	if err := m.saveConfigs(); err != nil {
		return &ipc.Response{Success: false, Error: fmt.Sprintf("failed to save config database: %v", err)}
	}

	// Clean logs
	_ = logs.DeleteLogs(m.configStore.GetConfigDir(), name)

	return &ipc.Response{Success: true}
}

func (m *Manager) handleList() *ipc.Response {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats := make([]process.ProcessStats, 0, len(m.processes))
	for _, p := range m.processes {
		stats = append(stats, p.GetStats())
	}

	data, err := json.Marshal(stats)
	if err != nil {
		return &ipc.Response{Success: false, Error: err.Error()}
	}

	return &ipc.Response{Success: true, Data: data}
}

type GPMStatusData struct {
	DaemonRunning    bool `json:"daemon_running"`
	ManagedProcesses int  `json:"managed_processes"`
	Online           int  `json:"online"`
	Stopped          int  `json:"stopped"`
	Failed           int  `json:"failed"`
}

func (m *Manager) handleStatus() *ipc.Response {
	m.mu.Lock()
	defer m.mu.Unlock()

	var online, stopped, failed int
	for _, p := range m.processes {
		stats := p.GetStats()
		switch stats.Status {
		case process.StatusOnline:
			online++
		case process.StatusStopped:
			stopped++
		case process.StatusFailed:
			failed++
		default:
			stopped++ // starting/stopping grouped as stopped/offline for general overview
		}
	}

	data := GPMStatusData{
		DaemonRunning:    true,
		ManagedProcesses: len(m.processes),
		Online:           online,
		Stopped:          stopped,
		Failed:           failed,
	}

	b, err := json.Marshal(data)
	if err != nil {
		return &ipc.Response{Success: false, Error: err.Error()}
	}

	return &ipc.Response{Success: true, Data: b}
}

type LogLinesResponse struct {
	Stdout []string `json:"stdout"`
	Stderr []string `json:"stderr"`
}

func (m *Manager) handleLogs(req *ipc.Request) *ipc.Response {
	m.mu.Lock()
	_, exists := m.processes[req.Name]
	m.mu.Unlock()

	if !exists {
		return &ipc.Response{Success: false, Error: "process not found"}
	}

	stdoutPath, stderrPath := logs.GetLogPaths(m.configStore.GetConfigDir(), req.Name)

	var (
		stdoutLines []string
		stderrLines []string
		err         error
	)

	// Determine if filtering or tailing
	if req.Query != "" {
		stdoutLines, err = logs.LogFilter(stdoutPath, req.Query, req.Lines)
		if err != nil {
			return &ipc.Response{Success: false, Error: fmt.Sprintf("failed to filter stdout: %v", err)}
		}
		stderrLines, err = logs.LogFilter(stderrPath, req.Query, req.Lines)
		if err != nil {
			return &ipc.Response{Success: false, Error: fmt.Sprintf("failed to filter stderr: %v", err)}
		}
	} else {
		stdoutLines, err = logs.TailFile(stdoutPath, req.Lines)
		if err != nil {
			return &ipc.Response{Success: false, Error: fmt.Sprintf("failed to tail stdout: %v", err)}
		}
		stderrLines, err = logs.TailFile(stderrPath, req.Lines)
		if err != nil {
			return &ipc.Response{Success: false, Error: fmt.Sprintf("failed to tail stderr: %v", err)}
		}
	}

	data := LogLinesResponse{
		Stdout: stdoutLines,
		Stderr: stderrLines,
	}

	b, err := json.Marshal(data)
	if err != nil {
		return &ipc.Response{Success: false, Error: err.Error()}
	}

	return &ipc.Response{Success: true, Data: b}
}

func (m *Manager) handleAutostart(name string, state bool) *ipc.Response {
	m.mu.Lock()
	p, exists := m.processes[name]
	m.mu.Unlock()

	if !exists {
		return &ipc.Response{Success: false, Error: "process not found"}
	}

	p.Lock()
	p.Config.AutoStart = state
	p.Unlock()

	if err := m.saveConfigs(); err != nil {
		return &ipc.Response{Success: false, Error: fmt.Sprintf("failed to save configuration: %v", err)}
	}

	return &ipc.Response{Success: true}
}

func (m *Manager) saveConfigs() error {
	m.mu.Lock()
	cfgs := make(map[string]config.ProcessConfig)
	for name, p := range m.processes {
		p.RLock()
		cfgs[name] = p.Config
		p.RUnlock()
	}
	m.mu.Unlock()

	return m.configStore.Save(cfgs)
}
