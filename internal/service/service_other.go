//go:build !linux

package service

import "fmt"

type UnimplementedServiceManager struct{}

// NewServiceManager fallback for non-Linux OS.
func NewServiceManager() ServiceManager {
	return &UnimplementedServiceManager{}
}

func (u *UnimplementedServiceManager) ConfigureStartup(configDir string) error {
	return fmt.Errorf("service management is only supported on Linux")
}

func (u *UnimplementedServiceManager) GetStartupStatus() (string, error) {
	return "unsupported", fmt.Errorf("service management is only supported on Linux")
}
