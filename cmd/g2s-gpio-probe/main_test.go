package main

import "testing"

func TestParseChannelsFlagDefault(t *testing.T) {
	channels, err := parseChannelsFlag("")
	if err != nil {
		t.Fatalf("parse default channels: %v", err)
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

func TestParseChannelsFlagExplicit(t *testing.T) {
	channels, err := parseChannelsFlag("GPIO16,GPIO20,GPIO21,GPIO26")
	if err != nil {
		t.Fatalf("parse explicit channels: %v", err)
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
