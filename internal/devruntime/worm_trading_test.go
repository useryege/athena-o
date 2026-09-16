package devruntime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWormTradingIsAnIndependentService(t *testing.T) {
	specs, err := ResolveServices([]string{"worm-trading"})
	require.NoError(t, err)
	require.Len(t, specs, 1)
	require.Equal(t, "./cmd/athena-worm-trading", specs[0].BuildPackage)
	require.Equal(t, []string{"postgres"}, specs[0].Infrastructure)
	require.Contains(t, specs[0].EnvironmentKeys, "ATHENA_ACCOUNT_STATE_POSTGRES_DSN")
	require.Contains(t, specs[0].EnvironmentKeys, "ATHENA_WORM_TRADING_CATALOG_BUDGET")
	require.NotContains(t, specs[0].EnvironmentKeys, "ATHENA_WORM_MARKETS_SERVER_ADDRESS")
}

func TestWormTradingEnvironmentUsesSelectedInstance(t *testing.T) {
	specs, err := ResolveServices([]string{"worm-trading"})
	require.NoError(t, err)
	m := NewManager(InstanceKey{Checkout: t.TempDir(), Name: "worm-test"})
	env, err := m.PrepareEnvironment(map[string]string{"ATHENA_WORM_TRADING_PORT": "28090"}, specs, "managed")
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:28090", serviceAddress("worm-trading", env))
	require.Equal(t, "127.0.0.1:28090", env["ATHENA_WORM_TRADING_SERVER_ADDRESS"])
	_, err = m.PrepareEnvironment(map[string]string{"ATHENA_ACCOUNT_STATE_POSTGRES_DSN": "postgres://localhost/account"}, specs, "external")
	require.ErrorContains(t, err, "ATHENA_WORM_TRADING_POSTGRES_DSN")
}
