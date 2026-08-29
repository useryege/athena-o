package wormtrading

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/useryege/athena/util/ratelimit"
	"github.com/useryege/athena/util/worm"
)

const (
	OfficialWormAPIBaseURL = worm.DefaultBaseURL

	DefaultWormAPIAttemptTimeout    = 5 * time.Second
	DefaultWormPositionBudget       = 20 * time.Second
	DefaultWormPositionConcurrency  = 4
	defaultWormUnauthenticatedRPM   = 100
	defaultWormUnauthenticatedBurst = 2
	defaultWormAuthenticatedRPM     = 240
	defaultWormAuthenticatedBurst   = 4
)

const (
	wormErrorReconnectRequired     = "RECONNECT_REQUIRED"
	wormErrorRateLimited           = "RATE_LIMITED"
	wormErrorUnavailable           = "WORM_UNAVAILABLE"
	wormErrorRejected              = "WORM_REJECTED"
	wormErrorInvalidResponse       = "INVALID_RESPONSE"
	wormErrorTimeout               = "TIMEOUT"
	wormErrorCancelled             = "CANCELLED"
	wormErrorCredentialUnavailable = "CREDENTIAL_UNAVAILABLE"
	wormErrorConnectOutcomeUnknown = "CONNECT_OUTCOME_UNKNOWN"
	wormWarningRevocationRequired  = "REVOCATION_REQUIRED"
)

// WormAPIClient is the deliberately small portion of util/worm needed by the
// credential, position-read, and read-only execution-preview boundaries.
// util/worm remains the source of the HMAC wire protocol and response DTOs.
type WormAPIClient interface {
	CreateAuthChallenge(context.Context, worm.CreateAuthChallengeRequest) (*worm.AuthChallenge, error)
	CreateAPIKey(context.Context, worm.CreateAPIKeyRequest) (*worm.APIKeySecret, error)
	RevokeAPIKey(context.Context, string) (*worm.APIKey, error)
	EstimateMarginPosition(context.Context, worm.EstimateMarginPositionOptions) (*worm.MarginPositionEstimate, error)
	ListPositionRequests(context.Context, worm.ListPositionRequestsOptions) (*worm.ListPositionRequestsResponse, error)
	ListMarginPositions(context.Context, worm.ListMarginPositionsOptions) (*worm.ListMarginPositionsResponse, error)
}

// WormAPIClientFactory never accepts a base URL. Every production client is
// therefore pinned to Worm's official API even when callers control service
// configuration.
type WormAPIClientFactory interface {
	NewUnauthenticatedClient() (WormAPIClient, error)
	NewAuthenticatedClient(apiKey, apiSecret string) (WormAPIClient, error)
}

type officialWormAPIClientFactory struct {
	attemptTimeout time.Duration
	// The limiters are factory-scoped so creating a client per wallet cannot
	// multiply the process-wide request allowance.
	unauthenticatedRateLimiter ratelimit.Limiter
	authenticatedRateLimiter   ratelimit.Limiter
}

func NewOfficialWormAPIClientFactory(attemptTimeout time.Duration) (WormAPIClientFactory, error) {
	if attemptTimeout <= 0 {
		return nil, errors.New("worm API attempt timeout must be positive")
	}
	unauthenticatedRateLimiter, err := ratelimit.New(ratelimit.Config{
		Requests: defaultWormUnauthenticatedRPM,
		Per:      time.Minute,
		Burst:    defaultWormUnauthenticatedBurst,
	})
	if err != nil {
		return nil, fmt.Errorf("create unauthenticated Worm rate limiter: %w", err)
	}
	authenticatedRateLimiter, err := ratelimit.New(ratelimit.Config{
		Requests: defaultWormAuthenticatedRPM,
		Per:      time.Minute,
		Burst:    defaultWormAuthenticatedBurst,
	})
	if err != nil {
		return nil, fmt.Errorf("create authenticated Worm rate limiter: %w", err)
	}
	return &officialWormAPIClientFactory{
		attemptTimeout:             attemptTimeout,
		unauthenticatedRateLimiter: unauthenticatedRateLimiter,
		authenticatedRateLimiter:   authenticatedRateLimiter,
	}, nil
}

func (f *officialWormAPIClientFactory) NewUnauthenticatedClient() (WormAPIClient, error) {
	return f.newClient("", "")
}

func (f *officialWormAPIClientFactory) NewAuthenticatedClient(apiKey, apiSecret string) (WormAPIClient, error) {
	if strings.TrimSpace(apiKey) == "" || strings.TrimSpace(apiSecret) == "" {
		return nil, errors.New("worm API credential is incomplete")
	}
	return f.newClient(apiKey, apiSecret)
}

func (f *officialWormAPIClientFactory) newClient(apiKey, apiSecret string) (WormAPIClient, error) {
	rateLimiter := f.unauthenticatedRateLimiter
	if apiKey != "" || apiSecret != "" {
		rateLimiter = f.authenticatedRateLimiter
	}
	return worm.NewClient(worm.Config{
		BaseURL:     OfficialWormAPIBaseURL,
		APIKey:      apiKey,
		APISecret:   apiSecret,
		Timeout:     f.attemptTimeout,
		RateLimiter: rateLimiter,
	})
}

type wormCapabilityStatus struct {
	mu sync.RWMutex

	credentialStoreConfigured bool
	credentialStoreReachable  bool
	credentialStoreStatus     string
	wormAPIReachable          bool
	lastWormSuccessAt         int64
	lastWormFailureAt         int64
	lastWormErrorCode         string
}

type wormCapabilityStatusSnapshot struct {
	CredentialStoreConfigured bool
	CredentialStoreReachable  bool
	CredentialStoreStatus     string
	WormAPIReachable          bool
	LastWormSuccessAt         int64
	LastWormFailureAt         int64
	LastWormErrorCode         string
}

func (s *wormCapabilityStatus) snapshot() wormCapabilityStatusSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return wormCapabilityStatusSnapshot{
		CredentialStoreConfigured: s.credentialStoreConfigured,
		CredentialStoreReachable:  s.credentialStoreReachable,
		CredentialStoreStatus:     s.credentialStoreStatus,
		WormAPIReachable:          s.wormAPIReachable,
		LastWormSuccessAt:         s.lastWormSuccessAt,
		LastWormFailureAt:         s.lastWormFailureAt,
		LastWormErrorCode:         s.lastWormErrorCode,
	}
}

func (s *wormCapabilityStatus) configureStore(configured bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentialStoreConfigured = configured
	if configured {
		s.credentialStoreStatus = "unknown"
		return
	}
	s.credentialStoreReachable = false
	s.credentialStoreStatus = "not_configured"
}

func (s *wormCapabilityStatus) recordStoreSuccess() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentialStoreReachable = true
	s.credentialStoreStatus = "running"
}

func (s *wormCapabilityStatus) recordStoreFailure() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentialStoreReachable = false
	s.credentialStoreStatus = "unavailable"
}

func (s *wormCapabilityStatus) recordWormResult(err error) {
	now := time.Now().Unix()
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil {
		s.wormAPIReachable = true
		s.lastWormSuccessAt = now
		s.lastWormErrorCode = ""
		return
	}
	s.lastWormFailureAt = now
	s.lastWormErrorCode = classifyWormError(err)
	if s.lastWormErrorCode == wormErrorReconnectRequired || s.lastWormErrorCode == wormErrorRejected ||
		s.lastWormErrorCode == wormErrorRateLimited || s.lastWormErrorCode == wormErrorInvalidResponse {
		s.wormAPIReachable = true
		return
	}
	if s.lastWormErrorCode == wormErrorCancelled {
		return
	}
	s.wormAPIReachable = false
}

func classifyWormError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) {
		return wormErrorCancelled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return wormErrorTimeout
	}
	var wormErr *worm.Error
	if errors.As(err, &wormErr) {
		switch wormErr.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return wormErrorReconnectRequired
		case http.StatusTooManyRequests:
			return wormErrorRateLimited
		case http.StatusRequestTimeout, http.StatusTooEarly:
			return wormErrorUnavailable
		}
		if wormErr.StatusCode >= 500 {
			return wormErrorUnavailable
		}
		return wormErrorRejected
	}
	errText := strings.ToLower(err.Error())
	if strings.Contains(errText, "decode worm response") || strings.Contains(errText, "invalid") {
		return wormErrorInvalidResponse
	}
	return wormErrorUnavailable
}

func isWormAuthenticationError(err error) bool {
	return classifyWormError(err) == wormErrorReconnectRequired
}

func isWormNotFoundError(err error) bool {
	var wormErr *worm.Error
	return errors.As(err, &wormErr) && wormErr.StatusCode == http.StatusNotFound
}

func isTemporaryWormErrorCode(code string) bool {
	switch code {
	case wormErrorRateLimited, wormErrorUnavailable, wormErrorTimeout, wormErrorCancelled:
		return true
	default:
		return false
	}
}
