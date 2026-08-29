//go:build windows
// +build windows

package ipc

import (
	"fmt"
	"net"
	"os/user"
	"strings"

	"github.com/Microsoft/go-winio"
)

// GetIPCPipe returns a user-specific Windows Named Pipe name.
func GetIPCPipe(configDir string) string {
	username := "default"
	if u, err := user.Current(); err == nil {
		username = u.Username
		// Handle DOMAIN\Username formats
		if idx := strings.LastIndex(username, "\\"); idx != -1 {
			username = username[idx+1:]
		}
		// Strip spaces or special chars to keep named pipe name safe
		username = strings.ReplaceAll(username, " ", "_")
	}
	return fmt.Sprintf(`\\.\pipe\gpm_%s`, username)
}

// ListenIPC opens a Windows Named Pipe listener with default security descriptors.
func ListenIPC(addr string) (net.Listener, error) {
	// Passing nil config will use the default security descriptor,
	// which allows the current user and administrators read/write access.
	return winio.ListenPipe(addr, nil)
}

// DialIPC connects to the Windows Named Pipe.
func DialIPC(addr string) (net.Conn, error) {
	return winio.DialPipe(addr, nil)
}
