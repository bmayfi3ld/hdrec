package power

import (
	"context"
	"io"
)

func Inhibit(ctx context.Context) (io.Closer, error) { return inhibit(ctx) }

type closer func() error

func (f closer) Close() error { return f() }
