package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bmayfi3ld/hdrec/pkg/record"
	"golang.org/x/term"
)

func liveOutput(w io.Writer, jsonMode bool, channel string) (func(record.Event), func()) {
	width := func() int { return 0 }
	if f, ok := w.(*os.File); ok && !jsonMode && term.IsTerminal(int(f.Fd())) && os.Getenv("TERM") != "dumb" {
		width = func() int { n, _, _ := term.GetSize(int(f.Fd())); return n }
	}
	queue := make(chan record.Event, 32)
	done := make(chan struct{})
	var latest atomic.Pointer[record.Event]
	go func() { defer close(done); renderLive(w, jsonMode, channel, width, queue, &latest) }()
	return func(e record.Event) {
			if e.Time.IsZero() {
				e.Time = time.Now()
			}
			if e.Kind == "metrics" {
				latest.Store(&e)
				return
			}
			select {
			case queue <- e:
			default:
			}
		}, func() {
			close(queue)
			timer := time.NewTimer(500 * time.Millisecond)
			defer timer.Stop()
			select {
			case <-done:
			case <-timer.C:
			}
		}
}

func renderLive(w io.Writer, jsonMode bool, channel string, width func() int, queue <-chan record.Event, latest *atomic.Pointer[record.Event]) {
	interval := time.Second
	if jsonMode {
		interval = 10 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	encoder := json.NewEncoder(w)
	active := false
	started := time.Now()
	clear := func() {
		if active {
			fmt.Fprint(w, "\r\x1b[2K")
			active = false
		}
	}
	snapshot := func() record.Event {
		now := time.Now()
		e := record.Event{Kind: "metrics", Time: now, State: "acquiring", Result: &record.Result{Elapsed: now.Sub(started).Seconds()}}
		if last := latest.Load(); last != nil {
			e = *last
			r := *last.Result
			e.Result = &r
			e.Time = now
			r.Elapsed += now.Sub(last.Time).Seconds()
			r.Unavailable = max(0, r.Elapsed-r.Receiving)
			if now.Sub(last.Time) > 2*time.Second {
				e.Rate = 0
			}
		}
		return e
	}
	draw := func() {
		if n := width(); n > 1 {
			e := snapshot()
			fmt.Fprint(w, statusLine(channel, e, n))
			active = true
		}
	}
	defer clear()
	for {
		select {
		case e, ok := <-queue:
			if !ok {
				return
			}
			if jsonMode {
				if encoder.Encode(e) != nil {
					return
				}
				continue
			}
			clear()
			fmt.Fprintf(w, "%s  %s\n", e.Time.Format("15:04:05"), safeLine(eventLine(e)))
			if e.Kind == "finalizing" {
				status := snapshot()
				status.State = "finalizing"
				latest.Store(&status)
			}
			if e.Kind != "session_stopped" {
				draw()
			}
		case <-ticker.C:
			if jsonMode {
				if encoder.Encode(snapshot()) != nil {
					return
				}
			} else {
				clear()
				draw()
			}
		}
	}
}

func statusLine(channel string, e record.Event, width int) string {
	r := e.Result
	state := strings.ToUpper(strings.ReplaceAll(e.State, "_", " "))
	elapsed := time.Duration(r.Elapsed * float64(time.Second)).Truncate(time.Second)
	fields := []string{state + "  " + safeLine(channel), elapsed.String() + " elapsed", fmt.Sprintf("%d files", r.Files), fmt.Sprintf("%.2f GiB", float64(r.Bytes)/(1<<30)), fmt.Sprintf("%.1f Mb/s", float64(e.Rate)/1e6), fmt.Sprintf("%d reconnects", r.Reconnects)}
	for len(fields) > 1 && len(strings.Join(fields, " | ")) >= width {
		fields = fields[:len(fields)-1]
	}
	line := strings.Join(fields, " | ")
	if len(line) >= width {
		line = line[:max(0, width-1)]
	}
	return line
}

func safeLine(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || (r >= 128 && r <= 159) {
			return ' '
		}
		return r
	}, s)
}

func eventLine(e record.Event) string {
	switch e.Kind {
	case "session_started":
		return fmt.Sprintf("Started channel %s on %s. Output: %s. Ctrl-C to stop.", e.Channel, e.Device, e.Path)
	case "file_opened":
		return "Opened " + filepath.Base(e.Path)
	case "file_closed":
		return "Saved " + filepath.Base(e.Path)
	case "reconnected":
		return fmt.Sprintf("Reconnected after %.1fs", e.Gap)
	case "retry_scheduled":
		if e.Code == "tuner_busy" {
			return fmt.Sprintf("No tuner available; retrying in %.1fs. Free a tuner or press Ctrl-C.", e.Delay)
		}
		return fmt.Sprintf("Retrying in %.1fs", e.Delay)
	case "finalizing":
		return "Finalizing recording"
	case "session_stopped":
		r := e.Result
		return fmt.Sprintf("Stopped (%s): %.1fs elapsed, %.1fs receiving, %.1fs unavailable (%.1fs initial wait), %d files, %d bytes, %d reconnects, known gaps: %t. Output: %s", r.Reason, r.Elapsed, r.Receiving, r.Unavailable, r.InitialWait, r.Files, r.Bytes, r.Reconnects, r.HadGaps, r.SessionDir)
	default:
		return e.Message
	}
}
