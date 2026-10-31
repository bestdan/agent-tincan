//go:build unix

package cli

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// checkWorkerPlatform reports why this platform cannot run per-request
// workers.
func checkWorkerPlatform() error { return nil }

// setWorkerProcessGroup starts c as the leader of its own process group and
// has its context end the whole group, so a killed worker takes its children
// with it.
func setWorkerProcessGroup(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cancel = func() error {
		err := syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
}
