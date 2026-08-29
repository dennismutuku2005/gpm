//go:build !windows
// +build !windows

package cli

import (
	"os"
	"path/filepath"
)

// isAdmin checks if the current process is running with root privileges on Unix/Linux systems.
func isAdmin() bool {
	return os.Getuid() == 0
}

// configureSystemPath on Linux usually doesn't need to append path since /usr/local/bin is already in PATH.
func configureSystemPath(targetDir string) error {
	return nil
}

// getInstallDestination returns the target binary copy destination.
func getInstallDestination(customDir string) (string, string) {
	dir := "/usr/local/bin"
	if customDir != "" {
		dir = customDir
	}
	return dir, filepath.Join(dir, "gpm")
}
