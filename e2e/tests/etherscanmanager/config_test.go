package etherscanmanagere2e

import (
	"testing"
	"time"

	"github.com/useryege/athena/e2e/internal/e2etest"
)

const (
	defaultEtherscanManagerAddress = "127.0.0.1:8100"
	defaultE2ETimeout              = 90 * time.Second

	envEtherscanManagerAddr = "ATHENA_E2E_ETHERSCAN_MANAGER_ADDR"
	envE2ETimeout           = "ATHENA_E2E_TIMEOUT"
)

type e2eConfig struct {
	addr    string
	timeout time.Duration
}

func loadConfig(t testing.TB) e2eConfig {
	t.Helper()

	return e2eConfig{
		addr:    e2etest.StringFromEnv(envEtherscanManagerAddr, defaultEtherscanManagerAddress),
		timeout: e2etest.DurationFromEnv(t, envE2ETimeout, defaultE2ETimeout),
	}
}
