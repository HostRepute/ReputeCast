//go:build unix

package testdb

import (
	"errors"
	"syscall"
)

// processAlive reports whether a process with this PID runs on this host.
func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
