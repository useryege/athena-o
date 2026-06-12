//go:build !windows

package main

import (
	"errors"
	"time"
)

func run(_ time.Duration) error {
	return errors.New("auto-clicker only supports Windows desktop")
}
