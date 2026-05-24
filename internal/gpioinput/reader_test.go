package gpioinput

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

func TestDefaultChannels(t *testing.T) {
	channels := DefaultChannels()
	if len(channels) != 4 {
		t.Fatalf("default channel count = %d, want 4", len(channels))
	}
	want := []string{"GPIO16", "GPIO20", "GPIO21", "GPIO26"}
	for i := range want {
		if channels[i] != want[i] {
			t.Fatalf("default channel[%d] = %q, want %q", i, channels[i], want[i])
		}
	}
}

func TestParseChannelsCSVExplicit(t *testing.T) {
	channels, err := ParseChannelsCSV("GPIO16, GPIO20,GPIO21,GPIO26")
	if err != nil {
		t.Fatalf("parse channels: %v", err)
	}
	want := []string{"GPIO16", "GPIO20", "GPIO21", "GPIO26"}
	if len(channels) != len(want) {
		t.Fatalf("channel count = %d, want %d", len(channels), len(want))
	}
	for i := range want {
		if channels[i] != want[i] {
			t.Fatalf("channel[%d] = %q, want %q", i, channels[i], want[i])
		}
	}
}

func TestMapRawValueToDigitalState(t *testing.T) {
	low, err := mapRawValueToDigitalState(0)
	if err != nil {
		t.Fatalf("map 0: %v", err)
	}
	high, err := mapRawValueToDigitalState(1)
	if err != nil {
		t.Fatalf("map 1: %v", err)
	}
	if low != inputs.InputStateLow {
		t.Fatalf("map 0 = %q, want LOW", low)
	}
	if high != inputs.InputStateHigh {
		t.Fatalf("map 1 = %q, want HIGH", high)
	}
	if _, err := mapRawValueToDigitalState(2); err == nil {
		t.Fatal("expected error for raw value 2")
	}
}

func TestProbeReportString(t *testing.T) {
	report := ProbeReport{
		CheckedAt:              time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		GOOS:                   "linux",
		GOARCH:                 "arm64",
		Driver:                 "linux-gpio-chardev-v2",
		ChipPath:               "/dev/gpiochip0",
		DefaultChannels:        []string{"GPIO16", "GPIO20", "GPIO21", "GPIO26"},
		PullUpRequestAttempted: true,
		PullUpRequestSupported: true,
		Warnings:               []string{"sample warning"},
	}
	text := report.String()
	for _, expected := range []string{
		"platform=linux/arm64",
		"chip_path=/dev/gpiochip0",
		"default_channels=GPIO16,GPIO20,GPIO21,GPIO26",
		"pull_up_request_supported=true",
		"warning=sample warning",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("probe string missing %q:\n%s", expected, text)
		}
	}
}

func TestReaderHardwareOptional(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("hardware GPIO test requires linux")
	}
	if strings.TrimSpace(os.Getenv("G2S_GPIO_HARDWARE_TEST")) != "1" {
		t.Skip("set G2S_GPIO_HARDWARE_TEST=1 to enable hardware GPIO test")
	}

	reader := NewReader()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	for _, channel := range DefaultChannels() {
		state, err := reader.Read(ctx, channel)
		if err != nil {
			t.Fatalf("read %s: %v", channel, err)
		}
		if state != inputs.InputStateHigh && state != inputs.InputStateLow {
			t.Fatalf("read %s returned invalid state %q", channel, state)
		}
	}
}
