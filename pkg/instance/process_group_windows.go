//go:build windows

package instance

import (
	"syscall"
)

func setProcessGroup(pid int) error {
	return nil
}

func killProcessGroup(pid int) error {
	handle, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer handle.Close()
	return syscall.TerminateProcess(handle, 1)
}
