//go:build windows
// +build windows

package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc/mgr"
)

// isAdmin checks if the current process is running with Administrator/elevated privileges on Windows.
func isAdmin() bool {
	m, err := mgr.Connect()
	if err != nil {
		return false
	}
	m.Disconnect()
	return true
}

// configureSystemPath registers the GPM installation folder in the Windows system environment PATH registry key.
func configureSystemPath(targetDir string) error {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `System\CurrentControlSet\Control\Session Manager\Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer k.Close()

	pathVal, _, err := k.GetStringValue("Path")
	if err != nil {
		return fmt.Errorf("failed to read Path registry value: %w", err)
	}

	targetDirLower := strings.ToLower(filepath.Clean(targetDir))
	paths := strings.Split(pathVal, ";")
	inPath := false
	for _, p := range paths {
		if strings.ToLower(filepath.Clean(p)) == targetDirLower {
			inPath = true
			break
		}
	}

	if !inPath {
		newPath := pathVal
		if !strings.HasSuffix(newPath, ";") && len(newPath) > 0 {
			newPath += ";"
		}
		newPath += targetDir
		err = k.SetStringValue("Path", newPath)
		if err != nil {
			return fmt.Errorf("failed to write Path to registry: %w", err)
		}
	}

	return nil
}

// getInstallDestination returns the target binary copy destination.
func getInstallDestination(customDir string) (string, string) {
	dir := `C:\Program Files\GPM`
	if customDir != "" {
		dir = customDir
	}
	return dir, filepath.Join(dir, "gpm.exe")
}
