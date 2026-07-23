package etherscanapilivee2e

import (
	"testing"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/e2e/internal/e2etest"
)

const (
	defaultE2ETimeout   = 90 * time.Second
	defaultQueryAddress = "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"

	envEtherscanAPIKey                 = "ATHENA_E2E_ETHERSCAN_API_KEY"
	envEtherscanAPIKeys                = "ATHENA_E2E_ETHERSCAN_API_KEYS"
	envEtherscanMultiKeyProbe          = "ATHENA_E2E_ETHERSCAN_MULTI_KEY_PROBE"
	envEtherscanMultiKeyStaggeredProbe = "ATHENA_E2E_ETHERSCAN_MULTI_KEY_STAGGERED_PROBE"
	envEtherscanQueryAddress           = "ATHENA_E2E_ETHERSCAN_QUERY_ADDRESS"
	envEtherscanRateLimitProbe         = "ATHENA_E2E_ETHERSCAN_RATE_LIMIT_PROBE"
	envE2ETimeout                      = "ATHENA_E2E_TIMEOUT"
	envE2ELive                         = "E2E_LIVE"
)

type e2eConfig struct {
	timeout      time.Duration
	queryAddress string
}

func loadConfig(t testing.TB) e2eConfig {
	t.Helper()

	queryAddress := e2etest.StringFromEnv(envEtherscanQueryAddress, defaultQueryAddress)
	if !ethcommon.IsHexAddress(queryAddress) {
		t.Fatalf("%s must be a valid EVM address, got %q", envEtherscanQueryAddress, queryAddress)
	}

	return e2eConfig{
		timeout:      e2etest.DurationFromEnv(t, envE2ETimeout, defaultE2ETimeout),
		queryAddress: ethcommon.HexToAddress(queryAddress).Hex(),
	}
}
