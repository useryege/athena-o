package application

import (
	"sync"
	"time"
)

type FieldStatus string

const (
	FieldStatusUnknown FieldStatus = "unknown"
	FieldStatusFailed  FieldStatus = "failed"
	FieldStatusReady   FieldStatus = "ready"
)

type FieldValue[T any] struct {
	mu         sync.RWMutex
	value      T
	status     FieldStatus
	resolvedAt time.Time
	updatedAt  time.Time
	lastError  string
}

func (v *FieldValue[T]) IsReady() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.status == FieldStatusReady
}

func (v *FieldValue[T]) IsFailed() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.status == FieldStatusFailed
}

func (v *FieldValue[T]) IsUnknown() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.status == "" || v.status == FieldStatusUnknown
}

func (v *FieldValue[T]) Get() (T, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.status != FieldStatusReady {
		var zero T
		return zero, false
	}

	return v.value, true
}

func (v *FieldValue[T]) Status() FieldStatus {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.status == "" {
		return FieldStatusUnknown
	}
	return v.status
}

func (v *FieldValue[T]) ResolvedAt() time.Time {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.resolvedAt
}

func (v *FieldValue[T]) UpdatedAt() time.Time {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.updatedAt
}

func (v *FieldValue[T]) LastError() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.lastError
}

func (v *FieldValue[T]) MarkReady(value T, now time.Time) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.value = value
	v.status = FieldStatusReady
	if v.resolvedAt.IsZero() {
		v.resolvedAt = now
	}
	v.updatedAt = now
	v.lastError = ""
}

func (v *FieldValue[T]) MarkFailed(err error, now time.Time) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.status = FieldStatusFailed
	v.updatedAt = now
	if err != nil {
		v.lastError = err.Error()
		return
	}
	v.lastError = ""
}

func (v *FieldValue[T]) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()

	var zero T
	v.value = zero
	v.status = FieldStatusUnknown
	v.resolvedAt = time.Time{}
	v.updatedAt = time.Time{}
	v.lastError = ""
}
