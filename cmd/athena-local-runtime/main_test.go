//go:build linux

package main

import (
	"bytes"
	"encoding/json"
	"github.com/useryege/athena/internal/devruntime"
	"testing"
)

func TestCLIStatusAndUnknownCommand(t *testing.T) {
	k, e := devruntime.NewInstanceKey(t.TempDir(), "test")
	if e != nil {
		t.Fatal(e)
	}
	s := devruntime.State{Version: 1, Key: k, RunID: "test-run", Phase: "stopped", DBMode: "managed"}
	if e = devruntime.SaveState(k.StatePath(), s); e != nil {
		t.Fatal(e)
	}
	var out bytes.Buffer
	if e = run([]string{"status", "--checkout", k.Checkout, "--instance", "test"}, &out); e != nil {
		t.Fatal(e)
	}
	var got devruntime.State
	if e = json.Unmarshal(out.Bytes(), &got); e != nil || got.RunID != "test-run" {
		t.Fatalf("status %s %v", out.String(), e)
	}
	if e = run([]string{"destroy-all", "--checkout", k.Checkout, "--instance", "test"}, &out); e == nil {
		t.Fatal("accepted unknown command")
	}
}
