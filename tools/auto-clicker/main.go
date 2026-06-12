package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	interval := flag.Duration("interval", 50*time.Millisecond, "left click interval")
	flag.Parse()

	if *interval <= 0 {
		fmt.Fprintln(os.Stderr, "error: -interval must be greater than 0")
		os.Exit(1)
	}

	if err := run(*interval); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
