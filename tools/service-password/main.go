package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"math/big"
	"os"
)

const defaultAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func main() {
	length := flag.Int("length", 32, "password length")
	flag.Parse()

	if *length <= 0 {
		fmt.Fprintln(os.Stderr, "error: -length must be greater than 0")
		os.Exit(1)
	}

	password, err := randomString(*length, defaultAlphabet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating password: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(password)
}

func randomString(length int, alphabet string) (string, error) {
	out := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = alphabet[n.Int64()]
	}
	return string(out), nil
}
