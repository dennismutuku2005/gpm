//go:build windows
// +build windows

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

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

// configureSystemPath registers the GPM installation folder in the Windows environment PATH registry key.
func configureSystemPath(targetDir string) error {
	var rootKey registry.Key
	var subKey string

	if isAdmin() {
		rootKey = registry.LOCAL_MACHINE
		subKey = `System\CurrentControlSet\Control\Session Manager\Environment`
	} else {
		rootKey = registry.CURRENT_USER
		subKey = `Environment`
	}

	k, err := registry.OpenKey(rootKey, subKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer k.Close()

	pathVal, _, err := k.GetStringValue("Path")
	if err != nil && err != registry.ErrNotExist {
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
		// Clean up any literal %PATH% or empty elements from previous improper writes
		rawPaths := strings.Split(pathVal, ";")
		var cleanPaths []string
		for _, p := range rawPaths {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" && !strings.EqualFold(trimmed, "%PATH%") {
				cleanPaths = append(cleanPaths, trimmed)
			}
		}
		cleanPaths = append(cleanPaths, targetDir)
		newPath := strings.Join(cleanPaths, ";")

		err = k.SetExpandStringValue("Path", newPath)
		if err != nil {
			return fmt.Errorf("failed to write Path to registry: %w", err)
		}
		broadcastEnvironmentChange()
	}

	return nil
}

func broadcastEnvironmentChange() {
	user32 := syscall.NewLazyDLL("user32.dll")
	sendMessageTimeout := user32.NewProc("SendMessageTimeoutW")

	envStr, _ := syscall.UTF16PtrFromString("Environment")
	var result uintptr
	const (
		HWND_BROADCAST   = 0xffff
		WM_SETTINGCHANGE = 0x001A
		SMTO_ABORTIFHUNG = 0x0002
	)

	_, _, _ = sendMessageTimeout.Call(
		uintptr(HWND_BROADCAST),
		uintptr(WM_SETTINGCHANGE),
		0,
		uintptr(unsafe.Pointer(envStr)),
		uintptr(SMTO_ABORTIFHUNG),
		uintptr(5000),
		uintptr(unsafe.Pointer(&result)),
	)
}

// getInstallDestination returns the target binary copy destination.
func getInstallDestination(customDir string) (string, string) {
	dir := customDir
	if dir == "" {
		if isAdmin() {
			dir = `C:\Program Files\GPM`
		} else {
			localAppData := os.Getenv("LOCALAPPDATA")
			if localAppData == "" {
				home, _ := os.UserHomeDir()
				localAppData = filepath.Join(home, "AppData", "Local")
			}
			dir = filepath.Join(localAppData, "GPM")
		}
	}
	return dir, filepath.Join(dir, "gpm.exe")
}
