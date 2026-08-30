//go:build linux

package process

import (
	"os"
	"syscall"
)

func interruptProcess(proc *os.Process) error {
	return proc.Signal(syscall.SIGTERM)
}
