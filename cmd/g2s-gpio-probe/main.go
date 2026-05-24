package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/gpioinput"
)

func main() {
	channelsFlag := flag.String("channels", "", "comma-separated BCM GPIO channels (example: GPIO16,GPIO20,GPIO21,GPIO26)")
	flag.Parse()

	channels, err := parseChannelsFlag(*channelsFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid -channels value: %v\n", err)
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	report := gpioinput.ProbeGPIOEnvironment(ctx)
	fmt.Println("=== GPIO Environment Probe ===")
	fmt.Println(report.String())
	fmt.Println()

	reader := gpioinput.NewReader()
	fmt.Println("=== GPIO Input Reads ===")
	hadReadError := false
	for _, channel := range channels {
		state, err := reader.Read(ctx, channel)
		if err != nil {
			hadReadError = true
			fmt.Printf("%s ERROR %v\n", channel, err)
			continue
		}
		fmt.Printf("%s %s\n", channel, state)
	}

	if len(report.Errors) > 0 || hadReadError {
		os.Exit(1)
	}
}

func parseChannelsFlag(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return gpioinput.DefaultChannels(), nil
	}
	return gpioinput.ParseChannelsCSV(raw)
}
