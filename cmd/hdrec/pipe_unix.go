//go:build unix

package main

import (
	"os/signal"
	"syscall"
)

// A vanished log consumer must not terminate an active recording.
func init() { signal.Ignore(syscall.SIGPIPE) }
