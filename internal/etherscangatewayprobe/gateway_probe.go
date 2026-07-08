package etherscangatewayprobe

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	etherscangatewaypkg "github.com/useryege/athena/pkg/apiclient/etherscangateway"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	DefaultGatewayPort       = "6776"
	AuthorizationMetadataKey = "authorization"
)

var apiKeyPattern = regexp.MustCompile(`[A-Za-z0-9]{20,}`)

type Category string

const (
	CategorySuccess        Category = "success"
	CategoryRateLimit      Category = "rate_limit"
	CategoryAuthentication Category = "authentication"
	CategoryPlan           Category = "plan"
	CategoryInvalidRequest Category = "invalid_request"
	CategoryMalformed      Category = "malformed"
	CategoryUpstream       Category = "upstream"
	CategoryOther          Category = "other"
)

type Counts struct {
	Success        int
	RateLimit      int
	Authentication int
	Plan           int
	InvalidRequest int
	Malformed      int
	Upstream       int
	Other          int
}

func (c *Counts) Record(category Category) {
	switch category {
	case CategorySuccess:
		c.Success++
	case CategoryRateLimit:
		c.RateLimit++
	case CategoryAuthentication:
		c.Authentication++
	case CategoryPlan:
		c.Plan++
	case CategoryInvalidRequest:
		c.InvalidRequest++
	case CategoryMalformed:
		c.Malformed++
	case CategoryUpstream:
		c.Upstream++
	default:
		c.Other++
	}
}

func (c Counts) Total() int {
	return c.Success + c.RateLimit + c.Authentication + c.Plan + c.InvalidRequest + c.Malformed + c.Upstream + c.Other
}

type EntitySummary struct {
	Label  string
	Counts Counts
}

func (s EntitySummary) String(kind string) string {
	return strings.Join([]string{
		fmt.Sprintf("%s=%s", kind, s.Label),
		fmt.Sprintf("success=%d", s.Counts.Success),
		fmt.Sprintf("rate_limit=%d", s.Counts.RateLimit),
		fmt.Sprintf("authentication=%d", s.Counts.Authentication),
		fmt.Sprintf("plan=%d", s.Counts.Plan),
		fmt.Sprintf("invalid_request=%d", s.Counts.InvalidRequest),
		fmt.Sprintf("malformed=%d", s.Counts.Malformed),
		fmt.Sprintf("upstream=%d", s.Counts.Upstream),
		fmt.Sprintf("other=%d", s.Counts.Other),
	}, " ")
}

type Config struct {
	APIKeys        []string
	GatewayAddrs   []string
	AuthToken      string
	QueryAddress   string
	RequestsPerKey int
	Interval       time.Duration
}

type Result struct {
	KeyCount         int
	GatewayCount     int
	RequestsPerKey   int
	Rounds           int
	Interval         time.Duration
	Total            int
	Counts           Counts
	Elapsed          time.Duration
	FirstStart       time.Time
	LastStart        time.Time
	StartSpread      time.Duration
	Samples          []string
	KeySummaries     []EntitySummary
	GatewaySummaries []EntitySummary
}

func (r Result) RequiredSuccess() int {
	return RequiredSuccess(r.Total)
}

func (r Result) String() string {
	parts := []string{
		fmt.Sprintf("keys=%d", r.KeyCount),
		fmt.Sprintf("gateways=%d", r.GatewayCount),
		fmt.Sprintf("rounds=%d", r.Rounds),
	}
	if r.Interval > 0 {
		parts = append(parts, fmt.Sprintf("interval=%s", r.Interval))
	}
	parts = append(parts,
		fmt.Sprintf("requests_per_key=%d", r.RequestsPerKey),
		fmt.Sprintf("total=%d", r.Total),
		fmt.Sprintf("required_success=%d", r.RequiredSuccess()),
		fmt.Sprintf("elapsed=%s", r.Elapsed.Round(time.Millisecond)),
		fmt.Sprintf("start_spread=%s", r.StartSpread.Round(time.Millisecond)),
		fmt.Sprintf("success=%d", r.Counts.Success),
		fmt.Sprintf("rate_limit=%d", r.Counts.RateLimit),
		fmt.Sprintf("authentication=%d", r.Counts.Authentication),
		fmt.Sprintf("plan=%d", r.Counts.Plan),
		fmt.Sprintf("invalid_request=%d", r.Counts.InvalidRequest),
		fmt.Sprintf("malformed=%d", r.Counts.Malformed),
		fmt.Sprintf("upstream=%d", r.Counts.Upstream),
		fmt.Sprintf("other=%d", r.Counts.Other),
	)
	if len(r.Samples) > 0 {
		parts = append(parts, fmt.Sprintf("samples=%q", r.Samples))
	}
	return strings.Join(parts, " ")
}

func (r Result) PerKeyLogLines() []string {
	lines := make([]string, 0, len(r.KeySummaries))
	for _, summary := range r.KeySummaries {
		lines = append(lines, summary.String("key"))
	}
	return lines
}

func (r Result) PerGatewayLogLines() []string {
	lines := make([]string, 0, len(r.GatewaySummaries))
	for _, summary := range r.GatewaySummaries {
		lines = append(lines, summary.String("gateway"))
	}
	return lines
}

func RequiredSuccess(total int) int {
	return (total*9 + 9) / 10
}

func ParseAPIKeys(raw string) []string {
	matches := apiKeyPattern.FindAllString(raw, -1)
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

func ParseGatewayAddrs(rawAddrs string, rawIPs string) ([]string, error) {
	if strings.TrimSpace(rawAddrs) != "" {
		return parseGatewayAddrValues(rawAddrs, false)
	}
	return parseGatewayAddrValues(rawIPs, true)
}

func RunGatewayMultiKeyStaggered(ctx context.Context, cfg Config) (Result, error) {
	result := Result{
		KeyCount:       len(cfg.APIKeys),
		GatewayCount:   len(cfg.GatewayAddrs),
		RequestsPerKey: cfg.RequestsPerKey,
		Rounds:         cfg.RequestsPerKey,
		Interval:       cfg.Interval,
		Total:          len(cfg.APIKeys) * cfg.RequestsPerKey,
	}
	if len(cfg.APIKeys) == 0 {
		return result, fmt.Errorf("at least one Etherscan API key is required")
	}
	if len(cfg.GatewayAddrs) == 0 {
		return result, fmt.Errorf("at least one Etherscan Gateway address is required")
	}
	if strings.TrimSpace(cfg.AuthToken) == "" {
		return result, fmt.Errorf("Etherscan Gateway auth token is required")
	}
	if strings.TrimSpace(cfg.QueryAddress) == "" {
		return result, fmt.Errorf("query address is required")
	}
	if cfg.RequestsPerKey <= 0 {
		return result, fmt.Errorf("requests per key must be positive")
	}
	if cfg.Interval < 0 {
		return result, fmt.Errorf("interval must not be negative")
	}

	conns := make([]*grpc.ClientConn, 0, len(cfg.GatewayAddrs))
	clients := make([]etherscangatewaypkg.EtherscanGatewayServiceClient, 0, len(cfg.GatewayAddrs))
	for _, gatewayAddr := range cfg.GatewayAddrs {
		conn, err := grpc.NewClient(gatewayAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return result, fmt.Errorf("create Etherscan Gateway gRPC client for %s: %w", gatewayAddr, err)
		}
		conns = append(conns, conn)
		clients = append(clients, etherscangatewaypkg.NewEtherscanGatewayServiceClient(conn))
	}
	defer func() {
		for _, conn := range conns {
			_ = conn.Close()
		}
	}()

	totalRequests := result.Total
	results := make(chan probeRequestResult, totalRequests)
	start := make(chan struct{})

	perKey := make(map[string]*EntitySummary, len(cfg.APIKeys))
	perKeyOrder := make([]string, 0, len(cfg.APIKeys))
	for i, key := range cfg.APIKeys {
		keyLabel := fmt.Sprintf("%02d:%s", i+1, APIKeyFingerprint(key))
		perKey[keyLabel] = &EntitySummary{Label: keyLabel}
		perKeyOrder = append(perKeyOrder, keyLabel)
	}

	perGateway := make(map[string]*EntitySummary, len(cfg.GatewayAddrs))
	perGatewayOrder := make([]string, 0, len(cfg.GatewayAddrs))
	for _, gatewayAddr := range cfg.GatewayAddrs {
		perGateway[gatewayAddr] = &EntitySummary{Label: gatewayAddr}
		perGatewayOrder = append(perGatewayOrder, gatewayAddr)
	}

	wg := sync.WaitGroup{}
	wg.Add(totalRequests)
	requestIndex := 0
	for round := 0; round < cfg.RequestsPerKey; round++ {
		for keyIndex, key := range cfg.APIKeys {
			index := requestIndex
			requestIndex++
			keyLabel := perKeyOrder[keyIndex]
			gatewayIndex := index % len(cfg.GatewayAddrs)
			gatewayLabel := cfg.GatewayAddrs[gatewayIndex]
			client := clients[gatewayIndex]

			go func(key string, keyLabel string, gatewayLabel string, client etherscangatewaypkg.EtherscanGatewayServiceClient, requestIndex int) {
				defer wg.Done()
				<-start
				if !waitRequestInterval(ctx, time.Duration(requestIndex)*cfg.Interval) {
					results <- probeRequestResult{
						keyLabel:     keyLabel,
						gatewayLabel: gatewayLabel,
						startedAt:    time.Now(),
						err:          ctx.Err(),
					}
					return
				}

				startedAt := time.Now()
				callCtx := metadata.AppendToOutgoingContext(ctx, AuthorizationMetadataKey, "Bearer "+strings.TrimSpace(cfg.AuthToken))
				_, err := client.ListNormalTransactions(callCtx, &etherscangatewaypkg.ListNormalTransactionsRequest{
					ApiKey:   key,
					ChainId:  1,
					Address:  cfg.QueryAddress,
					Page:     1,
					PageSize: 1,
					Sort:     etherscangatewaypkg.NormalTransactionSort_NORMAL_TRANSACTION_SORT_ASC,
				})
				results <- probeRequestResult{
					keyLabel:     keyLabel,
					gatewayLabel: gatewayLabel,
					startedAt:    startedAt,
					err:          err,
				}
			}(key, keyLabel, gatewayLabel, client, index)
		}
	}

	startedAt := time.Now()
	close(start)
	wg.Wait()
	close(results)

	result.Elapsed = time.Since(startedAt)
	for requestResult := range results {
		result.record(requestResult, perKey, perGateway)
	}
	result.finishStartSpread()
	result.KeySummaries = orderedSummaries(perKeyOrder, perKey)
	result.GatewaySummaries = orderedSummaries(perGatewayOrder, perGateway)
	return result, nil
}

func parseGatewayAddrValues(raw string, appendDefaultPort bool) ([]string, error) {
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
			if net.ParseIP(value) == nil {
				return nil, fmt.Errorf("gateway IP %q must be a pure IP address", value)
			}
			value = net.JoinHostPort(value, DefaultGatewayPort)
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

type probeRequestResult struct {
	keyLabel     string
	gatewayLabel string
	startedAt    time.Time
	err          error
}

func (r *Result) record(requestResult probeRequestResult, perKey map[string]*EntitySummary, perGateway map[string]*EntitySummary) {
	if r.FirstStart.IsZero() || requestResult.startedAt.Before(r.FirstStart) {
		r.FirstStart = requestResult.startedAt
	}
	if requestResult.startedAt.After(r.LastStart) {
		r.LastStart = requestResult.startedAt
	}

	category := ClassifyError(requestResult.err)
	r.Counts.Record(category)
	if keySummary := perKey[requestResult.keyLabel]; keySummary != nil {
		keySummary.Counts.Record(category)
	}
	if gatewaySummary := perGateway[requestResult.gatewayLabel]; gatewaySummary != nil {
		gatewaySummary.Counts.Record(category)
	}
	r.addSample(requestResult, category)
}

func (r *Result) addSample(requestResult probeRequestResult, category Category) {
	if requestResult.err == nil || len(r.Samples) >= 5 {
		return
	}
	label := fmt.Sprintf("%s@%s", requestResult.keyLabel, requestResult.gatewayLabel)
	r.Samples = append(r.Samples, fmt.Sprintf("%s:%s transport error redacted", label, category))
}

func (r *Result) finishStartSpread() {
	if r.FirstStart.IsZero() || r.LastStart.IsZero() {
		return
	}
	r.StartSpread = r.LastStart.Sub(r.FirstStart)
}

func orderedSummaries(order []string, summaries map[string]*EntitySummary) []EntitySummary {
	items := make([]EntitySummary, 0, len(order))
	for _, label := range order {
		item := summaries[label]
		if item == nil {
			continue
		}
		items = append(items, *item)
	}
	return items
}

func waitRequestInterval(ctx context.Context, delay time.Duration) bool {
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

func ClassifyError(err error) Category {
	if err == nil {
		return CategorySuccess
	}

	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.OK:
			return CategorySuccess
		case codes.ResourceExhausted:
			return CategoryRateLimit
		case codes.Unauthenticated:
			return CategoryAuthentication
		case codes.PermissionDenied:
			return CategoryPlan
		case codes.InvalidArgument:
			return CategoryInvalidRequest
		case codes.DataLoss:
			return CategoryMalformed
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return CategoryUpstream
		}
	}

	if strings.Contains(strings.ToLower(err.Error()), "rate limit") {
		return CategoryRateLimit
	}
	return CategoryOther
}

func APIKeyFingerprint(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 8 {
		return "****"
	}
	return fmt.Sprintf("%s...%s", key[:4], key[len(key)-3:])
}
