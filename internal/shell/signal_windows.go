//go:build windows

package shell

import (
	"os"
	"os/exec"
	"os/signal"
	"strconv"
)

// forwardSignals stops the running child process when the user presses Ctrl+C.
//
// Windows does not support delivering os.Interrupt/SIGTERM to another process
// via (*os.Process).Signal — the call returns "not supported by windows" and
// does nothing. Because drun traps the interrupt for the duration of a task,
// simply forwarding the signal leaves long-running child processes (and their
// grandchildren, e.g. a docs server) alive with no way to stop them. Instead we
// terminate the whole child process tree with taskkill so Ctrl+C works.
func forwardSignals(cmd *exec.Cmd) func() {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case _, ok := <-signalCh:
				if !ok {
					return
				}
				if cmd.Process != nil {
					terminateProcessTree(cmd.Process.Pid)
				}
			case <-done:
				return
			}
		}
	}()

	return func() {
		close(done)
		signal.Stop(signalCh)
		close(signalCh)
	}
}

// terminateProcessTree forcefully kills the process identified by pid together
// with any child processes it spawned.
func terminateProcessTree(pid int) {
	// #nosec G204 -- pid is an integer owned by drun, not user-controlled input.
	kill := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
	_ = kill.Run()
}
