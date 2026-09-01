package worm

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	DefaultWebMarketPositionPollInterval = time.Second
	DefaultWebMarketPositionPollTimeout  = 30 * time.Second
)

type WebMarketPositionSigner interface {
	WebSignInSigner
	WebPositionTransactionSigner
}

type WebPositionTransactionSigningRequest struct {
	WalletAddress     string
	PositionRequestID int64
	TransactionHex    string
	TransactionSHA256 [sha256.Size]byte
}

type WebPositionTransactionSigningResponse struct {
	Payload WebPositionFinalizePayload
}

type WebMarketPositionSubmitRequest struct {
	WalletAddress     string
	MarketConditionID string
	Funds             string
	IsYes             bool
}

type WebMarketPositionSubmitOptions struct {
	PollInterval time.Duration
	PollTimeout  time.Duration
}

type WebMarketPositionSubmitStage string

const (
	WebMarketPositionSubmitStageAuthenticating WebMarketPositionSubmitStage = "AUTHENTICATING"
	WebMarketPositionSubmitStageOpening        WebMarketPositionSubmitStage = "OPENING"
	WebMarketPositionSubmitStageSigning        WebMarketPositionSubmitStage = "SIGNING"
	WebMarketPositionSubmitStageFinalizing     WebMarketPositionSubmitStage = "FINALIZING"
	WebMarketPositionSubmitStageObserving      WebMarketPositionSubmitStage = "OBSERVING"
)

type WebMutationOutcome string

const (
	WebMutationOutcomeNotDispatched WebMutationOutcome = "NOT_DISPATCHED"
	WebMutationOutcomeAcknowledged  WebMutationOutcome = "ACKNOWLEDGED"
	WebMutationOutcomeRejected      WebMutationOutcome = "REJECTED"
	WebMutationOutcomeUnknown       WebMutationOutcome = "OUTCOME_UNKNOWN"
)

type WebMarketPositionSubmitStatus string

const (
	WebMarketPositionSubmitStatusAccepted               WebMarketPositionSubmitStatus = "WEB_REQUEST_ACCEPTED"
	WebMarketPositionSubmitStatusPending                WebMarketPositionSubmitStatus = "WEB_REQUEST_PENDING"
	WebMarketPositionSubmitStatusProviderFailed         WebMarketPositionSubmitStatus = "PROVIDER_FAILED"
	WebMarketPositionSubmitStatusOpenRejected           WebMarketPositionSubmitStatus = "OPEN_REJECTED"
	WebMarketPositionSubmitStatusOpenOutcomeUnknown     WebMarketPositionSubmitStatus = "OPEN_OUTCOME_UNKNOWN"
	WebMarketPositionSubmitStatusOpenedNotFinalized     WebMarketPositionSubmitStatus = "OPENED_NOT_FINALIZED"
	WebMarketPositionSubmitStatusFinalizeRejected       WebMarketPositionSubmitStatus = "FINALIZE_REJECTED"
	WebMarketPositionSubmitStatusFinalizeOutcomeUnknown WebMarketPositionSubmitStatus = "FINALIZE_OUTCOME_UNKNOWN"
)

type WebMarketPositionSubmitResult struct {
	Status          WebMarketPositionSubmitStatus
	Stage           WebMarketPositionSubmitStage
	OpenOutcome     WebMutationOutcome
	FinalizeOutcome WebMutationOutcome

	PositionRequestID  int64
	ProviderState      string
	ProviderOrderState string
	FundingTxID        *string
	RefundTxID         *string

	FinalizeMode       WebFinalizeMode
	TransactionSHA256  []byte
	TransactionVersion string
	RequiredSignatures int
	WalletSignerIndex  int
	SignerPublicKey    string
}

type WebMarketPositionSubmitError struct {
	Stage     WebMarketPositionSubmitStage
	Status    WebMarketPositionSubmitStatus
	RequestID int64
	err       error
}

func (e *WebMarketPositionSubmitError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf(
		"worm web market position submit failed: stage=%s status=%s request_id=%d",
		e.Stage,
		e.Status,
		e.RequestID,
	)
}

func (e *WebMarketPositionSubmitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func SubmitWebMarketPosition(
	ctx context.Context,
	client WebMarketPositionSubmitClient,
	signer WebMarketPositionSigner,
	request WebMarketPositionSubmitRequest,
	options WebMarketPositionSubmitOptions,
) (*WebMarketPositionSubmitResult, error) {
	resolvedOptions, err := resolveWebMarketPositionSubmitOptions(options)
	if err != nil {
		return nil, newWebMarketPositionSubmitError(
			WebMarketPositionSubmitStageAuthenticating,
			"",
			0,
			err,
		)
	}
	if client == nil {
		return nil, newWebMarketPositionSubmitError(
			WebMarketPositionSubmitStageAuthenticating,
			"",
			0,
			errors.New("Worm Web client is required"),
		)
	}
	if signer == nil {
		return nil, newWebMarketPositionSubmitError(
			WebMarketPositionSubmitStageAuthenticating,
			"",
			0,
			errors.New("Worm Web market position signer is required"),
		)
	}
	openCommand, err := PrepareWebMarketPositionOpen(request)
	if err != nil {
		return nil, newWebMarketPositionSubmitError(WebMarketPositionSubmitStageOpening, "", 0, err)
	}

	session, err := AuthenticateWebWallet(ctx, client, signer, request.WalletAddress)
	if err != nil {
		return nil, newWebMarketPositionSubmitError(WebMarketPositionSubmitStageAuthenticating, "", 0, err)
	}

	result := &WebMarketPositionSubmitResult{
		Stage:           WebMarketPositionSubmitStageOpening,
		OpenOutcome:     WebMutationOutcomeUnknown,
		FinalizeOutcome: WebMutationOutcomeNotDispatched,
	}
	opened, err := DispatchWebMarketPositionOpen(ctx, client, session, openCommand)
	applyWebPositionObservation(result, opened)
	if err != nil {
		if webMutationExplicitlyRejected(err) {
			result.Status = WebMarketPositionSubmitStatusOpenRejected
			result.OpenOutcome = WebMutationOutcomeRejected
		} else {
			result.Status = WebMarketPositionSubmitStatusOpenOutcomeUnknown
		}
		return result, newWebMarketPositionSubmitError(
			result.Stage,
			result.Status,
			result.PositionRequestID,
			err,
		)
	}
	result.OpenOutcome = WebMutationOutcomeAcknowledged
	if opened.IsTerminalFailure() {
		result.Status = WebMarketPositionSubmitStatusProviderFailed
		return result, newWebMarketPositionSubmitError(
			result.Stage,
			result.Status,
			result.PositionRequestID,
			errors.New("Worm Web position request reached terminal failure"),
		)
	}
	if opened.IsCompleted() {
		result.Status = WebMarketPositionSubmitStatusAccepted
		result.Stage = WebMarketPositionSubmitStageObserving
		return result, nil
	}

	result.Stage = WebMarketPositionSubmitStageSigning
	metadata, err := InspectWebPositionRequestTransaction(opened)
	if err != nil {
		result.Status = WebMarketPositionSubmitStatusOpenedNotFinalized
		return result, newWebMarketPositionSubmitError(result.Stage, result.Status, result.PositionRequestID, err)
	}
	result.TransactionSHA256 = append([]byte(nil), metadata.TransactionSHA256[:]...)
	result.TransactionVersion = metadata.TransactionVersion
	result.RequiredSignatures = metadata.RequiredSignatures
	result.WalletSignerIndex = metadata.WalletSignerIndex
	result.SignerPublicKey = metadata.SignerPublicKey

	finalizeCommand, err := PrepareWebPositionFinalize(
		ctx,
		signer,
		opened,
		metadata.TransactionSHA256,
	)
	if err != nil {
		result.Status = WebMarketPositionSubmitStatusOpenedNotFinalized
		return result, newWebMarketPositionSubmitError(result.Stage, result.Status, result.PositionRequestID, err)
	}
	result.FinalizeMode = finalizeCommand.FinalizeMode()

	result.Stage = WebMarketPositionSubmitStageFinalizing
	result.FinalizeOutcome = WebMutationOutcomeUnknown
	finalized, finalizeErr := DispatchWebPositionFinalize(ctx, client, session, finalizeCommand)
	applyWebPositionObservation(result, finalized)
	fallbackStatus := WebMarketPositionSubmitStatusPending
	var fallbackErr error
	if finalizeErr != nil {
		if webMutationExplicitlyRejected(finalizeErr) {
			result.FinalizeOutcome = WebMutationOutcomeRejected
			fallbackStatus = WebMarketPositionSubmitStatusFinalizeRejected
			fallbackErr = finalizeErr
		} else {
			fallbackStatus = WebMarketPositionSubmitStatusFinalizeOutcomeUnknown
			fallbackErr = finalizeErr
		}
	} else {
		result.FinalizeOutcome = WebMutationOutcomeAcknowledged
		if finalized.IsTerminalFailure() {
			fallbackStatus = WebMarketPositionSubmitStatusProviderFailed
			fallbackErr = errors.New("Worm Web position request reached terminal failure")
		} else if finalized.IsAccepted() {
			fallbackStatus = WebMarketPositionSubmitStatusAccepted
		}
	}

	result.Stage = WebMarketPositionSubmitStageObserving
	return observeWebMarketPositionSubmission(
		ctx,
		client,
		session,
		result,
		resolvedOptions,
		fallbackStatus,
		fallbackErr,
	)
}

func resolveWebMarketPositionSubmitOptions(
	options WebMarketPositionSubmitOptions,
) (WebMarketPositionSubmitOptions, error) {
	if options.PollInterval < 0 {
		return WebMarketPositionSubmitOptions{}, errors.New("Worm Web poll interval must not be negative")
	}
	if options.PollTimeout < 0 {
		return WebMarketPositionSubmitOptions{}, errors.New("Worm Web poll timeout must not be negative")
	}
	if options.PollInterval == 0 {
		options.PollInterval = DefaultWebMarketPositionPollInterval
	}
	if options.PollTimeout == 0 {
		options.PollTimeout = DefaultWebMarketPositionPollTimeout
	}
	return options, nil
}

func observeWebMarketPositionSubmission(
	ctx context.Context,
	client WebMarketPositionSubmitClient,
	session *WebAuthenticatedSession,
	result *WebMarketPositionSubmitResult,
	options WebMarketPositionSubmitOptions,
	fallbackStatus WebMarketPositionSubmitStatus,
	fallbackErr error,
) (*WebMarketPositionSubmitResult, error) {
	pollContext, cancel := context.WithTimeout(ctx, options.PollTimeout)
	defer cancel()

	var lastObservationErr error
	observedNonTerminal := false
	for {
		observation, err := ObserveWebPositionRequest(
			pollContext,
			client,
			session,
			result.PositionRequestID,
		)
		applyWebPositionObservation(result, observation)
		if err == nil {
			lastObservationErr = nil
			observedNonTerminal = true
			if observation.IsAccepted() {
				result.Status = WebMarketPositionSubmitStatusAccepted
				return result, nil
			}
			if observation.IsTerminalFailure() {
				result.Status = WebMarketPositionSubmitStatusProviderFailed
				return result, newWebMarketPositionSubmitError(
					result.Stage,
					result.Status,
					result.PositionRequestID,
					errors.New("Worm Web position request reached terminal failure"),
				)
			}
		}
		if err != nil {
			lastObservationErr = err
			if !webPositionStatusRetryable(err) {
				return finishWebMarketPositionObservation(
					result,
					fallbackStatus,
					fallbackErr,
					err,
					observedNonTerminal,
				)
			}
		}

		timer := time.NewTimer(options.PollInterval)
		select {
		case <-pollContext.Done():
			timer.Stop()
			return finishWebMarketPositionObservation(
				result,
				fallbackStatus,
				fallbackErr,
				errors.Join(lastObservationErr, pollContext.Err()),
				observedNonTerminal,
			)
		case <-timer.C:
		}
	}
}

func finishWebMarketPositionObservation(
	result *WebMarketPositionSubmitResult,
	fallbackStatus WebMarketPositionSubmitStatus,
	fallbackErr error,
	observationErr error,
	observedNonTerminal bool,
) (*WebMarketPositionSubmitResult, error) {
	if result.FinalizeOutcome == WebMutationOutcomeAcknowledged &&
		observedNonTerminal &&
		fallbackStatus != WebMarketPositionSubmitStatusProviderFailed {
		result.Status = WebMarketPositionSubmitStatusPending
		return result, nil
	}
	if result.FinalizeOutcome == WebMutationOutcomeAcknowledged && !observedNonTerminal {
		switch fallbackStatus {
		case WebMarketPositionSubmitStatusAccepted:
			result.Status = WebMarketPositionSubmitStatusAccepted
			return result, nil
		case WebMarketPositionSubmitStatusPending:
			result.Status = WebMarketPositionSubmitStatusPending
			return result, nil
		}
	}

	result.Status = fallbackStatus
	if result.Status == WebMarketPositionSubmitStatusAccepted ||
		result.Status == WebMarketPositionSubmitStatusPending {
		return result, nil
	}
	return result, newWebMarketPositionSubmitError(
		result.Stage,
		result.Status,
		result.PositionRequestID,
		errors.Join(fallbackErr, observationErr),
	)
}

func applyWebPositionObservation(
	result *WebMarketPositionSubmitResult,
	observation *WebPositionRequestObservation,
) {
	if result == nil || observation == nil {
		return
	}
	if result.PositionRequestID > 0 && observation.PositionRequestID != result.PositionRequestID {
		return
	}
	if observation.PositionRequestID > 0 {
		result.PositionRequestID = observation.PositionRequestID
	}
	result.ProviderState = observation.ProviderState
	result.ProviderOrderState = observation.ProviderOrderState
	if observation.FundingTxID != nil {
		fundingTxID := *observation.FundingTxID
		result.FundingTxID = &fundingTxID
	}
	if observation.RefundTxID != nil {
		refundTxID := *observation.RefundTxID
		result.RefundTxID = &refundTxID
	}
}

func normalizedWebProviderState(value string) string {
	return strings.ToLower(normalizedWebDiagnostic(value, 100))
}

func normalizedWebDiagnostic(value string, maximumBytes int) string {
	value = strings.ToValidUTF8(value, "")
	value = strings.Map(func(candidate rune) rune {
		if unicode.IsControl(candidate) {
			return ' '
		}
		return candidate
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= maximumBytes {
		return value
	}
	boundary := maximumBytes
	for boundary > 0 && !utf8.RuneStart(value[boundary]) {
		boundary--
	}
	return value[:boundary]
}

func validateWebSignInNonce(nonce string) error {
	if nonce == "" || strings.TrimSpace(nonce) != nonce {
		return errors.New("Worm Web sign-in challenge returned a non-canonical nonce")
	}
	if !utf8.ValidString(nonce) || len(nonce) > 128 {
		return errors.New("Worm Web sign-in challenge returned an invalid nonce")
	}
	for _, candidate := range nonce {
		if unicode.IsControl(candidate) {
			return errors.New("Worm Web sign-in challenge returned an invalid nonce")
		}
	}
	return nil
}

func webMutationExplicitlyRejected(err error) bool {
	var edge *WebEdgeBlockedError
	if errors.As(err, &edge) {
		return true
	}
	var api *WebAPIError
	if !errors.As(err, &api) {
		return false
	}
	if !api.StructuredJSON {
		return false
	}
	switch api.StatusCode {
	case http.StatusRequestTimeout,
		http.StatusConflict,
		http.StatusTooEarly,
		http.StatusTooManyRequests:
		return false
	}
	return api.StatusCode >= http.StatusOK && api.StatusCode < http.StatusMultipleChoices ||
		api.StatusCode >= http.StatusBadRequest && api.StatusCode < http.StatusInternalServerError
}

func webPositionStatusRetryable(err error) bool {
	var api *WebAPIError
	if errors.As(err, &api) {
		switch api.StatusCode {
		case http.StatusNotFound,
			http.StatusRequestTimeout,
			http.StatusConflict,
			http.StatusTooEarly,
			http.StatusTooManyRequests:
			return true
		default:
			return api.StatusCode >= http.StatusInternalServerError
		}
	}
	var transport *WebTransportError
	if errors.As(err, &transport) {
		return true
	}
	var response *WebResponseError
	return errors.As(err, &response)
}

func newWebMarketPositionSubmitError(
	stage WebMarketPositionSubmitStage,
	status WebMarketPositionSubmitStatus,
	requestID int64,
	err error,
) error {
	return &WebMarketPositionSubmitError{
		Stage:     stage,
		Status:    status,
		RequestID: requestID,
		err:       err,
	}
}
