//go:build !linux

package gpioinput

import (
	"context"
	"fmt"
	"runtime"

	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

func (r *Reader) Read(ctx context.Context, gpioChannel string) (inputs.DigitalState, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	_, canonical, err := ParseBCMChannel(gpioChannel)
	if err != nil {
		return "", err
	}
	return "", fmt.Errorf("gpio read for %s is unsupported on %s", canonical, runtime.GOOS)
}
