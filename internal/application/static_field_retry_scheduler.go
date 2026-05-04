package application

type StaticFieldRetryScheduler interface {
}

type staticFieldRetrySchedulerImpl struct {
}

func NewStaticFieldRetryScheduler() StaticFieldRetryScheduler {
	return &staticFieldRetrySchedulerImpl{}
}
