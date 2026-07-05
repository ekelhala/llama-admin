//go:build linux || darwin || freebsd || openbsd

package instance

import (
	"syscall"
)

func setProcessGroup(pid int) error {
	return syscall.Setpgid(pid, 0)
}

func killProcessGroup(pid int) error {
	return syscall.Kill(-pid, syscall.SIGTERM)
}
