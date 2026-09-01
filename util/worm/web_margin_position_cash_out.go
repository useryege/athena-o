package worm

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	DefaultWebMarginPositionCashOutPollInterval = time.Second
	DefaultWebMarginPositionCashOutPollTimeout  = 90 * time.Second
)

type WebMarginPositionCashOutRequest struct {
	WalletAddress     string
	MarketConditionID string
	PositionID        int64
	IsYes             bool
}

type WebMarginPositionCashOutOptions struct {
	PollInterval time.Duration
	PollTimeout  time.Duration
}

type WebMarginPositionCashOutStage string

const (
	WebMarginPositionCashOutStageAuthenticating WebMarginPositionCashOutStage = "AUTHENTICATING"
	WebMarginPositionCashOutStageLocating       WebMarginPositionCashOutStage = "LOCATING"
	WebMarginPositionCashOutStageClosing        WebMarginPositionCashOutStage = "CLOSING"
	WebMarginPositionCashOutStageObserving      WebMarginPositionCashOutStage = "OBSERVING"
)

type WebMarginPositionCashOutStatus string

const (
	WebMarginPositionCashOutStatusClosed              WebMarginPositionCashOutStatus = "POSITION_CLOSED"
	WebMarginPositionCashOutStatusPending             WebMarginPositionCashOutStatus = "POSITION_CLOSE_PENDING"
	WebMarginPositionCashOutStatusNotFound            WebMarginPositionCashOutStatus = "POSITION_NOT_FOUND"
	WebMarginPositionCashOutStatusMismatch            WebMarginPositionCashOutStatus = "POSITION_MISMATCH"
	WebMarginPositionCashOutStatusNotClosable         WebMarginPositionCashOutStatus = "POSITION_NOT_CLOSABLE"
	WebMarginPositionCashOutStatusLookupFailed        WebMarginPositionCashOutStatus = "POSITION_LOOKUP_FAILED"
	WebMarginPositionCashOutStatusCloseRejected       WebMarginPositionCashOutStatus = "CLOSE_REJECTED"
	WebMarginPositionCashOutStatusCloseOutcomeUnknown WebMarginPositionCashOutStatus = "CLOSE_OUTCOME_UNKNOWN"
)

type WebMarginPositionCashOutResult struct {
	Status       WebMarginPositionCashOutStatus
	Stage        WebMarginPositionCashOutStage
	CloseOutcome WebMutationOutcome

	PositionID        int64
	MarketConditionID string
	IsYes             bool
	ProviderState     string
	IsClosed          bool
	IsLiquidated      bool
}

type WebMarginPositionCashOutError struct {
	Stage      WebMarginPositionCashOutStage
	Status     WebMarginPositionCashOutStatus
	PositionID int64
	err        error
}

func (e *WebMarginPositionCashOutError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf(
		"worm web margin position cash out failed: stage=%s status=%s position_id=%d",
		e.Stage,
		e.Status,
		e.PositionID,
	)
}

func (e *WebMarginPositionCashOutError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func CashOutWebMarginPosition(
	ctx context.Context,
	client WebMarginPositionCashOutClient,
	signer WebSignInSigner,
	request WebMarginPositionCashOutRequest,
	options WebMarginPositionCashOutOptions,
) (*WebMarginPositionCashOutResult, error) {
	resolvedOptions, err := resolveWebMarginPositionCashOutOptions(options)
	if err != nil {
		return nil, newWebMarginPositionCashOutError(
			WebMarginPositionCashOutStageAuthenticating,
			"",
			request.PositionID,
			err,
		)
	}
	if client == nil {
		return nil, newWebMarginPositionCashOutError(
			WebMarginPositionCashOutStageAuthenticating,
			"",
			request.PositionID,
			errors.New("Worm Web cash-out client is required"),
		)
	}
	if signer == nil {
		return nil, newWebMarginPositionCashOutError(
			WebMarginPositionCashOutStageAuthenticating,
			"",
			request.PositionID,
			errors.New("Worm Web sign-in signer is required"),
		)
	}
	if _, err := canonicalWebSolanaPublicKey(request.WalletAddress, "wallet address"); err != nil {
		return nil, newWebMarginPositionCashOutError(
			WebMarginPositionCashOutStageAuthenticating,
			"",
			request.PositionID,
			err,
		)
	}
	if _, err := canonicalWebSolanaPublicKey(request.MarketConditionID, "market condition id"); err != nil {
		return nil, newWebMarginPositionCashOutError(
			WebMarginPositionCashOutStageLocating,
			"",
			request.PositionID,
			err,
		)
	}
	if request.PositionID <= 0 {
		return nil, newWebMarginPositionCashOutError(
			WebMarginPositionCashOutStageLocating,
			"",
			request.PositionID,
			errors.New("Worm Web margin position id must be positive"),
		)
	}

	session, err := AuthenticateWebWallet(ctx, client, signer, request.WalletAddress)
	if err != nil {
		return nil, newWebMarginPositionCashOutError(
			WebMarginPositionCashOutStageAuthenticating,
			"",
			request.PositionID,
			err,
		)
	}

	result := &WebMarginPositionCashOutResult{
		Stage:             WebMarginPositionCashOutStageLocating,
		CloseOutcome:      WebMutationOutcomeNotDispatched,
		PositionID:        request.PositionID,
		MarketConditionID: request.MarketConditionID,
		IsYes:             request.IsYes,
	}
	position, err := getExactWebMarginPosition(ctx, client, session.accessToken, request)
	if err != nil {
		status := WebMarginPositionCashOutStatusLookupFailed
		switch {
		case errors.Is(err, errWebMarginPositionNotFound):
			status = WebMarginPositionCashOutStatusNotFound
		case errors.Is(err, errWebMarginPositionMismatch):
			status = WebMarginPositionCashOutStatusMismatch
		}
		result.Status = status
		return result, newWebMarginPositionCashOutError(result.Stage, result.Status, result.PositionID, err)
	}
	applyWebMarginPositionCashOutResult(result, position)

	switch classifyWebMarginPosition(position) {
	case webMarginPositionDispositionClosed:
		result.Status = WebMarginPositionCashOutStatusClosed
		return result, nil
	case webMarginPositionDispositionClosing:
		result.Stage = WebMarginPositionCashOutStageObserving
		return observeWebMarginPositionCashOut(
			ctx,
			client,
			session.accessToken,
			request,
			result,
			resolvedOptions,
			true,
			nil,
		)
	case webMarginPositionDispositionLiquidated, webMarginPositionDispositionUnknown:
		result.Status = WebMarginPositionCashOutStatusNotClosable
		return result, newWebMarginPositionCashOutError(
			result.Stage,
			result.Status,
			result.PositionID,
			errors.New("Worm Web margin position is not safely closable"),
		)
	case webMarginPositionDispositionOpen:
	}

	result.Stage = WebMarginPositionCashOutStageClosing
	result.CloseOutcome = WebMutationOutcomeUnknown
	closeErr := client.CloseMarginPosition(ctx, session.accessToken, WebMarginPositionCloseRequest{
		MarketConditionID: request.MarketConditionID,
		IsYes:             request.IsYes,
		PositionID:        request.PositionID,
	})
	if closeErr == nil {
		result.CloseOutcome = WebMutationOutcomeAcknowledged
	} else if webMutationExplicitlyRejected(closeErr) {
		result.CloseOutcome = WebMutationOutcomeRejected
	}

	result.Stage = WebMarginPositionCashOutStageObserving
	return observeWebMarginPositionCashOut(
		ctx,
		client,
		session.accessToken,
		request,
		result,
		resolvedOptions,
		false,
		closeErr,
	)
}

func resolveWebMarginPositionCashOutOptions(
	options WebMarginPositionCashOutOptions,
) (WebMarginPositionCashOutOptions, error) {
	if options.PollInterval < 0 {
		return WebMarginPositionCashOutOptions{}, errors.New("Worm Web cash-out poll interval must not be negative")
	}
	if options.PollTimeout < 0 {
		return WebMarginPositionCashOutOptions{}, errors.New("Worm Web cash-out poll timeout must not be negative")
	}
	if options.PollInterval == 0 {
		options.PollInterval = DefaultWebMarginPositionCashOutPollInterval
	}
	if options.PollTimeout == 0 {
		options.PollTimeout = DefaultWebMarginPositionCashOutPollTimeout
	}
	return options, nil
}

var (
	errWebMarginPositionNotFound  = errors.New("Worm Web margin position was not found")
	errWebMarginPositionMismatch  = errors.New("Worm Web margin position identity does not match the cash-out target")
	errWebMarginPositionMalformed = errors.New("Worm Web margin position response is malformed")
)

func getExactWebMarginPosition(
	ctx context.Context,
	client WebMarginPositionCashOutClient,
	accessToken string,
	request WebMarginPositionCashOutRequest,
) (*WebMarginPosition, error) {
	positions, err := client.ListMarginPositions(ctx, accessToken, WebMarginPositionListOptions{
		MarketConditionID: request.MarketConditionID,
	})
	if err != nil {
		return nil, err
	}

	var matched *WebMarginPosition
	for index := range positions {
		position := &positions[index]
		if position.PositionID <= 0 {
			return nil, errWebMarginPositionMalformed
		}
		if int64(position.PositionID) != request.PositionID {
			continue
		}
		if matched != nil {
			return nil, errWebMarginPositionMismatch
		}
		matched = position
	}
	if matched == nil {
		return nil, errWebMarginPositionNotFound
	}
	if matched.Market == nil || matched.Market.ConditionID != request.MarketConditionID ||
		matched.IsYes == nil || *matched.IsYes != request.IsYes {
		return nil, errWebMarginPositionMismatch
	}
	return matched, nil
}

type webMarginPositionDisposition uint8

const (
	webMarginPositionDispositionUnknown webMarginPositionDisposition = iota
	webMarginPositionDispositionOpen
	webMarginPositionDispositionClosing
	webMarginPositionDispositionClosed
	webMarginPositionDispositionLiquidated
)

func classifyWebMarginPosition(position *WebMarginPosition) webMarginPositionDisposition {
	if position == nil {
		return webMarginPositionDispositionUnknown
	}
	state := WebMarginPositionState(normalizedWebProviderState(string(position.State)))
	if position.IsLiquidated != nil && *position.IsLiquidated || state == WebMarginPositionStateLiquidated {
		return webMarginPositionDispositionLiquidated
	}
	if state == WebMarginPositionStateClosing {
		return webMarginPositionDispositionClosing
	}
	if position.IsClosed != nil && *position.IsClosed || state == WebMarginPositionStateClosed {
		return webMarginPositionDispositionClosed
	}
	if state == WebMarginPositionStateOpen &&
		position.IsClosed != nil && !*position.IsClosed &&
		position.IsLiquidated != nil && !*position.IsLiquidated {
		return webMarginPositionDispositionOpen
	}
	return webMarginPositionDispositionUnknown
}

func applyWebMarginPositionCashOutResult(
	result *WebMarginPositionCashOutResult,
	position *WebMarginPosition,
) {
	if result == nil || position == nil {
		return
	}
	result.ProviderState = normalizedWebProviderState(string(position.State))
	if position.IsClosed != nil {
		result.IsClosed = *position.IsClosed
	}
	if position.IsLiquidated != nil {
		result.IsLiquidated = *position.IsLiquidated
	}
}

func observeWebMarginPositionCashOut(
	ctx context.Context,
	client WebMarginPositionCashOutClient,
	accessToken string,
	request WebMarginPositionCashOutRequest,
	result *WebMarginPositionCashOutResult,
	options WebMarginPositionCashOutOptions,
	observedClosing bool,
	closeErr error,
) (*WebMarginPositionCashOutResult, error) {
	pollContext, cancel := context.WithTimeout(ctx, options.PollTimeout)
	defer cancel()

	var lastObservationErr error
	for {
		position, observationErr := getExactWebMarginPosition(pollContext, client, accessToken, request)
		if observationErr == nil {
			applyWebMarginPositionCashOutResult(result, position)
			switch classifyWebMarginPosition(position) {
			case webMarginPositionDispositionClosed:
				result.Status = WebMarginPositionCashOutStatusClosed
				return result, nil
			case webMarginPositionDispositionLiquidated:
				result.Status = WebMarginPositionCashOutStatusNotClosable
				return result, newWebMarginPositionCashOutError(
					result.Stage,
					result.Status,
					result.PositionID,
					errors.New("Worm Web margin position was liquidated"),
				)
			case webMarginPositionDispositionClosing:
				observedClosing = true
				lastObservationErr = nil
			case webMarginPositionDispositionOpen:
				lastObservationErr = nil
				if result.CloseOutcome == WebMutationOutcomeRejected {
					result.Status = WebMarginPositionCashOutStatusCloseRejected
					return result, newWebMarginPositionCashOutError(
						result.Stage,
						result.Status,
						result.PositionID,
						closeErr,
					)
				}
			case webMarginPositionDispositionUnknown:
				lastObservationErr = errors.New("Worm Web margin position returned an unknown lifecycle state")
			}
		} else {
			lastObservationErr = observationErr
			if errors.Is(observationErr, errWebMarginPositionMismatch) ||
				errors.Is(observationErr, errWebMarginPositionMalformed) {
				return finishWebMarginPositionCashOutObservation(
					result,
					observedClosing,
					closeErr,
					observationErr,
				)
			}
			if !errors.Is(observationErr, errWebMarginPositionNotFound) &&
				!webPositionStatusRetryable(observationErr) {
				return finishWebMarginPositionCashOutObservation(
					result,
					observedClosing,
					closeErr,
					observationErr,
				)
			}
		}

		timer := time.NewTimer(options.PollInterval)
		select {
		case <-pollContext.Done():
			timer.Stop()
			return finishWebMarginPositionCashOutObservation(
				result,
				observedClosing,
				closeErr,
				errors.Join(lastObservationErr, pollContext.Err()),
			)
		case <-timer.C:
		}
	}
}

func finishWebMarginPositionCashOutObservation(
	result *WebMarginPositionCashOutResult,
	observedClosing bool,
	closeErr error,
	observationErr error,
) (*WebMarginPositionCashOutResult, error) {
	if errors.Is(observationErr, errWebMarginPositionMismatch) {
		switch result.CloseOutcome {
		case WebMutationOutcomeNotDispatched:
			result.Status = WebMarginPositionCashOutStatusMismatch
		case WebMutationOutcomeUnknown:
			result.Status = WebMarginPositionCashOutStatusCloseOutcomeUnknown
		default:
			result.Status = WebMarginPositionCashOutStatusLookupFailed
		}
		return result, newWebMarginPositionCashOutError(
			result.Stage,
			result.Status,
			result.PositionID,
			errors.Join(closeErr, observationErr),
		)
	}
	if errors.Is(observationErr, errWebMarginPositionMalformed) {
		if result.CloseOutcome == WebMutationOutcomeUnknown {
			result.Status = WebMarginPositionCashOutStatusCloseOutcomeUnknown
		} else {
			result.Status = WebMarginPositionCashOutStatusLookupFailed
		}
		return result, newWebMarginPositionCashOutError(
			result.Stage,
			result.Status,
			result.PositionID,
			errors.Join(closeErr, observationErr),
		)
	}
	if observedClosing || result.CloseOutcome == WebMutationOutcomeAcknowledged {
		result.Status = WebMarginPositionCashOutStatusPending
		return result, nil
	}

	switch result.CloseOutcome {
	case WebMutationOutcomeRejected:
		// A rejected mutation is reported as CLOSE_REJECTED only when a
		// successful reconciliation observed that the target remained open.
		result.Status = WebMarginPositionCashOutStatusLookupFailed
	case WebMutationOutcomeUnknown:
		result.Status = WebMarginPositionCashOutStatusCloseOutcomeUnknown
	case WebMutationOutcomeNotDispatched:
		result.Status = WebMarginPositionCashOutStatusLookupFailed
	default:
		result.Status = WebMarginPositionCashOutStatusCloseOutcomeUnknown
	}
	return result, newWebMarginPositionCashOutError(
		result.Stage,
		result.Status,
		result.PositionID,
		errors.Join(closeErr, observationErr),
	)
}

func newWebMarginPositionCashOutError(
	stage WebMarginPositionCashOutStage,
	status WebMarginPositionCashOutStatus,
	positionID int64,
	err error,
) error {
	return &WebMarginPositionCashOutError{
		Stage:      stage,
		Status:     status,
		PositionID: positionID,
		err:        err,
	}
}
