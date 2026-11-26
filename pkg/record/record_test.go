package record

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bmayfi3ld/hdrec/pkg/hdhomerun"
)

type sourceFunc func(context.Context, string) (io.ReadCloser, error)

func (f sourceFunc) Open(c context.Context, s string) (io.ReadCloser, error) { return f(c, s) }

type blocking struct {
	ctx   context.Context
	first string
}

func (b *blocking) Read(p []byte) (int, error) {
	if b.first != "" {
		n := copy(p, b.first)
		b.first = ""
		return n, nil
	}
	<-b.ctx.Done()
	return 0, b.ctx.Err()
}
func (b *blocking) Close() error { return nil }

func TestRecovery(t *testing.T) {
	for _, stall := range []bool{false, true} {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		opens := 0
		src := sourceFunc(func(c context.Context, _ string) (io.ReadCloser, error) {
			opens++
			switch opens {
			case 1:
				return nil, hdhomerun.ErrBusy
			case 2:
				if !stall {
					return io.NopCloser(strings.NewReader("first")), nil
				}
			}
			return &blocking{c, "data"}, nil
		})
		r, e := run(ctx, src, Request{OutputDir: t.TempDir()}, func(e Event) {
			if e.Kind == "reconnected" {
				cancel()
			}
		}, 10*time.Millisecond, time.Millisecond)
		cancel()
		if e != nil || r.Reconnects != 1 || !r.HadGaps || opens != 3 {
			t.Fatal(r, e, opens)
		}
		paths, e := filepath.Glob(filepath.Join(r.SessionDir, "*.ts"))
		if e != nil || len(paths) != 2 {
			t.Fatal(paths, e)
		}
	}
}
func TestCancelBackoff(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	opens := 0
	src := sourceFunc(func(context.Context, string) (io.ReadCloser, error) { opens++; return nil, hdhomerun.ErrBusy })
	r, e := run(ctx, src, Request{OutputDir: t.TempDir()}, func(e Event) {
		if e.Kind == "retry_scheduled" {
			cancel()
		}
	}, time.Second, time.Second)
	if e != nil || opens != 1 || r.Bytes != 0 {
		t.Fatal(r, e, opens)
	}
}
