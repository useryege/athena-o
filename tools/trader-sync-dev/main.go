//go:build linux

// Development fixtures are a local-only operator command, never a service mode.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/useryege/athena/internal/devruntime"
	"os"
	"time"
)

func main() {
	name := flag.String("instance", "", "managed instance name")
	checkout := flag.String("checkout", ".", "checkout directory")
	flag.Parse()
	key, err := devruntime.NewInstanceKey(*checkout, *name)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		var fixture devruntime.DevelopmentFixture
		fixture, err = devruntime.Seed(ctx, key, "trader-sync")
		if err == nil {
			err = json.NewEncoder(os.Stdout).Encode(fixture)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
