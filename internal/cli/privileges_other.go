//go:build !linux

package cli

import (
	"path/filepath"
)

func isAdmin() bool {
	return false
}

func configureSystemPath(targetDir string) error {
	return nil
}

func getInstallDestination(customDir string) (string, string) {
	dir := customDir
	if dir == "" {
		dir = "."
	}
	return dir, filepath.Join(dir, "gpm")
}
