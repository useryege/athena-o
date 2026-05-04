package application

type StaticFieldQueue interface {
}

type staticFieldQueueImpl struct {
}

func NewStaticFieldQueue() StaticFieldQueue {
	return &staticFieldQueueImpl{}
}
