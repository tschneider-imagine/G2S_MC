//go:build linux

package gpioinput

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
)

func probeGPIOEnvironment(ctx context.Context, report *ProbeReport) {
	select {
	case <-ctx.Done():
		report.Errors = append(report.Errors, ctx.Err().Error())
		return
	default:
	}

	report.Driver = "linux-gpio-chardev-v2"
	report.PullUpRequestAttempted = true

	if _, err := os.Stat(report.ChipPath); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("gpio chip %s not available: %v", report.ChipPath, err))
		report.PullUpRequestSupported = false
		return
	}

	info, err := readChipInfo(report.ChipPath)
	if err != nil {
		report.Errors = append(report.Errors, err.Error())
		report.PullUpRequestSupported = false
		return
	}
	report.ChipName = parseCString(info.Name[:])
	report.ChipLabel = parseCString(info.Label[:])
	report.LineCount = int(info.Lines)

	probeChannel := report.DefaultChannels[0]
	if err := probePullUpSupport(report.ChipPath, probeChannel); err != nil {
		report.PullUpRequestSupported = false
		if errors.Is(err, syscall.EINVAL) || isPullUpUnsupportedError(err) {
			report.Errors = append(report.Errors, fmt.Sprintf("pull-up bias request is unsupported on %s (%s): %v", report.ChipPath, probeChannel, err))
			return
		}
		report.Warnings = append(report.Warnings, fmt.Sprintf("pull-up probe on %s failed (%s): %v", report.ChipPath, probeChannel, err))
		return
	}
	report.PullUpRequestSupported = true
}

func isPullUpUnsupportedError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "pull-up bias request unsupported")
}
