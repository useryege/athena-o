package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOptionsRequirePreciseTargetAndDefaultToReadOnly(t *testing.T) {
	o, err := parseOptions([]string{"--database=worm_markets", "--expected-owner=athena"})
	require.NoError(t, err)
	require.False(t, o.Apply)
	require.Equal(t, 30*time.Second, o.Timeout)
	require.Equal(t, "worm_markets", o.Database)
	require.Equal(t, "athena", o.ExpectedOwner)
	require.Empty(t, o.RetainDSNEnvs)

	o, err = parseOptions([]string{
		"--database=worm_markets",
		"--expected-owner=athena",
		"--apply",
		"--timeout=2m",
		"--retain-dsn-env=ATHENA_ACCOUNT_STATE_POSTGRES_DSN",
		"--retain-dsn-env=ATHENA_WORM_TRADING_POSTGRES_DSN",
	})
	require.NoError(t, err)
	require.True(t, o.Apply)
	require.Equal(t, 2*time.Minute, o.Timeout)
	require.Equal(t, []string{
		"ATHENA_ACCOUNT_STATE_POSTGRES_DSN",
		"ATHENA_WORM_TRADING_POSTGRES_DSN",
	}, o.RetainDSNEnvs)
}

func TestOptionsRejectAmbiguousOrUnboundedRequests(t *testing.T) {
	for _, args := range [][]string{
		{},
		{"--database=worm_markets"},
		{"--expected-owner=athena"},
		{"--database=", "--expected-owner=athena"},
		{"--database=worm_markets", "--expected-owner="},
		{"--database=worm_markets", "--expected-owner=athena", "--timeout=0"},
		{"--database=worm_markets", "--expected-owner=athena", "--timeout=-1s"},
		{"--database=worm_markets", "--expected-owner=athena", "--retain-dsn-env="},
		{"--database=worm_markets", "--expected-owner=athena", "--retain-dsn-env=BAD-NAME"},
		{"--database=worm_markets", "--expected-owner=athena", "--retain-dsn-env=KEEP", "--retain-dsn-env=KEEP"},
		{"--database=worm_markets", "--expected-owner=athena", "surprise"},
	} {
		_, err := parseOptions(args)
		require.Error(t, err, "%v", args)
	}
}

func TestRunDoesNotExposeAdminDSNOrPassword(t *testing.T) {
	const secret = "retirement-secret-password"
	t.Setenv(adminDSNEnv, "postgres://athena:"+secret+"@127.0.0.1:1/postgres?sslmode=disable")
	var out bytes.Buffer
	err := run(context.Background(), []string{
		"--database=worm_markets",
		"--expected-owner=athena",
		"--timeout=10ms",
	}, &out)
	require.Error(t, err)
	require.NotContains(t, err.Error(), secret)
	require.NotContains(t, out.String(), secret)
	require.NotContains(t, err.Error(), "postgres://")
	require.NotContains(t, out.String(), "postgres://")
	require.True(t, strings.HasSuffix(out.String(), "\n"), "failed runs still emit one JSON report")
}
