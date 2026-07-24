package etherscangatewaylivee2e

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/useryege/athena/e2e/internal/e2etest"
	"github.com/useryege/athena/internal/etherscangatewayprobe"
)

const (
	etherscanGatewayProbeRounds          = 6
	etherscanGatewayProbeCooldown        = 2 * time.Second
	etherscanGatewayProbeRequestInterval = 10 * time.Millisecond
)

func TestEtherscanGatewayMultiKeyStaggeredSuccessProbe(t *testing.T) {
	if e2etest.StringFromEnv(envE2ELive, "") != "1" {
		t.Skipf("%s must be 1 to run this live probe", envE2ELive)
	}
	if e2etest.StringFromEnv(envEtherscanGatewayMultiKeyStaggeredProbe, "") != "1" {
		t.Skipf("%s must be 1 to intentionally run the Etherscan Gateway multi-key probe", envEtherscanGatewayMultiKeyStaggeredProbe)
	}

	cfg := loadConfig(t)
	keys := etherscangatewayprobe.ParseAPIKeys(e2etest.StringFromEnv(envEtherscanAPIKeys, ""))
	if len(keys) < 2 {
		t.Fatalf("%s must contain at least two API keys", envEtherscanAPIKeys)
	}

	gatewayAddrs, err := etherscangatewayprobe.ParseGatewayAddrs(
		e2etest.StringFromEnv(envEtherscanGatewayAddrs, ""),
		e2etest.StringFromEnv(envEtherscanGatewayIPs, ""),
	)
	if err != nil {
		t.Fatalf("parse Etherscan Gateway addresses: %v", err)
	}
	if len(gatewayAddrs) == 0 {
		t.Fatalf("%s or %s must contain at least one gateway", envEtherscanGatewayAddrs, envEtherscanGatewayIPs)
	}

	authToken := strings.TrimSpace(e2etest.StringFromEnv(envEtherscanGatewayAuthToken, ""))
	if authToken == "" {
		t.Fatalf("%s is required to run this live probe", envEtherscanGatewayAuthToken)
	}

	t.Logf("waiting %s before staggered Etherscan Gateway multi-key probe", etherscanGatewayProbeCooldown)
	time.Sleep(etherscanGatewayProbeCooldown)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()
	summary, err := etherscangatewayprobe.RunGatewayMultiKeyStaggered(ctx, etherscangatewayprobe.Config{
		APIKeys:        keys,
		GatewayAddrs:   gatewayAddrs,
		AuthToken:      authToken,
		QueryAddress:   cfg.queryAddress,
		RequestsPerKey: etherscanGatewayProbeRounds,
		Interval:       etherscanGatewayProbeRequestInterval,
	})
	if err != nil {
		t.Fatalf("run staggered Etherscan Gateway multi-key probe: %v", err)
	}
	t.Logf("staggered Etherscan Gateway multi-key probe summary: %s", summary)
	for _, line := range summary.PerKeyLogLines() {
		t.Logf("staggered Etherscan Gateway multi-key probe key summary: %s", line)
	}
	for _, line := range summary.PerGatewayLogLines() {
		t.Logf("staggered Etherscan Gateway multi-key probe gateway summary: %s", line)
	}

	if summary.Counts.Success >= summary.RequiredSuccess() {
		return
	}
	t.Fatalf(
		"expected staggered Etherscan Gateway aggregate success to reach at least 90%% of %d requests; summary: %s",
		summary.Total,
		summary,
	)
}
