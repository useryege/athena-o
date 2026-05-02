package application

import "time"

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

	AttemptCount  int       // number of attempts
	LastAttemptAt time.Time // last attempt at
	NextAttemptAt time.Time // next attempt at
	ResolvedAt    time.Time // successfully resolved at

	LastError string // last error
	Source    string // source of the value
}

func NewStaticValue[T any](value T) *StaticValue[T] {
	return &StaticValue[T]{
		Value:  value,
		Status: StaticValueUnknown,
	}
}
