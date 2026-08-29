//go:build !windows
// +build !windows

package ipc

import (
	"net"
	"os"
	"path/filepath"
)

// GetIPCPipe returns the Unix domain socket filepath for Linux/Unix.
func GetIPCPipe(configDir string) string {
	return filepath.Join(configDir, "gpm.sock")
}

// ListenIPC opens a Unix domain socket listener and restricts access to owner permissions.
func ListenIPC(addr string) (net.Listener, error) {
	_ = os.Remove(addr)
	l, err := net.Listen("unix", addr)
	if err != nil {
		return nil, err
	}
	// Restrict permissions to owner only (read/write)
	_ = os.Chmod(addr, 0600)
	return l, nil
}

// DialIPC connects to the Unix domain socket.
func DialIPC(addr string) (net.Conn, error) {
	return net.Dial("unix", addr)
}
