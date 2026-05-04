package application

import (
	"errors"
	"time"
)

type StaticValueStatus string

const (
	StaticValueUnknown StaticValueStatus = "unknown"
	StaticValuePending StaticValueStatus = "pending"
	StaticValueReady   StaticValueStatus = "ready"
	StaticValueFailed  StaticValueStatus = "failed"
)

type StaticValue[T any] struct {
	Value T

	Status StaticValueStatus

	AttemptCount  int
	LastAttemptAt time.Time
	NextAttemptAt time.Time
	ResolvedAt    time.Time

	LastError string
	Source    string
}

func (v *StaticValue[T]) IsReady() bool {
	return v.Status == StaticValueReady
}

func (v *StaticValue[T]) ShouldAttempt(now time.Time) bool {
	if v.Status == StaticValueReady {
		return false
	}
	if v.Status == StaticValueFailed && v.AttemptCount > 0 && v.NextAttemptAt.IsZero() {
		return false
	}

	if v.NextAttemptAt.IsZero() {
		return true
	}

	return !now.Before(v.NextAttemptAt)
}

func (v *StaticValue[T]) MarkAttempt(now time.Time) {
	if v.Status == StaticValueReady {
		return
	}

	v.Status = StaticValuePending
	v.LastAttemptAt = now
}

func (v *StaticValue[T]) MarkReady(value T, source string, now time.Time) {
	v.Value = value
	v.Status = StaticValueReady
	v.Source = source
	v.ResolvedAt = now
	v.LastAttemptAt = now
	v.LastError = ""
	v.NextAttemptAt = time.Time{}
}

func (v *StaticValue[T]) MarkFailed(cause error, policy StaticFieldPolicy, now time.Time) {
	if cause == nil {
		cause = errors.New("unknown static field error")
	}

	v.Status = StaticValueFailed
	v.AttemptCount++
	v.LastAttemptAt = now
	v.LastError = cause.Error()

	delay := policy.BaseDelay
	if delay < 0 {
		delay = 0
	}
	v.NextAttemptAt = now.Add(delay)
}
