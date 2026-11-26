package record

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmayfi3ld/hdrec/pkg/hdhomerun"
)

func TestOutcome(t *testing.T) {
	for _, tc := range []struct {
		r    Result
		err  error
		want int
	}{
		{Result{}, nil, 1},
		{Result{Bytes: 1}, nil, 0},
		{Result{Bytes: 1, HadGaps: true}, nil, 2},
		{Result{Bytes: 1}, errors.New("fatal"), 1},
	} {
		if got := tc.r.ExitCode(tc.err); got != tc.want {
			t.Fatal(got, tc.want)
		}
	}
}

func TestSessionLog(t *testing.T) {
	for _, fatal := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		src := sourceFunc(func(c context.Context, _ string) (io.ReadCloser, error) {
			if fatal {
				return nil, hdhomerun.ErrPermanent
			}
			return &blocking{c, "data"}, nil
		})
		r, e := Run(ctx, src, Request{OutputDir: t.TempDir()}, func(e Event) {
			if e.Kind == "file_opened" {
				cancel()
			}
		})
		cancel()
		want := 0
		if fatal {
			want = 1
		}
		if r.ExitCode(e) != want {
			t.Fatal(r, e)
		}
		f, e := os.Open(filepath.Join(r.SessionDir, "session.ndjson"))
		if e != nil {
			t.Fatal(e)
		}
		dec := json.NewDecoder(f)
		var last Event
		for {
			var event Event
			if e := dec.Decode(&event); e == io.EOF {
				break
			} else if e != nil {
				t.Fatal(e)
			}
			last = event
		}
		f.Close()
		if last.Kind != "session_stopped" || last.Result == nil || *last.Result != r {
			t.Fatal(last, r)
		}
	}
}
