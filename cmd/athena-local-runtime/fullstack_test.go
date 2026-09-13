//go:build linux

package main

import (
	"reflect"
	"testing"
)

func TestFullStackMakeArguments(t *testing.T) {
	t.Setenv("INSTANCE", "")
	t.Setenv("DB_MODE", "")
	t.Setenv("ENV_FILE", "literal $(touch impossible).env")
	for action, target := range map[string]string{"make-run-full-stack": "run-full-stack", "make-stop-full-stack": "stop", "make-reset-full-stack": "reset"} {
		got, err := makeArguments(action)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{target, "--instance", "full-stack"}
		if target == "run-full-stack" {
			want = append(want, "--env-file", "literal $(touch impossible).env", "--db-mode", "managed")
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: %v", action, got)
		}
	}
	t.Setenv("INSTANCE", "task10-fixture")
	got, err := makeArguments("make-stop-full-stack")
	if err != nil || !reflect.DeepEqual(got, []string{"stop", "--instance", "task10-fixture"}) {
		t.Fatal(got, err)
	}
	t.Setenv("DB_MODE", "external")
	if _, err := makeArguments("make-run-full-stack"); err == nil {
		t.Fatal("full-stack external accepted")
	}
}
