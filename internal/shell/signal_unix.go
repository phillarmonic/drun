//go:build !windows

package shell

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// forwardSignals relays interrupt/terminate signals to the running child
// process so that Ctrl+C cleanly stops the currently executing task.
func forwardSignals(cmd *exec.Cmd) func() {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case sig, ok := <-signalCh:
				if !ok {
					return
				}
				if cmd.Process != nil {
					_ = cmd.Process.Signal(sig)
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
