//go:build linux

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMakeExportsRuntimeInputsWithoutEvaluatingThem(t *testing.T) {
	root, e := filepath.Abs("../..")
	if e != nil {
		t.Fatal(e)
	}
	for _, key := range []string{"SERVICE", "SERVICES", "INSTANCE", "DB_MODE", "ENV_FILE", "TRADER_SYNC_IMAGE", "ACCOUNT_STATE_MAINTENANCE", "ACCOUNT_STATE_EXTERNAL_CONSUMERS_STOPPED"} {
		t.Run(key, func(t *testing.T) {
			dir := t.TempDir()
			marker := filepath.Join(dir, "executed")
			value := "$(shell touch " + marker + ")literal $(echo shell) $${literal}"
			overlay := filepath.Join(dir, "probe.mk")
			body := ".PHONY: runtime-raw-probe\nruntime-raw-probe:\n\t@python3 -c 'import os,json; print(json.dumps({k:os.environ.get(k) for k in [\"SERVICE\",\"SERVICES\",\"INSTANCE\",\"DB_MODE\",\"ENV_FILE\",\"TRADER_SYNC_IMAGE\",\"ACCOUNT_STATE_MAINTENANCE\",\"ACCOUNT_STATE_EXTERNAL_CONSUMERS_STOPPED\"]}))'\n"
			if e = os.WriteFile(overlay, []byte(body), 0600); e != nil {
				t.Fatal(e)
			}
			for _, origin := range []string{"command-line", "environment"} {
				cmd := exec.Command("make", "--no-print-directory", "-f", filepath.Join(root, "Makefile"), "-f", overlay, "runtime-raw-probe")
				cmd.Dir = root
				if origin == "command-line" {
					cmd.Args = append(cmd.Args, key+"="+value)
				} else {
					cmd.Env = append(os.Environ(), key+"="+value)
				}
				out, e := cmd.Output()
				if e != nil {
					t.Fatal(e)
				}
				var got map[string]string
				if e = json.Unmarshal(out, &got); e != nil {
					t.Fatalf("%v: %s", e, out)
				}
				if _, e = os.Stat(marker); !os.IsNotExist(e) {
					t.Fatal("Make executed runtime input before Go validation", origin)
				}
				if got[key] != value {
					t.Fatalf("%s raw value changed: %q", origin, got[key])
				}
			}
		})
	}
}

func TestMakeImageTargetPassesRawImageArgument(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "evaluated")
	image := "example:$(shell touch " + marker + ")$$(touch " + marker + ")"
	// The actual Make target assembles its Docker argv. This recorder performs no
	// Docker mutation and proves both Make and shell preserve the exact argument.
	docker := filepath.Join(dir, "docker")
	if err = os.WriteFile(docker, []byte("#!/usr/bin/env python3\nimport json,sys\nprint(json.dumps(sys.argv[1:]))\n"), 0700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("make", "--no-print-directory", "build-service-image", "SERVICE=trader-sync", "TRADER_SYNC_IMAGE="+image)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"))
	output, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var argv []string
	if err = json.Unmarshal(output, &argv); err != nil {
		t.Fatalf("%v: %s", err, output)
	}
	found := false
	for i, arg := range argv {
		if arg == "-t" && i+1 < len(argv) {
			found = argv[i+1] == image
		}
	}
	if !found {
		t.Fatal("image was not passed literally", strings.Join(argv, " | "))
	}
	if _, err = os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("Make image target evaluated data")
	}
}
