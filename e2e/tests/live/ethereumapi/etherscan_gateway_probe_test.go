package ethereumapilivee2e

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/useryege/athena/e2e/internal/e2etest"
	"github.com/useryege/athena/pkg/apiclient/etherscangateway"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	etherscanGatewayDefaultPort = "6776"
	etherscanGatewayProbeRounds = 3
)

func TestEtherscanGatewayMultiKeyStaggeredSuccessProbe(t *testing.T) {
	if e2etest.StringFromEnv(envE2ELive, "") != "1" {
		t.Skipf("%s must be 1 to run this live probe", envE2ELive)
	}
	if e2etest.StringFromEnv(envEtherscanGatewayMultiKeyStaggeredProbe, "") != "1" {
		t.Skipf("%s must be 1 to intentionally run the Etherscan Gateway multi-key probe", envEtherscanGatewayMultiKeyStaggeredProbe)
	}

	cfg := loadConfig(t)
	keys := parseEtherscanAPIKeys(e2etest.StringFromEnv(envEtherscanAPIKeys, ""))
	if len(keys) < 2 {
		t.Fatalf("%s must contain at least two API keys", envEtherscanAPIKeys)
	}

	gatewayAddrs, err := parseEtherscanGatewayAddrs(
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

	t.Logf("waiting %s before staggered Etherscan Gateway multi-key probe", etherscanRateLimitProbeCooldown)
	time.Sleep(etherscanRateLimitProbeCooldown)

	summary := runEtherscanGatewayMultiKeyStaggeredSuccessProbe(
		t,
		keys,
		gatewayAddrs,
		authToken,
		cfg,
		etherscanGatewayProbeRounds,
		etherscanMultiKeyProbeRequestInterval,
	)
	t.Logf("staggered Etherscan Gateway multi-key probe summary: %s", summary)
	for _, line := range summary.perKeySummaries() {
		t.Logf("staggered Etherscan Gateway multi-key probe key summary: %s", line)
	}
	for _, line := range summary.perGatewaySummaries() {
		t.Logf("staggered Etherscan Gateway multi-key probe gateway summary: %s", line)
	}

	if summary.success >= summary.requiredSuccess() {
		return
	}
	t.Fatalf(
		"expected staggered Etherscan Gateway aggregate success to reach at least 90%% of %d requests; summary: %s",
		summary.total,
		summary,
	)
}

func runEtherscanGatewayMultiKeyStaggeredSuccessProbe(
	t testing.TB,
	keys []string,
	gatewayAddrs []string,
	authToken string,
	cfg e2eConfig,
	rounds int,
	interval time.Duration,
) etherscanMultiKeyProbeSummary {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	conns := make([]*grpc.ClientConn, 0, len(gatewayAddrs))
	clients := make([]etherscangateway.EtherscanGatewayServiceClient, 0, len(gatewayAddrs))
	for _, gatewayAddr := range gatewayAddrs {
		conn, err := grpc.NewClient(gatewayAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			t.Fatalf("create Etherscan Gateway gRPC client for %s: %v", gatewayAddr, err)
		}
		conns = append(conns, conn)
		clients = append(clients, etherscangateway.NewEtherscanGatewayServiceClient(conn))
	}
	defer func() {
		for _, conn := range conns {
			_ = conn.Close()
		}
	}()

	totalRequests := len(keys) * rounds
	results := make(chan etherscanMultiKeyProbeResult, totalRequests)
	start := make(chan struct{})

	summary := etherscanMultiKeyProbeSummary{
		keyCount:        len(keys),
		gatewayCount:    len(gatewayAddrs),
		requestsPerKey:  rounds,
		rounds:          rounds,
		interval:        interval,
		total:           totalRequests,
		perKey:          make(map[string]*etherscanMultiKeyProbeKeySummary, len(keys)),
		perKeyOrder:     make([]string, 0, len(keys)),
		perGateway:      make(map[string]*etherscanMultiKeyProbeGatewaySummary, len(gatewayAddrs)),
		perGatewayOrder: make([]string, 0, len(gatewayAddrs)),
	}
	for i, key := range keys {
		keyLabel := fmt.Sprintf("%02d:%s", i+1, etherscanAPIKeyFingerprint(key))
		summary.perKey[keyLabel] = &etherscanMultiKeyProbeKeySummary{keyLabel: keyLabel}
		summary.perKeyOrder = append(summary.perKeyOrder, keyLabel)
	}
	for _, gatewayAddr := range gatewayAddrs {
		summary.perGateway[gatewayAddr] = &etherscanMultiKeyProbeGatewaySummary{gatewayLabel: gatewayAddr}
		summary.perGatewayOrder = append(summary.perGatewayOrder, gatewayAddr)
	}

	wg := sync.WaitGroup{}
	wg.Add(totalRequests)
	requestIndex := 0
	for round := 0; round < rounds; round++ {
		for keyIndex, key := range keys {
			index := requestIndex
			requestIndex++
			keyLabel := summary.perKeyOrder[keyIndex]
			gatewayIndex := index % len(gatewayAddrs)
			gatewayLabel := gatewayAddrs[gatewayIndex]
			client := clients[gatewayIndex]

			go func(key string, keyLabel string, gatewayLabel string, client etherscangateway.EtherscanGatewayServiceClient, requestIndex int) {
				defer wg.Done()
				<-start
				if !waitEtherscanProbeRequestInterval(ctx, time.Duration(requestIndex)*interval) {
					results <- etherscanMultiKeyProbeResult{
						keyLabel:     keyLabel,
						gatewayLabel: gatewayLabel,
						startedAt:    time.Now(),
						err:          ctx.Err(),
						redactError:  true,
					}
					return
				}

				startedAt := time.Now()
				callCtx := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+authToken)
				_, err := client.ListNormalTransactions(callCtx, &etherscangateway.ListNormalTransactionsRequest{
					ApiKey:   key,
					ChainId:  1,
					Address:  cfg.queryAddress,
					Page:     1,
					PageSize: 1,
					Sort:     etherscangateway.NormalTransactionSort_NORMAL_TRANSACTION_SORT_ASC,
				})
				results <- etherscanMultiKeyProbeResult{
					keyLabel:     keyLabel,
					gatewayLabel: gatewayLabel,
					startedAt:    startedAt,
					err:          err,
					redactError:  true,
				}
			}(key, keyLabel, gatewayLabel, client, index)
		}
	}

	startedAt := time.Now()
	close(start)
	wg.Wait()
	close(results)

	summary.elapsed = time.Since(startedAt)
	for result := range results {
		summary.record(result)
	}
	summary.finishStartSpread()
	return summary
}

func parseEtherscanGatewayAddrs(rawAddrs string, rawIPs string) ([]string, error) {
	if strings.TrimSpace(rawAddrs) != "" {
		return parseEtherscanGatewayAddrValues(rawAddrs, false)
	}
	return parseEtherscanGatewayAddrValues(rawIPs, true)
}

func parseEtherscanGatewayAddrValues(raw string, appendDefaultPort bool) ([]string, error) {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	addrs := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		value = strings.Trim(value, `"'`)
		if value == "" {
			continue
		}
		if appendDefaultPort {
			value = net.JoinHostPort(value, etherscanGatewayDefaultPort)
		} else if _, _, err := net.SplitHostPort(value); err != nil {
			return nil, fmt.Errorf("gateway address %q must include host and port: %w", value, err)
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		addrs = append(addrs, value)
	}
	return addrs, nil
}
