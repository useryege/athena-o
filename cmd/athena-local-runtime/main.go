//go:build linux

// athena-local-runtime is a foreground local resource owner.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/useryege/athena/internal/devruntime"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func makeArguments(action string) ([]string, error) {
	service := os.Getenv("SERVICE")
	services := os.Getenv("SERVICES")
	instance := os.Getenv("INSTANCE")
	mode := os.Getenv("DB_MODE")
	if mode == "" {
		mode = "managed"
	}
	file := os.Getenv("ENV_FILE")
	if file == "" {
		file = ".env"
	}
	target := ""
	switch action {
	case "make-run-full-stack", "make-stop-full-stack", "make-reset-full-stack":
		if instance == "" {
			instance = "full-stack"
		}
		if action == "make-stop-full-stack" {
			return []string{"stop", "--instance", instance}, nil
		}
		if action == "make-reset-full-stack" {
			return []string{"reset", "--instance", instance}, nil
		}
		if mode != "managed" {
			return nil, errors.New("full stack requires DB_MODE=managed")
		}
		return []string{"run-full-stack", "--instance", instance, "--env-file", file, "--db-mode", mode}, nil
	case "make-build":
		target = "build"
		services = service
		if instance == "" {
			instance = service
		}
	case "make-run-service":
		target = "run"
		services = service
		if instance == "" {
			instance = service
		}
	case "make-run-services":
		target = "run"
	case "make-status":
		target = "status"
	case "make-stop":
		target = "stop"
	case "make-reset":
		target = "reset"
	case "make-seed":
		target = "seed"
		services = service
	default:
		return nil, errors.New("unknown Make runtime action")
	}
	if instance == "" {
		return nil, errors.New("INSTANCE is required for this command")
	}
	if target == "build" || target == "run" || target == "seed" {
		if _, e := devruntime.ResolveServices(strings.Fields(services)); e != nil {
			return nil, e
		}
		return []string{target, "--services", services, "--instance", instance, "--env-file", file, "--db-mode", mode}, nil
	}
	return []string{target, "--instance", instance}, nil
}
func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: athena-local-runtime <build|run|run-full-stack|status|stop|reset|seed> --instance NAME")
	}
	if strings.HasPrefix(args[0], "make-") {
		converted, e := makeArguments(args[0])
		if e != nil {
			return e
		}
		args = converted
	}
	action := args[0]
	switch action {
	case "build", "run", "run-full-stack", "supervise-full-stack", "supervise", "supervise-build", "status", "stop", "reset", "seed":
	default:
		return fmt.Errorf("unknown runtime command %q", action)
	}
	flags := flag.NewFlagSet(action, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	checkout := flags.String("checkout", ".", "checkout directory")
	name := flags.String("instance", "", "instance name")
	services := flags.String("services", "", "space separated explicit service names")
	mode := flags.String("db-mode", "managed", "managed or external")
	file := flags.String("env-file", ".env", "dotenv configuration file")
	if e := flags.Parse(args[1:]); e != nil {
		return e
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if e := devruntime.CheckPlatform(); e != nil {
		return e
	}
	key, e := devruntime.NewInstanceKey(*checkout, *name)
	if e != nil {
		return e
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	switch action {
	case "build", "run", "run-full-stack":
		var specs []devruntime.ServiceSpec
		if action == "run-full-stack" {
			if *mode != "managed" || *services != "" {
				return errors.New("full stack requires managed mode and its explicit service graph")
			}
		} else {
			var err error
			specs, err = devruntime.ResolveServices(strings.Fields(*services))
			if err != nil {
				return err
			}
		}
		target := "supervise"
		if action == "run-full-stack" {
			target = "supervise-full-stack"
		}
		forwarded := append([]string{}, args[1:]...)
		if action == "build" {
			for _, spec := range specs {
				if spec.Name == "ui" {
					return errors.New("build-service builds independent Go binaries")
				}
			}
			target = "supervise-build"
			job, e := devruntime.NewInstanceKey(key.Checkout, "build-"+key.Namespace[:12]+"-"+devruntime.NewRunID()[:8])
			if e != nil {
				return e
			}
			key = job
			forwarded = []string{"--checkout", key.Checkout, "--instance", key.Name, "--services", *services}
		}
		self, e := os.Executable()
		if e != nil {
			return e
		}
		immutable, e := devruntime.ImmutableExecutable(key, self)
		if e != nil {
			return e
		}
		childArgs := append([]string{target}, forwarded...)
		env := os.Environ()
		for i := len(env) - 1; i >= 0; i-- {
			if strings.HasPrefix(env[i], devruntime.RunIDEnv+"=") {
				env = append(env[:i], env[i+1:]...)
			}
		}
		runID := devruntime.NewRunID()
		env = append(env, devruntime.RunIDEnv+"="+runID)
		child := exec.Command(immutable, childArgs...)
		child.Env = env
		child.Stdin = os.Stdin
		child.Stdout = out
		child.Stderr = os.Stderr
		child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		return superviseForeground(ctx, child, runID)
	case "supervise-build":
		return devruntime.Build(ctx, key, strings.Fields(*services))

	case "supervise-full-stack":
		return devruntime.RunFullStack(ctx, devruntime.RunOptions{Key: key, Services: strings.Fields(*services), DBMode: *mode, EnvFile: *file})
	case "supervise":
		return devruntime.Run(ctx, devruntime.RunOptions{Key: key, Services: strings.Fields(*services), DBMode: *mode, EnvFile: *file})
	case "status":
		s, e := devruntime.Status(ctx, key)
		if e != nil {
			return e
		}
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(s)
	case "stop":
		bounded, c := context.WithTimeout(ctx, 2*time.Minute)
		defer c()
		return devruntime.Stop(bounded, key)
	case "reset":
		bounded, c := context.WithTimeout(ctx, 2*time.Minute)
		defer c()
		return devruntime.Reset(bounded, key)
	case "seed":
		names := strings.Fields(*services)
		if len(names) != 1 {
			return errors.New("seed requires one service")
		}
		bounded, c := context.WithTimeout(ctx, 2*time.Minute)
		defer c()
		fixture, err := devruntime.Seed(bounded, key, names[0])
		if err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(fixture)
	}
	return nil
}
func main() {
	if e := run(os.Args[1:], os.Stdout); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}

// This relay stays in the terminal foreground group. Only the supervised child
// receives an independent PGID; a terminal signal is forwarded through a
// verified identity, then the relay waits for the child's normal Stop/reap path.
func superviseForeground(ctx context.Context, child *exec.Cmd, runID string) error {
	expected, e := filepath.EvalSymlinks(child.Path)
	if e != nil {
		return e
	}
	if e := child.Start(); e != nil {
		return e
	}
	done := make(chan error, 1)
	go func() { done <- child.Wait() }()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	var identity devruntime.ProcessIdentity
	for identity.PID == 0 {
		select {
		case e := <-done:
			return e
		case <-deadline.C:
			_ = child.Process.Kill()
			<-done
			return errors.New("cannot establish foreground supervisor identity")
		case <-tick.C:
			p, e := devruntime.ReadProcess(child.Process.Pid)
			if e == nil && p.RunID == runID && p.Exe == expected && p.PGID == p.PID {
				identity = p
			}
		}
	}
	select {
	case e := <-done:
		return e
	case <-ctx.Done():
		e := devruntime.SignalProcess(identity, syscall.SIGTERM)
		if errors.Is(e, os.ErrNotExist) {
			e = nil
		}
		if e != nil {
			return e
		}
		// The service's own bounded Stop protocol decides escalation. The launcher
		// remains alive so Make and the terminal wait for that actual result.
		return <-done
	}
}
