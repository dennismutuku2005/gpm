//go:build windows

package process

import "os"

func interruptProcess(proc *os.Process) error {
	return proc.Kill()
}
