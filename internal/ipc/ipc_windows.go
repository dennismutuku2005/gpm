//go:build windows

package ipc

import (
	"net"
)

// GetIPCPipe returns the address for Windows IPC (TCP loopback).
func GetIPCPipe(configDir string) string {
	return "127.0.0.1:9457"
}

// ListenIPC creates a TCP loopback listener on Windows.
func ListenIPC(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}

// DialIPC connects to the TCP loopback listener on Windows.
func DialIPC(addr string) (net.Conn, error) {
	return net.Dial("tcp", addr)
}
