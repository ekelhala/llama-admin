//go:build windows

package instance

import (
	"os/exec"
	"syscall"
)

func prepareProcessGroup(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200} // CREATE_NEW_PROCESS_GROUP
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
