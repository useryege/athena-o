package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
)

func main() {
	byteCount := flag.Int("bytes", 32, "number of random bytes to generate")
	format := flag.String("format", "base64", "output format: base64 or hex")
	flag.Parse()

	if *byteCount <= 0 {
		fmt.Fprintln(os.Stderr, "error: -bytes must be greater than 0")
		os.Exit(1)
	}

	secretBytes := make([]byte, *byteCount)
	if _, err := rand.Read(secretBytes); err != nil {
		fmt.Fprintf(os.Stderr, "error generating random secret: %v\n", err)
		os.Exit(1)
	}

	switch *format {
	case "base64":
		fmt.Println(base64.StdEncoding.EncodeToString(secretBytes))
	case "hex":
		fmt.Println(hex.EncodeToString(secretBytes))
	default:
		fmt.Fprintf(os.Stderr, "error: unsupported -format %q; use base64 or hex\n", *format)
		os.Exit(1)
	}
}
