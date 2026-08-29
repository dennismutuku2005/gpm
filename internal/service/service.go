package service

import (
	"github.com/dennismutuku2005/gpm/internal/manager"
)

// ServiceManager interface handles registering GPM as a startup background service.
type ServiceManager interface {
	ConfigureStartup(configDir string) error
	GetStartupStatus() (string, error)
}

// Global variables to check runtime context
var IsWindowsServiceFunc func() bool = func() bool { return false }
var RunWindowsServiceFunc func(m *manager.Manager) error = func(m *manager.Manager) error { return nil }

// IsWindowsService checks if GPM is running as a Windows Service.
func IsWindowsService() bool {
	return IsWindowsServiceFunc()
}

// RunWindowsService starts the Windows Service loop.
func RunWindowsService(m *manager.Manager) error {
	return RunWindowsServiceFunc(m)
}
