//go:build !linux

package ipc

import (
	"fmt"
	"net"
)

func GetIPCPipe(configDir string) string {
	return ""
}

func ListenIPC(addr string) (net.Listener, error) {
	return nil, fmt.Errorf("IPC socket is only supported on Linux")
}

func DialIPC(addr string) (net.Conn, error) {
	return nil, fmt.Errorf("IPC socket is only supported on Linux")
}
