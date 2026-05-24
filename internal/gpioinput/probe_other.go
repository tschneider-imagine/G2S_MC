//go:build !linux

package gpioinput

import (
	"context"
	"fmt"
)

func probeGPIOEnvironment(_ context.Context, report *ProbeReport) {
	report.Driver = "unsupported"
	report.PullUpRequestAttempted = false
	report.PullUpRequestSupported = false
	report.Errors = append(report.Errors, fmt.Sprintf("gpio probe requires Linux GPIO character devices; current platform is %s", report.GOOS))
}
