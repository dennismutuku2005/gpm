//go:build windows

package service

import "fmt"

type WindowsServiceManager struct{}

// NewServiceManager initializes the Windows service manager.
func NewServiceManager() ServiceManager {
	return &WindowsServiceManager{}
}

// ConfigureStartup configures startup on Windows.
func (w *WindowsServiceManager) ConfigureStartup(configDir string) error {
	return fmt.Errorf("windows service auto-start is not supported yet")
}

// GetStartupStatus returns startup status on Windows.
func (w *WindowsServiceManager) GetStartupStatus() (string, error) {
	return "not_supported (windows service integration not configured)", nil
}
