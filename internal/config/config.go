package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// ProcessConfig represents the configuration for a managed application process.
type ProcessConfig struct {
	Name         string            `json:"name"`
	Command      string            `json:"command"`
	Directory    string            `json:"directory"`
	Args         []string          `json:"args"`
	Environment  map[string]string `json:"environment"`
	AutoStart    bool              `json:"auto_start"`
	AutoRestart  bool              `json:"auto_restart"`
	RestartDelay int               `json:"restart_delay"` // in seconds
	MaxRestarts  int               `json:"max_restarts"`
}

// ConfigStore manages loading, saving, and querying configurations.
type ConfigStore struct {
	mu        sync.RWMutex
	configDir string
	filePath  string
}

// NewConfigStore initializes a new ConfigStore. It resolves the config path based on
// the OS configuration directory, or overrides it if the GPM_CONFIG_DIR env var is set.
func NewConfigStore() (*ConfigStore, error) {
	var dir string
	if envDir := os.Getenv("GPM_CONFIG_DIR"); envDir != "" {
		dir = envDir
	} else {
		userConfig, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user config dir: %w", err)
		}
		dir = filepath.Join(userConfig, "gpm")
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config dir: %w", err)
	}

	return &ConfigStore{
		configDir: dir,
		filePath:  filepath.Join(dir, "processes.json"),
	}, nil
}

// GetConfigDir returns the directory where GPM configurations are stored.
func (cs *ConfigStore) GetConfigDir() string {
	return cs.configDir
}

// Load reads and parses all process configurations from processes.json.
func (cs *ConfigStore) Load() (map[string]ProcessConfig, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	configs := make(map[string]ProcessConfig)

	file, err := os.Open(cs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// If file does not exist, return empty map
			return configs, nil
		}
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if len(data) == 0 {
		return configs, nil
	}

	var list []ProcessConfig
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	for _, cfg := range list {
		configs[cfg.Name] = cfg
	}

	return configs, nil
}

// Save atomically writes the map of process configurations back to processes.json.
func (cs *ConfigStore) Save(configs map[string]ProcessConfig) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var list []ProcessConfig
	for _, cfg := range configs {
		list = append(list, cfg)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configs: %w", err)
	}

	// Write atomically using a temporary file in the same directory.
	tmpFile, err := os.CreateTemp(cs.configDir, "processes.*.json.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp config file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		// Clean up the temp file if renaming fails or was cancelled
		if tmpFile != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	tmpFile = nil // prevent the defer clean-up from trying to close it again

	// Rename temp file to actual config file (atomic replacement)
	if err := os.Rename(tmpPath, cs.filePath); err != nil {
		return fmt.Errorf("failed to rename temp config file: %w", err)
	}

	return nil
}
