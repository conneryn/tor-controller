package onionbalancedaemon

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type OnionBalance struct {
	cmd *exec.Cmd
}

func (t *OnionBalance) Start(ctx context.Context) {
	go func() {
		for {
			fmt.Println("starting onionbalance...")
			t.cmd = exec.CommandContext(ctx,
				"onionbalance",
				"--config", "/run/onionbalance/config.yaml",
				// "--verbosity", "debug",
				"--ip", "127.0.0.1",
				"--port", "9051",
				"--hs-version", "v3",
			)
			t.cmd.Stdout = os.Stdout
			t.cmd.Stderr = os.Stderr

			err := t.cmd.Start()
			if err != nil {
				fmt.Print(err)
			}

			t.cmd.Wait()

			// Check if ctx is done (shutting down)
			select {
			case <-ctx.Done():
				fmt.Println("terminating onionbalance...")
				return
			default:
				// sleep, then restart
				time.Sleep(time.Second * 3)
			}
		}
	}()
}

func (t *OnionBalance) Reload() {
	fmt.Println("reloading onionbalance...")

	if t.cmd != nil && (t.cmd.ProcessState == nil || !t.cmd.ProcessState.Exited()) {
		fmt.Println("stopping existing onionbalance...")
		t.cmd.Process.Signal(syscall.SIGHUP)
	} else {
		fmt.Println("onionbalance is not currently running...")
	}
}
