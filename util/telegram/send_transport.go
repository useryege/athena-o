package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SendError preserves the classification before sanitizing provider error text.
type SendError struct {
	Kind       string
	Code       string
	RetryAfter time.Duration
	Err        error
}

func (e *SendError) Error() string {
	return fmt.Sprintf("telegram send %s (%s): %v", e.Kind, e.Code, e.Err)
}
func (e *SendError) Unwrap() error { return e.Err }

type sendAdmissionKey struct{}
type sendTimeoutKey struct{}
type sendStartedKey struct{}
type sendObservationKey struct{}
type sendObservation struct {
	once             sync.Once
	started          bool
	callback         func(time.Time)
	releaseAdmission func()
	response         *telegramAPIResponse
}

// WithSendStarted attaches a constant-time callback invoked at actual RoundTrip entry.
// Callers should hand off the timestamp through a buffered channel, never wait for I/O here.
func WithSendStarted(ctx context.Context, started func(time.Time)) context.Context {
	return context.WithValue(ctx, sendStartedKey{}, started)
}

// WithSendAdmission attaches a cancellable pre-start gate. Waiting must hold no
// application gate. Its release callback is invoked immediately after the actual
// RoundTrip started handshake, or on any failure before that handshake.
func WithSendAdmission(ctx context.Context, admit func(context.Context) (func(), error)) context.Context {
	return context.WithValue(ctx, sendAdmissionKey{}, admit)
}

// WithSendTimeout limits HTTP execution, excluding local pre-start admission.
func WithSendTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return context.WithValue(ctx, sendTimeoutKey{}, timeout)
}

// SDK request construction is complete here; admission precedes http.Client's
// timeout. The read lock is handed to sendTransport so Tighten cannot overtake
// admission between Do and the actual started event. It never spans HTTP I/O.
type sendHTTPClient struct{ client *http.Client }

func (c *sendHTTPClient) Do(req *http.Request) (*http.Response, error) {
	observation, _ := req.Context().Value(sendObservationKey{}).(*sendObservation)
	if observation == nil {
		return c.client.Do(req)
	}
	if admit, ok := req.Context().Value(sendAdmissionKey{}).(func(context.Context) (func(), error)); ok {
		release, err := admit(req.Context())
		if err != nil {
			if req.Body != nil {
				_ = req.Body.Close()
			}
			return nil, err
		}
		if release != nil {
			var once sync.Once
			observation.releaseAdmission = func() { once.Do(release) }
			defer observation.releaseAdmission()
		}
	}
	if timeout, ok := req.Context().Value(sendTimeoutKey{}).(time.Duration); ok && timeout > 0 {
		ctx, cancel := context.WithTimeout(req.Context(), timeout)
		defer cancel() // sendTransport buffers the complete response before returning.
		req = req.WithContext(ctx)
	}
	return c.client.Do(req)
}

type sendTransport struct{ base http.RoundTripper }

func (t *sendTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	observation, _ := req.Context().Value(sendObservationKey{}).(*sendObservation)
	if observation == nil || !strings.HasSuffix(req.URL.Path, "/sendMessage") {
		return t.base.RoundTrip(req)
	}
	observation.once.Do(func() {
		observation.started = true
		if observation.callback != nil {
			observation.callback(time.Now())
		}
	})
	if observation.releaseAdmission != nil {
		observation.releaseAdmission()
	}
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	// Observe the complete envelope before the SDK reduces nonstandard provider errors to text.
	// Reinstall exactly the same bytes for the SDK's normal result/error decoding.
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	_ = resp.Body.Close()
	if readErr != nil {
		return nil, readErr
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	var envelope struct {
		OK *bool `json:"ok"`
	}
	var response telegramAPIResponse
	if json.Unmarshal(body, &envelope) == nil && envelope.OK != nil && json.Unmarshal(body, &response) == nil {
		observation.response = &response
	}
	return resp, nil
}

func classifySendError(err error, observation *sendObservation) error {
	result := &SendError{Kind: "failed", Code: "not_started", Err: sanitizeTelegramRequestError(err)}
	if observation.started {
		result.Kind = "unknown"
		result.Code = "response_lost"
	}
	if response := observation.response; response != nil && !response.OK && response.ErrorCode > 0 {
		result.Kind = "failed"
		result.Code = "provider_" + strconv.Itoa(response.ErrorCode)
		result.Err = telegramAPIResponseError(*response)
		if response.ErrorCode == 429 || response.ErrorCode >= 500 {
			result.Kind = "retryable"
		}
		result.RetryAfter = time.Duration(response.Parameters.RetryAfter) * time.Second
		if telegramRecipientUnreachableDescription(response.Description) || response.ErrorCode == 403 || response.ErrorCode == 404 {
			result.Code = "recipient_unreachable"
		}
	}
	return result
}
