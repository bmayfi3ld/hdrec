package power

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"syscall"
)

func inhibit(ctx context.Context) (io.Closer, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("SetThreadExecutionState")
	if e := proc.Find(); e != nil {
		return nil, e
	}
	ready := make(chan error, 1)
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		if r, _, e := proc.Call(0x80000001); r == 0 {
			ready <- fmt.Errorf("SetThreadExecutionState: %w", e)
			return
		}
		ready <- nil
		<-release
		if r, _, e := proc.Call(0x80000000); r == 0 {
			done <- e
		} else {
			done <- nil
		}
	}()
	if e := <-ready; e != nil {
		return nil, e
	}
	return closer(func() error { close(release); return <-done }), nil
}
