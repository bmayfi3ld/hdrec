package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/bmayfi3ld/hdrec/pkg/record"
)

type blockedWriter struct{ entered, release chan struct{} }

func (w blockedWriter) Write(p []byte) (int, error) {
	select {
	case <-w.entered:
	default:
		close(w.entered)
	}
	<-w.release
	return len(p), nil
}

func TestStalledOutput(t *testing.T) {
	w := blockedWriter{make(chan struct{}), make(chan struct{})}
	report, finish := liveOutput(w, true, "7.1")
	report(record.Event{Kind: "session_started"})
	select {
	case <-w.entered:
	case <-time.After(time.Second):
		t.Fatal("writer not entered")
	}
	progressed := make(chan struct{})
	go func() {
		for i := 0; i < 200; i++ {
			report(record.Event{Kind: "error"})
			report(record.Event{Kind: "metrics", Result: &record.Result{Files: i}})
		}
		close(progressed)
	}()
	select {
	case <-progressed:
	case <-time.After(time.Second):
		close(w.release)
		t.Fatal("capture blocked on rendering")
	}
	close(w.release)
	finish()
}
func TestOutput(t *testing.T) {
	e := record.Event{Kind: "session_stopped", Result: &record.Result{Bytes: 456, Files: 3}}
	for _, mode := range []bool{true, false} {
		var out bytes.Buffer
		report, finish := liveOutput(&out, mode, "7.1")
		report(e)
		finish()
		if mode {
			var got record.Event
			if err := json.Unmarshal(out.Bytes(), &got); err != nil || *got.Result != *e.Result {
				t.Fatal(out.String(), err)
			}
		} else if !strings.Contains(out.String(), "3 files, 456 bytes") || strings.Contains(out.String(), "\x1b") {
			t.Fatal(out.String())
		}
	}
	e.State = "recording"
	e.Rate = 8_000_000
	wide, narrow := statusLine("7.1", e, 100), statusLine("7.1", e, 50)
	if !strings.Contains(wide, "Mb/s") || strings.Contains(narrow, "Mb/s") || len(narrow) >= 50 || !strings.Contains(narrow, "3 files") {
		t.Fatal(wide, narrow)
	}
}
func TestJSONStartupError(t *testing.T) {
	var out, errOut bytes.Buffer
	if Run(context.Background(), []string{"record", "7.1", "--json", "--device", "no-url"}, &out, &errOut) != 1 || errOut.Len() != 0 {
		t.Fatal(errOut.String())
	}
	var got record.Event
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || got.Kind != "error" || got.Code != "device_selection" {
		t.Fatal(out.String(), err)
	}
}
