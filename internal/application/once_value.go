package application

import "time"

type OnceValue[T any] struct {
	Value      T
	ResolvedAt time.Time
}

func (v *OnceValue[T]) Get() T {
	return v.Value
}

func (v *OnceValue[T]) Set(value T) {
	v.Value = value
	v.ResolvedAt = time.Now()
}

func (v *OnceValue[T]) IsReady() bool {
	return !v.ResolvedAt.IsZero()
}
