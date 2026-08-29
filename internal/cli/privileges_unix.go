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
	dir := customDir
	if dir == "" {
		if isAdmin() {
			dir = "/usr/local/bin"
		} else {
			home, _ := os.UserHomeDir()
			dir = filepath.Join(home, ".local", "bin")
		}
	}
	return dir, filepath.Join(dir, "gpm")
}
