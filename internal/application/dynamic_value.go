package application

import "time"

type DynamicValue[T comparable] struct {
	Value T

	Initialized bool

	LastCheckedAt time.Time
	LastChangedAt time.Time
	UpdatedAt     time.Time

	LastError string
}

func (v *DynamicValue[T]) Update(newValue T, now time.Time) bool {
	v.LastCheckedAt = now

	if !v.Initialized {
		v.Value = newValue
		v.Initialized = true
		v.UpdatedAt = now
		v.LastChangedAt = now
		v.LastError = ""
		return true
	}

	if v.Value == newValue {
		v.UpdatedAt = now
		v.LastError = ""
		return false
	}

	v.Value = newValue
	v.UpdatedAt = now
	v.LastChangedAt = now
	v.LastError = ""
	return true
}
