//go:build linux

package service

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

type LinuxServiceManager struct{}

// NewServiceManager initializes the Linux service manager.
func NewServiceManager() ServiceManager {
	return &LinuxServiceManager{}
}

// ConfigureStartup creates and installs a systemd unit file for GPM.
func (l *LinuxServiceManager) ConfigureStartup(configDir string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.Abs(execPath)
	if err != nil {
		return err
	}

	currUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	systemdContent := fmt.Sprintf(`[Unit]
Description=Go Process Manager Daemon
After=network.target

[Service]
Type=simple
ExecStart=%s daemon
Restart=always
User=%s
Environment=GPM_CONFIG_DIR=%s

[Install]
WantedBy=multi-user.target
`, execPath, currUser.Username, configDir)

	servicePath := "/etc/systemd/system/gpm.service"
	
	// Try writing directly. If it fails, suggest using sudo or write to a temp file and copy.
	err = os.WriteFile(servicePath, []byte(systemdContent), 0644)
	if err != nil {
		// Attempting fallback: write to temporary file, then instruct user to run with sudo
		tmpPath := filepath.Join(os.TempDir(), "gpm.service")
		_ = os.WriteFile(tmpPath, []byte(systemdContent), 0644)

		return fmt.Errorf("permission denied writing to %s.\n"+
			"Please run GPM with root privileges, or execute the following manually:\n"+
			"  sudo cp %s %s\n"+
			"  sudo systemctl daemon-reload\n"+
			"  sudo systemctl enable gpm\n"+
			"  sudo systemctl start gpm", servicePath, tmpPath, servicePath)
	}

	// Reload systemd daemon
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Enable GPM service
	if err := exec.Command("systemctl", "enable", "gpm").Run(); err != nil {
		return fmt.Errorf("failed to enable gpm service: %w", err)
	}

	// Start GPM service
	if err := exec.Command("systemctl", "start", "gpm").Run(); err != nil {
		return fmt.Errorf("failed to start gpm service: %w", err)
	}

	return nil
}

// GetStartupStatus checks if the systemd gpm service is enabled and running.
func (l *LinuxServiceManager) GetStartupStatus() (string, error) {
	servicePath := "/etc/systemd/system/gpm.service"
	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		return "not_installed (systemd service file missing)", nil
	}

	// Check if active
	cmdActive := exec.Command("systemctl", "is-active", "gpm")
	outActive, _ := cmdActive.Output()
	activeStr := strings.TrimSpace(string(outActive))

	// Check if enabled
	cmdEnabled := exec.Command("systemctl", "is-enabled", "gpm")
	outEnabled, _ := cmdEnabled.Output()
	enabledStr := strings.TrimSpace(string(outEnabled))

	return fmt.Sprintf("installed (Systemd: %s, Enabled: %s)", activeStr, enabledStr), nil
}
