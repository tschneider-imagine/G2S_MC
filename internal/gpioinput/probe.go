package gpioinput

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"
)

type ProbeReport struct {
	CheckedAt              time.Time `json:"checked_at"`
	GOOS                   string    `json:"goos"`
	GOARCH                 string    `json:"goarch"`
	Driver                 string    `json:"driver"`
	ChipPath               string    `json:"chip_path"`
	ChipName               string    `json:"chip_name,omitempty"`
	ChipLabel              string    `json:"chip_label,omitempty"`
	LineCount              int       `json:"line_count,omitempty"`
	DefaultChannels        []string  `json:"default_channels"`
	PullUpRequestAttempted bool      `json:"pull_up_request_attempted"`
	PullUpRequestSupported bool      `json:"pull_up_request_supported"`
	Warnings               []string  `json:"warnings,omitempty"`
	Errors                 []string  `json:"errors,omitempty"`
}

func (r ProbeReport) String() string {
	lines := []string{
		fmt.Sprintf("checked_at=%s", r.CheckedAt.Format(time.RFC3339)),
		fmt.Sprintf("platform=%s/%s", r.GOOS, r.GOARCH),
		fmt.Sprintf("driver=%s", r.Driver),
		fmt.Sprintf("chip_path=%s", r.ChipPath),
		fmt.Sprintf("default_channels=%s", strings.Join(r.DefaultChannels, ",")),
		fmt.Sprintf("pull_up_request_attempted=%t", r.PullUpRequestAttempted),
		fmt.Sprintf("pull_up_request_supported=%t", r.PullUpRequestSupported),
	}
	if r.ChipName != "" {
		lines = append(lines, fmt.Sprintf("chip_name=%s", r.ChipName))
	}
	if r.ChipLabel != "" {
		lines = append(lines, fmt.Sprintf("chip_label=%s", r.ChipLabel))
	}
	if r.LineCount > 0 {
		lines = append(lines, fmt.Sprintf("chip_lines=%d", r.LineCount))
	}
	for _, warning := range r.Warnings {
		lines = append(lines, "warning="+warning)
	}
	for _, err := range r.Errors {
		lines = append(lines, "error="+err)
	}
	return strings.Join(lines, "\n")
}

func ProbeGPIOEnvironment(ctx context.Context) ProbeReport {
	report := ProbeReport{
		CheckedAt:       time.Now().UTC(),
		GOOS:            runtime.GOOS,
		GOARCH:          runtime.GOARCH,
		ChipPath:        DefaultChipPath,
		DefaultChannels: DefaultChannels(),
	}
	probeGPIOEnvironment(ctx, &report)
	return report
}
