//go:build !linux

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "athena-local-runtime requires Linux/WSL with pidfd process identity verification; no resources were allocated")
	os.Exit(1)
}
