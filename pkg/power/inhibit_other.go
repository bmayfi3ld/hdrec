//go:build !linux && !darwin && !windows

package power

import (
	"context"
	"errors"
	"io"
)

func inhibit(context.Context) (io.Closer, error) {
	return nil, errors.New("idle sleep inhibition not available on this platform")
}
