package commands

import (
	"fmt"
	"net/url"
	"time"

	"github.com/useryege/athena/internal/solanadiscovery/rpcservice"
)

type config struct {
	Address, RPCURL, DSN, InternalToken  string
	Port, Concurrency, RequestsPerSecond int
	StartSlot, RangeSize                 uint64
	RequestTimeout, PollInterval         time.Duration
}

func defaultConfig() config {
	return config{Address: "127.0.0.1", Port: 8112, RPCURL: "https://api.mainnet-beta.solana.com", RangeSize: 4, Concurrency: 1, RequestsPerSecond: 1, RequestTimeout: 15 * time.Second, PollInterval: 2 * time.Second}
}

func (c config) validate() error {
	if _, err := rpcservice.NormalizeInternalAuthToken(c.InternalToken); err != nil {
		return err
	}
	u, err := url.Parse(c.RPCURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" {
		return fmt.Errorf("Solana RPC must be an HTTP(S) URL with a host")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("Solana listen port must be 1..65535")
	}
	if c.RangeSize < 1 || c.RangeSize > 32 {
		return fmt.Errorf("Solana scan range must be 1..32 slots")
	}
	if c.Concurrency < 1 || c.Concurrency > 32 {
		return fmt.Errorf("Solana concurrency must be 1..32")
	}
	if c.RequestsPerSecond < 1 || c.RequestsPerSecond > 1000 {
		return fmt.Errorf("Solana requests per second must be 1..1000")
	}
	if c.RequestTimeout <= 0 || c.RequestTimeout > time.Minute {
		return fmt.Errorf("Solana request timeout must be positive and at most 1 minute")
	}
	if c.PollInterval <= 0 {
		return fmt.Errorf("Solana poll interval must be positive")
	}
	return nil
}
