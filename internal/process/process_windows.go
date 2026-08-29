//go:build windows
// +build windows

package process

import "os"

// interruptProcess on Windows: os.Interrupt is not deliverable to child
// processes via Signal(). We fall back to Kill() directly; the 3-second
// graceful timeout window in Stop() is the safety net before we reach here.
func interruptProcess(proc *os.Process) error {
	return proc.Kill()
}
