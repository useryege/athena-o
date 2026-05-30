package pipeline

import (
	"context"
	"errors"
)

type ProjectPipeline struct {
	lifecycles []Lifecycle
}

func NewProjectPipeline(lifecycles ...Lifecycle) *ProjectPipeline {
	return &ProjectPipeline{lifecycles: lifecycles}
}

func (p *ProjectPipeline) Start(ctx context.Context) error {
	if p == nil {
		return nil
	}
	started := make([]Lifecycle, 0, len(p.lifecycles))
	for _, lifecycle := range p.lifecycles {
		if lifecycle == nil {
			continue
		}
		if err := lifecycle.Start(ctx); err != nil {
			for i := len(started) - 1; i >= 0; i-- {
				_ = started[i].Stop()
			}
			return err
		}
		started = append(started, lifecycle)
	}
	return nil
}

func (p *ProjectPipeline) Stop() error {
	if p == nil {
		return nil
	}
	errs := make([]error, 0, len(p.lifecycles))
	for i := len(p.lifecycles) - 1; i >= 0; i-- {
		if p.lifecycles[i] == nil {
			continue
		}
		errs = append(errs, p.lifecycles[i].Stop())
	}
	return errors.Join(errs...)
}
