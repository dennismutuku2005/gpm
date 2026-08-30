package service

// ServiceManager interface handles registering GPM as a startup background service.
type ServiceManager interface {
	ConfigureStartup(configDir string) error
	GetStartupStatus() (string, error)
}

