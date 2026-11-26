package cli

import (
	"context"
	"errors"
	"io"
	"testing"
)

type testCloser struct{ closed bool }

func (c *testCloser) Close() error { c.closed = true; return nil }
func TestPower(t *testing.T) {
	warnings := 0
	warn := func(string) { warnings++ }
	holdPower(context.Background(), func(context.Context) (io.Closer, error) { return nil, errors.New("unavailable") }, warn)()
	c := new(testCloser)
	holdPower(context.Background(), func(context.Context) (io.Closer, error) { return c, nil }, warn)()
	if warnings != 1 || !c.closed {
		t.Fatal(warnings, c.closed)
	}
}
