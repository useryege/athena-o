package ethereumapilivee2e

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/useryege/athena/e2e/internal/e2etest"
	utilethereumapi "github.com/useryege/athena/util/ethereumapi"
)

const (
	defaultEtherscanRateLimitProbeRequests = 8
	etherscanRateLimitProbeCooldown        = 2 * time.Second
	etherscanMultiKeyProbeRequestsPerKey   = 3
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

func TestEtherscanProxyMultiKeyAggregateRateLimitProbe(t *testing.T) {
	if e2etest.StringFromEnv(envE2ELive, "") != "1" {
		t.Skipf("%s must be 1 to run this live probe", envE2ELive)
	}
	if e2etest.StringFromEnv(envEtherscanProxyMultiKeyProbe, "") != "1" {
		t.Skipf("%s must be 1 to intentionally run the Etherscan proxy multi-key probe", envEtherscanProxyMultiKeyProbe)
	}

	cfg := loadConfig(t)
	keys := parseEtherscanAPIKeys(e2etest.StringFromEnv(envEtherscanAPIKeys, ""))
	if len(keys) < 2 {
		t.Fatalf("%s must contain at least two API keys", envEtherscanAPIKeys)
	}
	proxies, err := parseEtherscanProxyURLs(e2etest.StringFromEnv(envEtherscanProxyURLs, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(proxies) == 0 {
		t.Fatalf("%s must contain at least one HTTP or HTTPS proxy URL", envEtherscanProxyURLs)
	}

	t.Logf("waiting %s before Etherscan proxy multi-key probe", etherscanRateLimitProbeCooldown)
	time.Sleep(etherscanRateLimitProbeCooldown)

	summary := runEtherscanProxyMultiKeyAggregateRateLimitProbe(t, keys, proxies, cfg)
	t.Logf("Etherscan proxy multi-key aggregate probe summary: %s", summary)
	for _, line := range summary.perKeySummaries() {
		t.Logf("Etherscan proxy multi-key aggregate probe key summary: %s", line)
	}
	for _, line := range summary.perProxySummaries() {
		t.Logf("Etherscan proxy multi-key aggregate probe proxy summary: %s", line)
	}

	if summary.success >= summary.requiredSuccess() {
		return
	}
	t.Fatalf(
		"expected proxy aggregate success to reach at least 90%% of %d requests; summary: %s",
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
			go func(keyLabel string, client utilethereumapi.EthereumAPI) {
				defer wg.Done()
				<-start
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
			}(keyLabel, client)
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

func runEtherscanProxyMultiKeyAggregateRateLimitProbe(t testing.TB, keys []string, proxies []etherscanProxySpec, cfg e2eConfig) etherscanMultiKeyProbeSummary {
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
	for i, key := range keys {
		keyLabel := fmt.Sprintf("%02d:%s", i+1, etherscanAPIKeyFingerprint(key))
		proxy := proxies[i%len(proxies)]
		summary.perKey[keyLabel] = &etherscanMultiKeyProbeKeySummary{
			keyLabel:   keyLabel,
			proxyLabel: proxy.label,
		}
		summary.perKeyOrder = append(summary.perKeyOrder, keyLabel)

		client := utilethereumapi.NewEthereumAPIWithConfig(utilethereumapi.Config{
			BaseURL:    utilethereumapi.DefaultBaseURL,
			APIKey:     key,
			Timeout:    cfg.timeout,
			HTTPClient: newEtherscanProxyHTTPClient(proxy.url, cfg.timeout),
		})
		for i := 0; i < etherscanMultiKeyProbeRequestsPerKey; i++ {
			go func(keyLabel string, proxyLabel string, client utilethereumapi.EthereumAPI) {
				defer wg.Done()
				<-start
				startedAt := time.Now()
				_, err := client.ListNormalTransactions(ctx, utilethereumapi.ListNormalTransactionsOptions{
					ChainID:  1,
					Address:  cfg.queryAddress,
					Page:     1,
					PageSize: 1,
					Sort:     utilethereumapi.NormalTransactionSortASC,
				})
				results <- etherscanMultiKeyProbeResult{
					keyLabel:   keyLabel,
					proxyLabel: proxyLabel,
					startedAt:  startedAt,
					err:        err,
				}
			}(keyLabel, proxy.label, client)
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
	s.samples = append(s.samples, err.Error())
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
	keyLabel   string
	proxyLabel string
	startedAt  time.Time
	err        error
}

type etherscanMultiKeyProbeSummary struct {
	keyCount       int
	proxyCount     int
	requestsPerKey int
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
	firstStart     time.Time
	lastStart      time.Time
	startSpread    time.Duration
	samples        []string
	perKey         map[string]*etherscanMultiKeyProbeKeySummary
	perKeyOrder    []string
	perProxy       map[string]*etherscanMultiKeyProbeProxySummary
	perProxyOrder  []string
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
	if proxySummary := s.perProxy[result.proxyLabel]; proxySummary != nil {
		proxySummary.recordCategory(category)
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
	sample := etherscanProbeErrorSample(result.err, result.proxyLabel != "")
	if result.proxyLabel == "" {
		s.samples = append(s.samples, fmt.Sprintf("%s:%s", result.keyLabel, sample))
		return
	}
	s.samples = append(s.samples, fmt.Sprintf("%s@%s:%s", result.keyLabel, result.proxyLabel, sample))
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
		fmt.Sprintf("proxies=%d", s.proxyCount),
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
	}
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
	return strings.Join([]string{
		fmt.Sprintf("proxy=%s", s.proxyLabel),
		fmt.Sprintf("success=%d", s.success),
		fmt.Sprintf("rate_limit=%d", s.rateLimit),
		fmt.Sprintf("authentication=%d", s.authentication),
		fmt.Sprintf("plan=%d", s.plan),
		fmt.Sprintf("invalid_request=%d", s.invalidRequest),
		fmt.Sprintf("malformed=%d", s.malformed),
		fmt.Sprintf("upstream=%d", s.upstream),
		fmt.Sprintf("other=%d", s.other),
	}, " ")
}

func classifyEtherscanProbeError(err error) etherscanProbeResultCategory {
	if err == nil {
		return etherscanProbeResultSuccess
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
	if !redactTransport {
		return err.Error()
	}
	var apiErr *utilethereumapi.APIError
	if errors.As(err, &apiErr) {
		return err.Error()
	}
	return fmt.Sprintf("%s transport error redacted", classifyEtherscanProbeError(err))
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

type etherscanProxySpec struct {
	label string
	url   *url.URL
}

func parseEtherscanProxyURLs(raw string) ([]etherscanProxySpec, error) {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	})
	proxies := make([]etherscanProxySpec, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		rawProxy := strings.Trim(strings.TrimSpace(part), `"'`)
		if rawProxy == "" {
			continue
		}
		proxyURL, err := url.Parse(rawProxy)
		if err != nil || proxyURL.Host == "" {
			return nil, fmt.Errorf("%s contains an invalid proxy URL", envEtherscanProxyURLs)
		}
		switch proxyURL.Scheme {
		case "http", "https":
		default:
			return nil, fmt.Errorf("%s only supports http:// and https:// proxy URLs", envEtherscanProxyURLs)
		}
		key := proxyURL.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		proxies = append(proxies, etherscanProxySpec{
			label: etherscanProxyLabel(len(proxies)+1, proxyURL),
			url:   proxyURL,
		})
	}
	return proxies, nil
}

func newEtherscanProxyHTTPClient(proxyURL *url.URL, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}
}

func etherscanAPIKeyFingerprint(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 8 {
		return "****"
	}
	return fmt.Sprintf("%s...%s", key[:4], key[len(key)-3:])
}

func etherscanProxyLabel(index int, proxyURL *url.URL) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(fmt.Sprintf("%s://%s", proxyURL.Scheme, proxyURL.Host)))
	return fmt.Sprintf("%02d:%s-proxy-%08x", index, proxyURL.Scheme, hash.Sum32())
}
