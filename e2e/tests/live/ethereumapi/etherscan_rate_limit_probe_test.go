package ethereumapilivee2e

import (
	"context"
	"errors"
	"fmt"
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
)

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
	if err == nil {
		s.success++
		return
	}

	var apiErr *utilethereumapi.APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == 429 || isEtherscanRateLimitError(err) {
			s.rateLimit++
			s.addSample(err)
			return
		}
		switch apiErr.Type {
		case utilethereumapi.APIErrorTypeRateLimit:
			s.rateLimit++
		case utilethereumapi.APIErrorTypeAuthentication:
			s.authentication++
		case utilethereumapi.APIErrorTypePlan:
			s.plan++
		case utilethereumapi.APIErrorTypeInvalidRequest:
			s.invalidRequest++
		case utilethereumapi.APIErrorTypeMalformed:
			s.malformed++
		case utilethereumapi.APIErrorTypeUpstream:
			s.upstream++
		default:
			s.other++
		}
		s.addSample(err)
		return
	}

	if isEtherscanRateLimitError(err) {
		s.rateLimit++
	} else {
		s.other++
	}
	s.addSample(err)
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

func isEtherscanRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "rate limit")
}
