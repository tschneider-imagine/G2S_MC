package gpioinput

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

const (
	DefaultChipPath = "/dev/gpiochip0"
)

var defaultBCMChannels = []string{"GPIO16", "GPIO20", "GPIO21", "GPIO26"}

// DigitalReader reads a single GPIO input channel and returns HIGH or LOW.
type DigitalReader interface {
	Read(ctx context.Context, gpioChannel string) (inputs.DigitalState, error)
}

// Reader reads BCM GPIO channels using the platform implementation.
type Reader struct {
	ChipPath string
	Consumer string
}

func NewReader() *Reader {
	return &Reader{
		ChipPath: DefaultChipPath,
		Consumer: "g2s_gpio_probe",
	}
}

func DefaultChannels() []string {
	channels := make([]string, len(defaultBCMChannels))
	copy(channels, defaultBCMChannels)
	return channels
}

func ParseChannelsCSV(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		channel := strings.TrimSpace(part)
		if channel == "" {
			continue
		}
		index, canonical, err := ParseBCMChannel(channel)
		if err != nil {
			return nil, err
		}
		if index < 0 {
			return nil, fmt.Errorf("invalid channel %q", channel)
		}
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		result = append(result, canonical)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("at least one GPIO channel is required")
	}
	return result, nil
}

func ParseBCMChannel(channel string) (int, string, error) {
	trimmed := strings.TrimSpace(channel)
	if trimmed == "" {
		return 0, "", fmt.Errorf("gpio channel is required")
	}

	value := trimmed
	if len(trimmed) >= 4 && strings.EqualFold(trimmed[:4], "GPIO") {
		value = strings.TrimSpace(trimmed[4:])
	}
	if value == "" {
		return 0, "", fmt.Errorf("gpio channel %q is missing BCM number", channel)
	}

	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, "", fmt.Errorf("gpio channel %q must use BCM format like GPIO16", channel)
	}
	if n < 0 {
		return 0, "", fmt.Errorf("gpio channel %q must be >= 0", channel)
	}
	return n, fmt.Sprintf("GPIO%d", n), nil
}

func mapRawValueToDigitalState(raw uint64) (inputs.DigitalState, error) {
	switch raw {
	case 0:
		return inputs.InputStateLow, nil
	case 1:
		return inputs.InputStateHigh, nil
	default:
		return "", fmt.Errorf("unexpected raw GPIO value %d", raw)
	}
}
