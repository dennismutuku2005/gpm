//go:build windows
// +build windows

package service

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"github.com/dennismutuku2005/gpm/internal/manager"
)

type WindowsServiceManager struct{}

// NewServiceManager initializes the Windows service manager.
func NewServiceManager() ServiceManager {
	return &WindowsServiceManager{}
}

func init() {
	IsWindowsServiceFunc = func() bool {
		isSvc, err := svc.IsAnInteractiveSession()
		if err != nil {
			return false
		}
		return !isSvc
	}

	RunWindowsServiceFunc = func(m *manager.Manager) error {
		return svc.Run("GPM", &gpmService{manager: m})
	}
}

type gpmService struct {
	manager *manager.Manager
}

// Execute is the entry point for the Windows Service control loop.
func (s *gpmService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	// Start the daemon manager in the background
	go func() {
		if err := s.manager.Start(); err != nil {
			// Log to event log or print
			os.Exit(1)
		}
	}()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for c := range r {
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			changes <- svc.Status{State: svc.StopPending}
			s.manager.Stop()
			break loop
		default:
			// Ignore other commands
		}
	}

	changes <- svc.Status{State: svc.Stopped}
	return
}

// ConfigureStartup installs GPM as a Windows Service pointing to the current executable, or as User Autostart registry key if non-admin.
func (w *WindowsServiceManager) ConfigureStartup(configDir string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.Abs(execPath)
	if err != nil {
		return err
	}

	m, err := mgr.Connect()
	if err == nil {
		defer m.Disconnect()

		// Check if already exists
		s, err := m.OpenService("GPM")
		if err == nil {
			s.Close()
			return fmt.Errorf("GPM Windows Service is already installed")
		}

		cfg := mgr.Config{
			DisplayName:      "Go Process Manager",
			Description:      "Manages and monitors background Go applications.",
			StartType:        mgr.StartAutomatic,
			ServiceStartName: "", // LocalSystem
		}

		// We pass "daemon" and the configuration directory override arguments
		s, err = m.CreateService("GPM", execPath, cfg, "daemon", "--config-dir", configDir)
		if err == nil {
			defer s.Close()
			_ = s.Start()
			return nil
		}
	}

	// Fallback for non-admin user: Register in HKCU Run registry key for user logon autostart
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return nil // Non-critical: Daemon will auto-boot on first gpm command anyway
	}
	defer k.Close()

	cmd := fmt.Sprintf(`"%s" daemon --config-dir "%s"`, execPath, configDir)
	_ = k.SetStringValue("GPMDaemon", cmd)
	return nil
}

// GetStartupStatus checks the SCM for the GPM service and its state, or HKCU user registry autostart key.
func (w *WindowsServiceManager) GetStartupStatus() (string, error) {
	m, err := mgr.Connect()
	if err == nil {
		defer m.Disconnect()

		s, err := m.OpenService("GPM")
		if err == nil {
			defer s.Close()

			status, err := s.Query()
			if err == nil {
				cfg, err := s.Config()
				if err == nil {
					var state string
					switch status.State {
					case svc.Stopped:
						state = "Stopped"
					case svc.StartPending:
						state = "StartPending"
					case svc.StopPending:
						state = "StopPending"
					case svc.Running:
						state = "Running"
					default:
						state = "Unknown"
					}

					var startType string
					switch cfg.StartType {
					case mgr.StartAutomatic:
						startType = "Automatic"
					case mgr.StartManual:
						startType = "Manual"
					case mgr.StartDisabled:
						startType = "Disabled"
					default:
						startType = "Unknown"
					}

					return fmt.Sprintf("installed (Windows Service: %s, StartType: %s)", state, startType), nil
				}
			}
		}
	}

	// Check HKCU Run key for User Autostart
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		if val, _, err := k.GetStringValue("GPMDaemon"); err == nil && val != "" {
			return "installed (User Autostart registry key)", nil
		}
	}

	return "not_installed (Auto-start service/registry key not configured)", nil
}
