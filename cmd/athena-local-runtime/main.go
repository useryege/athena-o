//go:build linux

// athena-local-runtime is a local resource owner, not a business-service host.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/useryege/athena/internal/devruntime"
)

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: athena-local-runtime <status|stop|reset> --instance NAME [--checkout PATH]")
	}
	action := args[0]
	if action != "status" && action != "stop" && action != "reset" {
		return fmt.Errorf("unknown runtime command %q", action)
	}
	flags := flag.NewFlagSet(action, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	checkout := flags.String("checkout", ".", "checkout directory")
	name := flags.String("instance", "", "instance name")
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
	m := devruntime.NewManager(key)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	switch action {
	case "status":
		s, e := m.Status()
		if e != nil {
			return e
		}
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(s)
	case "stop":
		return m.Stop(ctx)
	case "reset":
		return m.Reset(ctx)
	}
	return nil
}
func main() {
	if e := run(os.Args[1:], os.Stdout); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
