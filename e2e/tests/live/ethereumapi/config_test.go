package ethereumapilivee2e

import (
	"testing"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/e2e/internal/e2etest"
)

const (
	defaultEthereumAPIAddress = "127.0.0.1:8100"
	defaultE2ETimeout         = 90 * time.Second
	defaultQueryAddress       = "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"

	envEthereumAPIAddr         = "ATHENA_E2E_ETHEREUM_API_ADDR"
	envEthereumAPIQueryAddress = "ATHENA_E2E_ETHEREUM_API_ADDRESS"
	envE2ETimeout              = "ATHENA_E2E_TIMEOUT"
	envE2ELive                 = "E2E_LIVE"
)

type e2eConfig struct {
	addr         string
	timeout      time.Duration
	queryAddress string
}

func loadConfig(t testing.TB) e2eConfig {
	t.Helper()

	queryAddress := e2etest.StringFromEnv(envEthereumAPIQueryAddress, defaultQueryAddress)
	if !ethcommon.IsHexAddress(queryAddress) {
		t.Fatalf("%s must be a valid EVM address, got %q", envEthereumAPIQueryAddress, queryAddress)
	}

	return e2eConfig{
		addr:         e2etest.StringFromEnv(envEthereumAPIAddr, defaultEthereumAPIAddress),
		timeout:      e2etest.DurationFromEnv(t, envE2ETimeout, defaultE2ETimeout),
		queryAddress: ethcommon.HexToAddress(queryAddress).Hex(),
	}
}
