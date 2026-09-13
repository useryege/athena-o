//go:build linux

package main

import (
	"bytes"
	"encoding/json"
	"github.com/useryege/athena/internal/devruntime"
	"io"
	"reflect"
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

func TestMakeArgumentsAreValidatedAsData(t *testing.T) {
	t.Setenv("SERVICE", "trader-sync;touch /tmp/task9-injection")
	if err := run([]string{"make-build"}, io.Discard); err == nil {
		t.Fatal("shell-like service accepted")
	}
	t.Setenv("SERVICES", "api-server trader-sync")
	t.Setenv("INSTANCE", "")
	if err := run([]string{"make-run-services"}, io.Discard); err == nil {
		t.Fatal("combination without explicit instance accepted")
	}
}
func TestMakeRunServiceDefaultsInstanceAndPassesPathsLiterally(t *testing.T) {
	t.Setenv("SERVICE", "trader-sync")
	t.Setenv("INSTANCE", "")
	t.Setenv("ENV_FILE", "/tmp/a $(touch marker).env")
	args, e := makeArguments("make-run-service")
	if e != nil {
		t.Fatal(e)
	}
	want := []string{"run", "--services", "trader-sync", "--instance", "trader-sync", "--env-file", "/tmp/a $(touch marker).env", "--db-mode", "managed"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("%v", args)
	}
}
