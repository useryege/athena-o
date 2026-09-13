//go:build integration && linux

package main

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestMakeTerminalInterruptWaitsForStop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python3", "../../hack/local-runtime-pty_test.py")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("real PTY stop: %v\n%s", err, output)
	}
	t.Logf("%s", output)
}
