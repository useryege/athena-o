package ethereumapilivee2e

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/useryege/athena/e2e/internal/e2etest"
	utilethereumapi "github.com/useryege/athena/util/ethereumapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultEtherscanRateLimitProbeRequests = 8
	etherscanRateLimitProbeCooldown        = 2 * time.Second
	etherscanMultiKeyProbeRequestsPerKey   = 3
	etherscanMultiKeyProbeRequestInterval  = 10 * time.Millisecond
)

var etherscanAPIKeyPattern = regexp.MustCompile(`[A-Za-z0-9]{20,}`)

func TestEtherscanFreePlanRateLimitProbe(t *testing.T) {
	if e2etest.StringFromEnv(envE2ELive, "") != "1" {
		t.Skipf("%s must be 1 to run this live probe", envE2ELive)
	}
	if e2etest.StringFromEnv(envEtherscanRateLimitProbe, "") != "1" {
		t.Skipf("%s must be 1 to intentionally trigger Etherscan rate limiting", envEtherscanRateLimitProbe)
	}

	cfg := loadConfig(t)
	apiKey := e2etest.StringFromEnv(envEtherscanAPIKey, "")
	if apiKey == "" {
		apiKey = e2etest.StringFromEnv(envEtherscanAPIKeyFallback, "")
	}
	if apiKey == "" {
		t.Fatalf("%s or %s is required to run this live probe", envEtherscanAPIKey, envEtherscanAPIKeyFallback)
	}

	client := utilethereumapi.NewEthereumAPIWithConfig(utilethereumapi.Config{
		BaseURL: utilethereumapi.DefaultBaseURL,
		APIKey:  apiKey,
		Timeout: cfg.timeout,
	})

	t.Logf("waiting %s before Etherscan rate-limit probe", etherscanRateLimitProbeCooldown)
	time.Sleep(etherscanRateLimitProbeCooldown)

	summary := runEtherscanRateLimitProbe(t, client, cfg)
	t.Logf("Etherscan rate-limit probe summary: %s", summary)

	if summary.rateLimit > 0 {
		return
	}
	t.Fatalf(
		"expected at least one Etherscan rate-limit response after %d concurrent requests; summary: %s",
		defaultEtherscanRateLimitProbeRequests,
		summary,
	)
}

func TestEtherscanMultiKeyAggregateRateLimitProbe(t *testing.T) {
	if e2etest.StringFromEnv(envE2ELive, "") != "1" {
		t.Skipf("%s must be 1 to run this live probe", envE2ELive)
	}
	if e2etest.StringFromEnv(envEtherscanMultiKeyProbe, "") != "1" {
		t.Skipf("%s must be 1 to intentionally run the Etherscan multi-key probe", envEtherscanMultiKeyProbe)
	}

	cfg := loadConfig(t)
	keys := parseEtherscanAPIKeys(e2etest.StringFromEnv(envEtherscanAPIKeys, ""))
	if len(keys) < 2 {
		t.Fatalf("%s must contain at least two API keys", envEtherscanAPIKeys)
	}

	t.Logf("waiting %s before Etherscan multi-key probe", etherscanRateLimitProbeCooldown)
	time.Sleep(etherscanRateLimitProbeCooldown)

	summary := runEtherscanMultiKeyAggregateRateLimitProbe(t, keys, cfg)
	t.Logf("Etherscan multi-key aggregate probe summary: %s", summary)
	for _, line := range summary.perKeySummaries() {
		t.Logf("Etherscan multi-key aggregate probe key summary: %s", line)
	}

	if summary.success >= summary.requiredSuccess() {
		return
	}
	t.Fatalf(
		"expected aggregate success to reach at least 90%% of %d requests; summary: %s",
		summary.total,
		summary,
	)
}

func TestEtherscanMultiKeyAggregateStaggeredRateLimitProbe(t *testing.T) {
	if e2etest.StringFromEnv(envE2ELive, "") != "1" {
		t.Skipf("%s must be 1 to run this live probe", envE2ELive)
	}
	if e2etest.StringFromEnv(envEtherscanMultiKeyStaggeredProbe, "") != "1" {
		t.Skipf("%s must be 1 to intentionally run the staggered Etherscan multi-key probe", envEtherscanMultiKeyStaggeredProbe)
	}

	cfg := loadConfig(t)
	keys := parseEtherscanAPIKeys(e2etest.StringFromEnv(envEtherscanAPIKeys, ""))
	if len(keys) < 2 {
		t.Fatalf("%s must contain at least two API keys", envEtherscanAPIKeys)
	}

	t.Logf("waiting %s before staggered Etherscan multi-key probe", etherscanRateLimitProbeCooldown)
	time.Sleep(etherscanRateLimitProbeCooldown)

	summary := runEtherscanMultiKeyAggregateRateLimitProbeWithInterval(t, keys, cfg, etherscanMultiKeyProbeRequestInterval)
	t.Logf("staggered Etherscan multi-key aggregate probe summary: %s", summary)
	for _, line := range summary.perKeySummaries() {
		t.Logf("staggered Etherscan multi-key aggregate probe key summary: %s", line)
	}

	if summary.success >= summary.requiredSuccess() {
		return
	}
	t.Fatalf(
		"expected staggered aggregate success to reach at least 90%% of %d requests; summary: %s",
		summary.total,
		summary,
	)
}

func TestEtherscanProxyMultiKeyAggregateStaggeredRateLimitProbe(t *testing.T) {
	if e2etest.StringFromEnv(envE2ELive, "") != "1" {
		t.Skipf("%s must be 1 to run this live probe", envE2ELive)
	}
	if e2etest.StringFromEnv(envEtherscanProxyMultiKeyStaggeredProbe, "") != "1" {
		t.Skipf("%s must be 1 to intentionally run the staggered Etherscan proxy multi-key probe", envEtherscanProxyMultiKeyStaggeredProbe)
	}

	cfg := loadConfig(t)
	keys := parseEtherscanAPIKeys(e2etest.StringFromEnv(envEtherscanAPIKeys, ""))
	if len(keys) < 2 {
		t.Fatalf("%s must contain at least two API keys", envEtherscanAPIKeys)
	}
	proxies, err := parseEtherscanProxyURLs(e2etest.StringFromEnv(envEtherscanProxyURLs, ""))
	if err != nil {
		t.Fatalf("parse %s: %v", envEtherscanProxyURLs, err)
	}
	if len(proxies) == 0 {
		t.Fatalf("%s must contain at least one HTTP/HTTPS proxy URL", envEtherscanProxyURLs)
	}

	t.Logf("waiting %s before staggered Etherscan proxy multi-key probe", etherscanRateLimitProbeCooldown)
	time.Sleep(etherscanRateLimitProbeCooldown)

	summary := runEtherscanProxyMultiKeyAggregateRateLimitProbe(t, keys, proxies, cfg, etherscanMultiKeyProbeRequestInterval)
	t.Logf("staggered Etherscan proxy multi-key aggregate probe summary: %s", summary)
	for _, line := range summary.perKeySummaries() {
		t.Logf("staggered Etherscan proxy multi-key aggregate probe key summary: %s", line)
	}
	for _, line := range summary.perProxySummaries() {
		t.Logf("staggered Etherscan proxy multi-key aggregate probe proxy summary: %s", line)
	}

	if summary.success >= summary.requiredSuccess() {
		return
	}
	t.Fatalf(
		"expected staggered proxy aggregate success to reach at least 90%% of %d requests; summary: %s",
		summary.total,
		summary,
	)
}

func runEtherscanRateLimitProbe(t testing.TB, client utilethereumapi.EthereumAPI, cfg e2eConfig) etherscanRateLimitProbeSummary {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	startedAt := time.Now()
	results := make(chan error, defaultEtherscanRateLimitProbeRequests)
	start := make(chan struct{})

	wg := sync.WaitGroup{}
	wg.Add(defaultEtherscanRateLimitProbeRequests)
	for i := 0; i < defaultEtherscanRateLimitProbeRequests; i++ {
		go func() {
			defer wg.Done()
			<-start
			_, err := client.ListNormalTransactions(ctx, utilethereumapi.ListNormalTransactionsOptions{
				ChainID:  1,
				Address:  cfg.queryAddress,
				Page:     1,
				PageSize: 1,
				Sort:     utilethereumapi.NormalTransactionSortASC,
			})
			results <- err
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	summary := etherscanRateLimitProbeSummary{
		total:   defaultEtherscanRateLimitProbeRequests,
		elapsed: time.Since(startedAt),
	}
	for err := range results {
		summary.record(err)
	}
	return summary
}

func runEtherscanMultiKeyAggregateRateLimitProbe(t testing.TB, keys []string, cfg e2eConfig) etherscanMultiKeyProbeSummary {
	t.Helper()
	return runEtherscanMultiKeyAggregateRateLimitProbeWithInterval(t, keys, cfg, 0)
}

func runEtherscanProxyMultiKeyAggregateRateLimitProbe(t testing.TB, keys []string, proxies []etherscanProxySpec, cfg e2eConfig, interval time.Duration) etherscanMultiKeyProbeSummary {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	totalRequests := len(keys) * etherscanMultiKeyProbeRequestsPerKey
	results := make(chan etherscanMultiKeyProbeResult, totalRequests)
	start := make(chan struct{})

	summary := etherscanMultiKeyProbeSummary{
		keyCount:       len(keys),
		proxyCount:     len(proxies),
		requestsPerKey: etherscanMultiKeyProbeRequestsPerKey,
		total:          totalRequests,
		perKey:         make(map[string]*etherscanMultiKeyProbeKeySummary, len(keys)),
		perKeyOrder:    make([]string, 0, len(keys)),
		perProxy:       make(map[string]*etherscanMultiKeyProbeProxySummary, len(proxies)),
		perProxyOrder:  make([]string, 0, len(proxies)),
	}
	for _, proxy := range proxies {
		summary.perProxy[proxy.label] = &etherscanMultiKeyProbeProxySummary{proxyLabel: proxy.label}
		summary.perProxyOrder = append(summary.perProxyOrder, proxy.label)
	}

	wg := sync.WaitGroup{}
	wg.Add(totalRequests)
	requestIndex := 0
	for i, key := range keys {
		proxy := proxies[i%len(proxies)]
		keyLabel := fmt.Sprintf("%02d:%s", i+1, etherscanAPIKeyFingerprint(key))
		summary.perKey[keyLabel] = &etherscanMultiKeyProbeKeySummary{keyLabel: keyLabel, proxyLabel: proxy.label}
		summary.perKeyOrder = append(summary.perKeyOrder, keyLabel)

		client := utilethereumapi.NewEthereumAPIWithConfig(utilethereumapi.Config{
			BaseURL:    utilethereumapi.DefaultBaseURL,
			APIKey:     key,
			Timeout:    cfg.timeout,
			HTTPClient: newEtherscanProxyHTTPClient(proxy.url, cfg.timeout),
		})
		for i := 0; i < etherscanMultiKeyProbeRequestsPerKey; i++ {
			index := requestIndex
			requestIndex++
			go func(keyLabel string, proxyLabel string, client utilethereumapi.EthereumAPI, requestIndex int) {
				defer wg.Done()
				<-start
				if !waitEtherscanProbeRequestInterval(ctx, time.Duration(requestIndex)*interval) {
					results <- etherscanMultiKeyProbeResult{
						keyLabel:    keyLabel,
						proxyLabel:  proxyLabel,
						startedAt:   time.Now(),
						err:         ctx.Err(),
						redactError: true,
					}
					return
				}
				startedAt := time.Now()
				_, err := client.ListNormalTransactions(ctx, utilethereumapi.ListNormalTransactionsOptions{
					ChainID:  1,
					Address:  cfg.queryAddress,
					Page:     1,
					PageSize: 1,
					Sort:     utilethereumapi.NormalTransactionSortASC,
				})
				results <- etherscanMultiKeyProbeResult{
					keyLabel:    keyLabel,
					proxyLabel:  proxyLabel,
					startedAt:   startedAt,
					err:         err,
					redactError: true,
				}
			}(keyLabel, proxy.label, client, index)
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

func runEtherscanMultiKeyAggregateRateLimitProbeWithInterval(t testing.TB, keys []string, cfg e2eConfig, interval time.Duration) etherscanMultiKeyProbeSummary {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	totalRequests := len(keys) * etherscanMultiKeyProbeRequestsPerKey
	results := make(chan etherscanMultiKeyProbeResult, totalRequests)
	start := make(chan struct{})

	summary := etherscanMultiKeyProbeSummary{
		keyCount:       len(keys),
		requestsPerKey: etherscanMultiKeyProbeRequestsPerKey,
		total:          totalRequests,
		perKey:         make(map[string]*etherscanMultiKeyProbeKeySummary, len(keys)),
		perKeyOrder:    make([]string, 0, len(keys)),
	}

	wg := sync.WaitGroup{}
	wg.Add(totalRequests)
	requestIndex := 0
	for i, key := range keys {
		keyLabel := fmt.Sprintf("%02d:%s", i+1, etherscanAPIKeyFingerprint(key))
		summary.perKey[keyLabel] = &etherscanMultiKeyProbeKeySummary{keyLabel: keyLabel}
		summary.perKeyOrder = append(summary.perKeyOrder, keyLabel)

		client := utilethereumapi.NewEthereumAPIWithConfig(utilethereumapi.Config{
			BaseURL: utilethereumapi.DefaultBaseURL,
			APIKey:  key,
			Timeout: cfg.timeout,
		})
		for i := 0; i < etherscanMultiKeyProbeRequestsPerKey; i++ {
			index := requestIndex
			requestIndex++
			go func(keyLabel string, client utilethereumapi.EthereumAPI, requestIndex int) {
				defer wg.Done()
				<-start
				if !waitEtherscanProbeRequestInterval(ctx, time.Duration(requestIndex)*interval) {
					results <- etherscanMultiKeyProbeResult{
						keyLabel:  keyLabel,
						startedAt: time.Now(),
						err:       ctx.Err(),
					}
					return
				}
				startedAt := time.Now()
				_, err := client.ListNormalTransactions(ctx, utilethereumapi.ListNormalTransactionsOptions{
					ChainID:  1,
					Address:  cfg.queryAddress,
					Page:     1,
					PageSize: 1,
					Sort:     utilethereumapi.NormalTransactionSortASC,
				})
				results <- etherscanMultiKeyProbeResult{
					keyLabel:  keyLabel,
					startedAt: startedAt,
					err:       err,
				}
			}(keyLabel, client, index)
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

func waitEtherscanProbeRequestInterval(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

type etherscanRateLimitProbeSummary struct {
	total          int
	success        int
	rateLimit      int
	authentication int
	plan           int
	invalidRequest int
	malformed      int
	upstream       int
	other          int
	elapsed        time.Duration
	samples        []string
}

func (s *etherscanRateLimitProbeSummary) record(err error) {
	s.recordCategory(classifyEtherscanProbeError(err))
	s.addSample(err)
}

func (s *etherscanRateLimitProbeSummary) recordCategory(category etherscanProbeResultCategory) {
	switch category {
	case etherscanProbeResultSuccess:
		s.success++
	case etherscanProbeResultRateLimit:
		s.rateLimit++
	case etherscanProbeResultAuthentication:
		s.authentication++
	case etherscanProbeResultPlan:
		s.plan++
	case etherscanProbeResultInvalidRequest:
		s.invalidRequest++
	case etherscanProbeResultMalformed:
		s.malformed++
	case etherscanProbeResultUpstream:
		s.upstream++
	default:
		s.other++
	}
}

func (s *etherscanRateLimitProbeSummary) addSample(err error) {
	if err == nil || len(s.samples) >= 3 {
		return
	}
	s.samples = append(s.samples, etherscanProbeErrorSample(err, false))
}

func (s etherscanRateLimitProbeSummary) String() string {
	parts := []string{
		fmt.Sprintf("total=%d", s.total),
		fmt.Sprintf("elapsed=%s", s.elapsed.Round(time.Millisecond)),
		fmt.Sprintf("success=%d", s.success),
		fmt.Sprintf("rate_limit=%d", s.rateLimit),
		fmt.Sprintf("authentication=%d", s.authentication),
		fmt.Sprintf("plan=%d", s.plan),
		fmt.Sprintf("invalid_request=%d", s.invalidRequest),
		fmt.Sprintf("malformed=%d", s.malformed),
		fmt.Sprintf("upstream=%d", s.upstream),
		fmt.Sprintf("other=%d", s.other),
	}
	if len(s.samples) > 0 {
		parts = append(parts, fmt.Sprintf("samples=%q", s.samples))
	}
	return strings.Join(parts, " ")
}

type etherscanProbeResultCategory string

const (
	etherscanProbeResultSuccess        etherscanProbeResultCategory = "success"
	etherscanProbeResultRateLimit      etherscanProbeResultCategory = "rate_limit"
	etherscanProbeResultAuthentication etherscanProbeResultCategory = "authentication"
	etherscanProbeResultPlan           etherscanProbeResultCategory = "plan"
	etherscanProbeResultInvalidRequest etherscanProbeResultCategory = "invalid_request"
	etherscanProbeResultMalformed      etherscanProbeResultCategory = "malformed"
	etherscanProbeResultUpstream       etherscanProbeResultCategory = "upstream"
	etherscanProbeResultOther          etherscanProbeResultCategory = "other"
)

type etherscanMultiKeyProbeResult struct {
	keyLabel     string
	proxyLabel   string
	gatewayLabel string
	startedAt    time.Time
	err          error
	redactError  bool
}

type etherscanMultiKeyProbeSummary struct {
	keyCount        int
	proxyCount      int
	gatewayCount    int
	requestsPerKey  int
	rounds          int
	interval        time.Duration
	total           int
	success         int
	rateLimit       int
	authentication  int
	plan            int
	invalidRequest  int
	malformed       int
	upstream        int
	other           int
	elapsed         time.Duration
	firstStart      time.Time
	lastStart       time.Time
	startSpread     time.Duration
	samples         []string
	perKey          map[string]*etherscanMultiKeyProbeKeySummary
	perKeyOrder     []string
	perProxy        map[string]*etherscanMultiKeyProbeProxySummary
	perProxyOrder   []string
	perGateway      map[string]*etherscanMultiKeyProbeGatewaySummary
	perGatewayOrder []string
}

func (s *etherscanMultiKeyProbeSummary) record(result etherscanMultiKeyProbeResult) {
	if s.firstStart.IsZero() || result.startedAt.Before(s.firstStart) {
		s.firstStart = result.startedAt
	}
	if result.startedAt.After(s.lastStart) {
		s.lastStart = result.startedAt
	}

	category := classifyEtherscanProbeError(result.err)
	s.recordCategory(category)
	if keySummary := s.perKey[result.keyLabel]; keySummary != nil {
		keySummary.recordCategory(category)
	}
	if result.proxyLabel != "" {
		if proxySummary := s.perProxy[result.proxyLabel]; proxySummary != nil {
			proxySummary.recordCategory(category)
		}
	}
	if result.gatewayLabel != "" {
		if gatewaySummary := s.perGateway[result.gatewayLabel]; gatewaySummary != nil {
			gatewaySummary.recordCategory(category)
		}
	}
	s.addSample(result)
}

func (s *etherscanMultiKeyProbeSummary) recordCategory(category etherscanProbeResultCategory) {
	switch category {
	case etherscanProbeResultSuccess:
		s.success++
	case etherscanProbeResultRateLimit:
		s.rateLimit++
	case etherscanProbeResultAuthentication:
		s.authentication++
	case etherscanProbeResultPlan:
		s.plan++
	case etherscanProbeResultInvalidRequest:
		s.invalidRequest++
	case etherscanProbeResultMalformed:
		s.malformed++
	case etherscanProbeResultUpstream:
		s.upstream++
	default:
		s.other++
	}
}

func (s *etherscanMultiKeyProbeSummary) addSample(result etherscanMultiKeyProbeResult) {
	if result.err == nil || len(s.samples) >= 5 {
		return
	}
	label := result.keyLabel
	if result.proxyLabel != "" {
		label = fmt.Sprintf("%s@%s", result.keyLabel, result.proxyLabel)
	}
	if result.gatewayLabel != "" {
		label = fmt.Sprintf("%s@%s", label, result.gatewayLabel)
	}
	s.samples = append(s.samples, fmt.Sprintf("%s:%s", label, etherscanProbeErrorSample(result.err, result.redactError)))
}

func (s *etherscanMultiKeyProbeSummary) finishStartSpread() {
	if s.firstStart.IsZero() || s.lastStart.IsZero() {
		return
	}
	s.startSpread = s.lastStart.Sub(s.firstStart)
}

func (s etherscanMultiKeyProbeSummary) requiredSuccess() int {
	return (s.total*9 + 9) / 10
}

func (s etherscanMultiKeyProbeSummary) String() string {
	parts := []string{
		fmt.Sprintf("keys=%d", s.keyCount),
	}
	if s.proxyCount > 0 {
		parts = append(parts, fmt.Sprintf("proxies=%d", s.proxyCount))
	}
	if s.gatewayCount > 0 {
		parts = append(parts, fmt.Sprintf("gateways=%d", s.gatewayCount))
	}
	if s.rounds > 0 {
		parts = append(parts, fmt.Sprintf("rounds=%d", s.rounds))
	}
	if s.interval > 0 {
		parts = append(parts, fmt.Sprintf("interval=%s", s.interval))
	}
	parts = append(parts,
		fmt.Sprintf("requests_per_key=%d", s.requestsPerKey),
		fmt.Sprintf("total=%d", s.total),
		fmt.Sprintf("required_success=%d", s.requiredSuccess()),
		fmt.Sprintf("elapsed=%s", s.elapsed.Round(time.Millisecond)),
		fmt.Sprintf("start_spread=%s", s.startSpread.Round(time.Millisecond)),
		fmt.Sprintf("success=%d", s.success),
		fmt.Sprintf("rate_limit=%d", s.rateLimit),
		fmt.Sprintf("authentication=%d", s.authentication),
		fmt.Sprintf("plan=%d", s.plan),
		fmt.Sprintf("invalid_request=%d", s.invalidRequest),
		fmt.Sprintf("malformed=%d", s.malformed),
		fmt.Sprintf("upstream=%d", s.upstream),
		fmt.Sprintf("other=%d", s.other),
	)
	if len(s.samples) > 0 {
		parts = append(parts, fmt.Sprintf("samples=%q", s.samples))
	}
	return strings.Join(parts, " ")
}

func (s etherscanMultiKeyProbeSummary) perKeySummaries() []string {
	lines := make([]string, 0, len(s.perKeyOrder))
	for _, keyLabel := range s.perKeyOrder {
		keySummary := s.perKey[keyLabel]
		if keySummary == nil {
			continue
		}
		lines = append(lines, keySummary.String())
	}
	return lines
}

func (s etherscanMultiKeyProbeSummary) perProxySummaries() []string {
	lines := make([]string, 0, len(s.perProxyOrder))
	for _, proxyLabel := range s.perProxyOrder {
		proxySummary := s.perProxy[proxyLabel]
		if proxySummary == nil {
			continue
		}
		lines = append(lines, proxySummary.String())
	}
	return lines
}

func (s etherscanMultiKeyProbeSummary) perGatewaySummaries() []string {
	lines := make([]string, 0, len(s.perGatewayOrder))
	for _, gatewayLabel := range s.perGatewayOrder {
		gatewaySummary := s.perGateway[gatewayLabel]
		if gatewaySummary == nil {
			continue
		}
		lines = append(lines, gatewaySummary.String())
	}
	return lines
}

type etherscanMultiKeyProbeKeySummary struct {
	keyLabel       string
	proxyLabel     string
	success        int
	rateLimit      int
	authentication int
	plan           int
	invalidRequest int
	malformed      int
	upstream       int
	other          int
}

func (s *etherscanMultiKeyProbeKeySummary) recordCategory(category etherscanProbeResultCategory) {
	switch category {
	case etherscanProbeResultSuccess:
		s.success++
	case etherscanProbeResultRateLimit:
		s.rateLimit++
	case etherscanProbeResultAuthentication:
		s.authentication++
	case etherscanProbeResultPlan:
		s.plan++
	case etherscanProbeResultInvalidRequest:
		s.invalidRequest++
	case etherscanProbeResultMalformed:
		s.malformed++
	case etherscanProbeResultUpstream:
		s.upstream++
	default:
		s.other++
	}
}

func (s etherscanMultiKeyProbeKeySummary) String() string {
	parts := []string{
		fmt.Sprintf("key=%s", s.keyLabel),
	}
	if s.proxyLabel != "" {
		parts = append(parts, fmt.Sprintf("proxy=%s", s.proxyLabel))
	}
	parts = append(parts,
		fmt.Sprintf("success=%d", s.success),
		fmt.Sprintf("rate_limit=%d", s.rateLimit),
		fmt.Sprintf("authentication=%d", s.authentication),
		fmt.Sprintf("plan=%d", s.plan),
		fmt.Sprintf("invalid_request=%d", s.invalidRequest),
		fmt.Sprintf("malformed=%d", s.malformed),
		fmt.Sprintf("upstream=%d", s.upstream),
		fmt.Sprintf("other=%d", s.other),
	)
	return strings.Join(parts, " ")
}

type etherscanMultiKeyProbeProxySummary struct {
	proxyLabel     string
	success        int
	rateLimit      int
	authentication int
	plan           int
	invalidRequest int
	malformed      int
	upstream       int
	other          int
}

type etherscanMultiKeyProbeGatewaySummary struct {
	gatewayLabel   string
	success        int
	rateLimit      int
	authentication int
	plan           int
	invalidRequest int
	malformed      int
	upstream       int
	other          int
}

func (s *etherscanMultiKeyProbeGatewaySummary) recordCategory(category etherscanProbeResultCategory) {
	switch category {
	case etherscanProbeResultSuccess:
		s.success++
	case etherscanProbeResultRateLimit:
		s.rateLimit++
	case etherscanProbeResultAuthentication:
		s.authentication++
	case etherscanProbeResultPlan:
		s.plan++
	case etherscanProbeResultInvalidRequest:
		s.invalidRequest++
	case etherscanProbeResultMalformed:
		s.malformed++
	case etherscanProbeResultUpstream:
		s.upstream++
	default:
		s.other++
	}
}

func (s etherscanMultiKeyProbeGatewaySummary) String() string {
	parts := []string{
		fmt.Sprintf("gateway=%s", s.gatewayLabel),
		fmt.Sprintf("success=%d", s.success),
		fmt.Sprintf("rate_limit=%d", s.rateLimit),
		fmt.Sprintf("authentication=%d", s.authentication),
		fmt.Sprintf("plan=%d", s.plan),
		fmt.Sprintf("invalid_request=%d", s.invalidRequest),
		fmt.Sprintf("malformed=%d", s.malformed),
		fmt.Sprintf("upstream=%d", s.upstream),
		fmt.Sprintf("other=%d", s.other),
	}
	return strings.Join(parts, " ")
}

func (s *etherscanMultiKeyProbeProxySummary) recordCategory(category etherscanProbeResultCategory) {
	switch category {
	case etherscanProbeResultSuccess:
		s.success++
	case etherscanProbeResultRateLimit:
		s.rateLimit++
	case etherscanProbeResultAuthentication:
		s.authentication++
	case etherscanProbeResultPlan:
		s.plan++
	case etherscanProbeResultInvalidRequest:
		s.invalidRequest++
	case etherscanProbeResultMalformed:
		s.malformed++
	case etherscanProbeResultUpstream:
		s.upstream++
	default:
		s.other++
	}
}

func (s etherscanMultiKeyProbeProxySummary) String() string {
	parts := []string{
		fmt.Sprintf("proxy=%s", s.proxyLabel),
		fmt.Sprintf("success=%d", s.success),
		fmt.Sprintf("rate_limit=%d", s.rateLimit),
		fmt.Sprintf("authentication=%d", s.authentication),
		fmt.Sprintf("plan=%d", s.plan),
		fmt.Sprintf("invalid_request=%d", s.invalidRequest),
		fmt.Sprintf("malformed=%d", s.malformed),
		fmt.Sprintf("upstream=%d", s.upstream),
		fmt.Sprintf("other=%d", s.other),
	}
	return strings.Join(parts, " ")
}

func classifyEtherscanProbeError(err error) etherscanProbeResultCategory {
	if err == nil {
		return etherscanProbeResultSuccess
	}

	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.OK:
			return etherscanProbeResultSuccess
		case codes.ResourceExhausted:
			return etherscanProbeResultRateLimit
		case codes.Unauthenticated:
			return etherscanProbeResultAuthentication
		case codes.PermissionDenied:
			return etherscanProbeResultPlan
		case codes.InvalidArgument:
			return etherscanProbeResultInvalidRequest
		case codes.DataLoss:
			return etherscanProbeResultMalformed
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return etherscanProbeResultUpstream
		}
	}

	var apiErr *utilethereumapi.APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == 429 || isEtherscanRateLimitError(err) {
			return etherscanProbeResultRateLimit
		}
		switch apiErr.Type {
		case utilethereumapi.APIErrorTypeRateLimit:
			return etherscanProbeResultRateLimit
		case utilethereumapi.APIErrorTypeAuthentication:
			return etherscanProbeResultAuthentication
		case utilethereumapi.APIErrorTypePlan:
			return etherscanProbeResultPlan
		case utilethereumapi.APIErrorTypeInvalidRequest:
			return etherscanProbeResultInvalidRequest
		case utilethereumapi.APIErrorTypeMalformed:
			return etherscanProbeResultMalformed
		case utilethereumapi.APIErrorTypeUpstream:
			return etherscanProbeResultUpstream
		default:
			return etherscanProbeResultOther
		}
	}

	if isEtherscanRateLimitError(err) {
		return etherscanProbeResultRateLimit
	}
	return etherscanProbeResultOther
}

func isEtherscanRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "rate limit")
}

func etherscanProbeErrorSample(err error, redactTransport bool) string {
	if err == nil {
		return ""
	}
	var apiErr *utilethereumapi.APIError
	if redactTransport && !errors.As(err, &apiErr) {
		return fmt.Sprintf("%s transport error redacted", classifyEtherscanProbeError(err))
	}
	return err.Error()
}

type etherscanProxySpec struct {
	label string
	url   *url.URL
}

func parseEtherscanAPIKeys(raw string) []string {
	matches := etherscanAPIKeyPattern.FindAllString(raw, -1)
	keys := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		key := strings.TrimSpace(match)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys
}

func parseEtherscanProxyURLs(raw string) ([]etherscanProxySpec, error) {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	})
	proxies := make([]etherscanProxySpec, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		value = strings.Trim(value, `"'`)
		if value == "" {
			continue
		}
		proxyURL, err := url.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL %q: %w", etherscanProxyDisplayLabel(len(proxies)+1, value), err)
		}
		if proxyURL.Scheme != "http" && proxyURL.Scheme != "https" {
			return nil, fmt.Errorf("proxy URL %q must use http or https scheme", etherscanProxyDisplayLabel(len(proxies)+1, value))
		}
		if proxyURL.Host == "" {
			return nil, fmt.Errorf("proxy URL %q must include a host", etherscanProxyDisplayLabel(len(proxies)+1, value))
		}
		proxies = append(proxies, etherscanProxySpec{
			label: etherscanProxyLabel(len(proxies)+1, proxyURL),
			url:   proxyURL,
		})
	}
	return proxies, nil
}

func newEtherscanProxyHTTPClient(proxyURL *url.URL, timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

func etherscanProxyLabel(index int, proxyURL *url.URL) string {
	return fmt.Sprintf("%02d:%s://%s", index, proxyURL.Scheme, proxyURL.Host)
}

func etherscanProxyDisplayLabel(index int, raw string) string {
	proxyURL, err := url.Parse(raw)
	if err != nil || proxyURL.Host == "" {
		return fmt.Sprintf("%02d:<invalid>", index)
	}
	return etherscanProxyLabel(index, proxyURL)
}

func etherscanAPIKeyFingerprint(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 8 {
		return "****"
	}
	return fmt.Sprintf("%s...%s", key[:4], key[len(key)-3:])
}
