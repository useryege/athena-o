package application

import (
	"time"
)

type RetryUntilReadyValueStatus string

const (
	RetryUntilReadyValueStatusUnknown RetryUntilReadyValueStatus = "unknown"
	RetryUntilReadyValueStatusPending RetryUntilReadyValueStatus = "pending"
	RetryUntilReadyValueStatusFailed  RetryUntilReadyValueStatus = "failed"
	RetryUntilReadyValueStatusReady   RetryUntilReadyValueStatus = "ready"
)

type RetryUntilReadyValue[T any] struct {
	Value  T
	Status RetryUntilReadyValueStatus

	AttemptCount  int
	LastAttemptAt time.Time
	ResolvedAt    time.Time

	LastError string
	Retryable bool
}

func (v *RetryUntilReadyValue[T]) IsReady() bool {
	return v.Status == RetryUntilReadyValueStatusReady
}
func (v *RetryUntilReadyValue[T]) IsPending() bool {
	return v.Status == RetryUntilReadyValueStatusPending
}
func (v *RetryUntilReadyValue[T]) IsFailed() bool {
	return v.Status == RetryUntilReadyValueStatusFailed
}
func (v *RetryUntilReadyValue[T]) IsUnknown() bool {
	return v.Status == RetryUntilReadyValueStatusUnknown
}

func (v *RetryUntilReadyValue[T]) ShouldAttempt() bool {
	if v.Status == RetryUntilReadyValueStatusReady {
		return false
	}

	if v.Status == RetryUntilReadyValueStatusFailed && !v.Retryable {
		return false
	}

	return true
}

func (v *RetryUntilReadyValue[T]) MarkAttempt(now time.Time) {
	if v.Status == RetryUntilReadyValueStatusReady {
		return
	}

	v.Status = RetryUntilReadyValueStatusPending
	v.AttemptCount++
	v.LastAttemptAt = now
}

func (v *RetryUntilReadyValue[T]) MarkReady(value T, now time.Time) {
	v.Value = value
	v.Status = RetryUntilReadyValueStatusReady
	v.ResolvedAt = now
	v.LastAttemptAt = now
	v.LastError = ""
	v.Retryable = false
}

func (v *RetryUntilReadyValue[T]) MarkFailed(
	err error,
	now time.Time,
	retryable bool,
) {
	v.Status = RetryUntilReadyValueStatusFailed
	v.LastAttemptAt = now
	v.Retryable = retryable

	if err != nil {
		v.LastError = err.Error()
	} else {
		v.LastError = ""
	}

	if !retryable {
		return
	}
}

func (v *RetryUntilReadyValue[T]) MarkNotAvailable(
	now time.Time,
) {
	v.Status = RetryUntilReadyValueStatusFailed
	v.LastAttemptAt = now
	v.LastError = "not available yet"
	v.Retryable = true
}

func (v *RetryUntilReadyValue[T]) MarkTerminalFailed(err error, now time.Time) {
	v.Status = RetryUntilReadyValueStatusFailed
	v.LastAttemptAt = now
	v.Retryable = false

	if err != nil {
		v.LastError = err.Error()
	}
}

func (v *RetryUntilReadyValue[T]) Get() (T, bool) {
	if v.Status != RetryUntilReadyValueStatusReady {
		var zero T
		return zero, false
	}

	return v.Value, true
}

func (v *RetryUntilReadyValue[T]) Reset() {
	var zero T

	v.Value = zero
	v.Status = RetryUntilReadyValueStatusUnknown
	v.AttemptCount = 0
	v.LastAttemptAt = time.Time{}
	v.ResolvedAt = time.Time{}
	v.LastError = ""
	v.Retryable = true
}
