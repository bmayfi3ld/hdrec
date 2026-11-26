package record

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLongPart(t *testing.T) {
	root := t.TempDir()
	dir, e := newSession(root)
	if e != nil {
		t.Fatal(e)
	}
	other, e := newSession(root)
	if e != nil || other == dir {
		t.Fatal(other, e)
	}
	start := time.Now()
	p, e := openPart(dir, 1, start)
	if e != nil {
		t.Fatal(e)
	}
	if _, e := openPart(dir, 1, start); !os.IsExist(e) {
		t.Fatalf("expected collision: %v", e)
	}
	for _, at := range []time.Time{start, start.Add(8 * time.Hour)} {
		if _, e := p.write([]byte("data"), at); e != nil {
			t.Fatal(e)
		}
	}
	path, e := p.finalize()
	if e != nil {
		t.Fatal(e)
	}
	entries, e := os.ReadDir(dir)
	if e != nil || len(entries) != 1 || entries[0].Name() != filepath.Base(path) {
		t.Fatal(entries, e)
	}
	b, e := os.ReadFile(path)
	if e != nil || string(b) != "datadata" {
		t.Fatal(string(b), e)
	}
}

func TestFailedFinalizationKeepsPartial(t *testing.T) {
	dir := t.TempDir()
	p, e := openPart(dir, 1, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	if e := os.Mkdir(filepath.Join(dir, "part-000001.ts"), 0700); e != nil {
		t.Fatal(e)
	}
	path, e := p.finalize()
	if e == nil {
		t.Fatal("rename over directory succeeded")
	}
	if _, e := os.Stat(path); e != nil {
		t.Fatal(e)
	}
	if _, e := p.write([]byte("data"), time.Now()); e == nil {
		t.Fatal("write on closed file succeeded")
	}
}
