//go:build windows

package cli

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// isAdmin checks if the current process is running with elevated administrator privileges on Windows.
func isAdmin() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		1,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.Token(0)
	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}
	return member
}

func configureSystemPath(targetDir string) error {
	return nil
}

func getInstallDestination(customDir string) (string, string) {
	dir := customDir
	if dir == "" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" {
			dir = filepath.Join(localAppData, "gpm", "bin")
		} else {
			dir = `C:\Program Files\gpm`
		}
	}
	return dir, filepath.Join(dir, "gpm.exe")
}
