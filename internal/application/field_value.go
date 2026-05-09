package application

import "time"

type FieldStatus string

const (
	FieldStatusUnknown FieldStatus = "unknown"
	FieldStatusFailed  FieldStatus = "failed"
	FieldStatusReady   FieldStatus = "ready"
)

type FieldValue[T any] struct {
	Value      T
	Status     FieldStatus
	ResolvedAt time.Time
	LastError  string
}

func (v *FieldValue[T]) IsReady() bool {
	return v.Status == FieldStatusReady
}

func (v *FieldValue[T]) IsFailed() bool {
	return v.Status == FieldStatusFailed
}

func (v *FieldValue[T]) IsUnknown() bool {
	return v.Status == "" || v.Status == FieldStatusUnknown
}

func (v *FieldValue[T]) Get() (T, bool) {
	if v.Status != FieldStatusReady {
		var zero T
		return zero, false
	}

	return v.Value, true
}

func (v *FieldValue[T]) MarkReady(value T, now time.Time) {
	v.Value = value
	v.Status = FieldStatusReady
	v.ResolvedAt = now
	v.LastError = ""
}

func (v *FieldValue[T]) MarkFailed(err error, now time.Time) {
	v.Status = FieldStatusFailed
	v.ResolvedAt = now
	if err != nil {
		v.LastError = err.Error()
		return
	}
	v.LastError = ""
}

func (v *FieldValue[T]) Reset() {
	var zero T
	v.Value = zero
	v.Status = FieldStatusUnknown
	v.ResolvedAt = time.Time{}
	v.LastError = ""
}
