//go:build linux || darwin

package power

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"time"
)

func inhibit(ctx context.Context) (io.Closer, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("caffeinate", "-i", "sh", "-c", "echo ready; cat >/dev/null")
	} else {
		cmd = exec.Command("systemd-inhibit", "--what=idle", "--mode=block", "--who=hdrec", "--why=Recording television", "sh", "-c", "echo ready; cat >/dev/null")
	}
	input, e := cmd.StdinPipe()
	if e != nil {
		return nil, e
	}
	output, e := cmd.StdoutPipe()
	if e != nil {
		input.Close()
		return nil, e
	}
	if e = cmd.Start(); e != nil {
		input.Close()
		output.Close()
		return nil, e
	}
	ready := make(chan error, 1)
	go func() {
		var b [6]byte
		_, e := io.ReadFull(output, b[:])
		if e == nil && string(b[:]) != "ready\n" {
			e = fmt.Errorf("unexpected inhibitor response")
		}
		ready <- e
	}()
	close := func() error { input.Close(); return cmd.Wait() }
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	select {
	case e = <-ready:
	case <-timer.C:
		e = fmt.Errorf("inhibitor startup timed out")
	case <-ctx.Done():
		e = ctx.Err()
	}
	if e != nil {
		input.Close()
		cmd.Process.Kill()
		cmd.Wait()
		return nil, fmt.Errorf("sleep inhibitor: %w", e)
	}
	return closer(close), nil
}
