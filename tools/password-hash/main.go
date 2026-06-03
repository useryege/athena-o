package main

import (
	"flag"
	"fmt"
	"os"

	"golang.org/x/term"

	passwordutil "github.com/useryege/athena/util/password"
)

func main() {
	passwordFlag := flag.String("password", "", "plaintext password (non-interactive; may appear in shell history)")
	flag.Parse()

	password := *passwordFlag
	if password == "" {
		fmt.Fprint(os.Stderr, "Password: ")
		passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading password: %v\n", err)
			os.Exit(1)
		}
		password = string(passwordBytes)
	}

	hash, err := passwordutil.HashPassword(password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error hashing password: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(hash)
}
