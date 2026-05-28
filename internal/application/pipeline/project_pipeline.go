package pipeline

import (
	"context"
	"errors"
)

type ProjectPipeline struct {
	discoveryIndexer ProjectDiscoveryIndexer
	stateReconciler  ProjectStateReconciler
}

func NewProjectPipeline(discoveryIndexer ProjectDiscoveryIndexer, stateReconciler ProjectStateReconciler) *ProjectPipeline {
	return &ProjectPipeline{
		discoveryIndexer: discoveryIndexer,
		stateReconciler:  stateReconciler,
	}
}

func (p *ProjectPipeline) Start(ctx context.Context) error {
	if p == nil {
		return nil
	}
	if p.discoveryIndexer != nil {
		if err := p.discoveryIndexer.Start(ctx); err != nil {
			return err
		}
	}
	if p.stateReconciler != nil {
		if err := p.stateReconciler.Start(ctx); err != nil {
			if p.discoveryIndexer != nil {
				_ = p.discoveryIndexer.Stop()
			}
			return err
		}
	}
	return nil
}

func (p *ProjectPipeline) Stop() error {
	if p == nil {
		return nil
	}
	var reconcilerErr error
	if p.stateReconciler != nil {
		reconcilerErr = p.stateReconciler.Stop()
	}
	var discoveryErr error
	if p.discoveryIndexer != nil {
		discoveryErr = p.discoveryIndexer.Stop()
	}
	return errors.Join(discoveryErr, reconcilerErr)
}
