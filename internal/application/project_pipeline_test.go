package application

import (
	"context"
	"errors"
	"testing"
)

type lifecycleStub struct {
	startErr error
	stopErr  error

	startCalls int
	stopCalls  int
}

func (l *lifecycleStub) Start(context.Context) error {
	l.startCalls++
	return l.startErr
}

func (l *lifecycleStub) Stop() error {
	l.stopCalls++
	return l.stopErr
}

func TestProjectPipelineStartOrderAndStopRollback(t *testing.T) {
	discovery := &lifecycleStub{}
	reconciler := &lifecycleStub{startErr: errors.New("start failed")}
	pipeline := NewProjectPipeline(discovery, reconciler)

	err := pipeline.Start(context.Background())
	if err == nil {
		t.Fatalf("expected start error")
	}
	if discovery.startCalls != 1 {
		t.Fatalf("discovery start calls = %d, want 1", discovery.startCalls)
	}
	if reconciler.startCalls != 1 {
		t.Fatalf("reconciler start calls = %d, want 1", reconciler.startCalls)
	}
	if discovery.stopCalls != 1 {
		t.Fatalf("discovery stop calls = %d, want 1", discovery.stopCalls)
	}
}

func TestProjectPipelineStopStopsBoth(t *testing.T) {
	discovery := &lifecycleStub{}
	reconciler := &lifecycleStub{}
	pipeline := NewProjectPipeline(discovery, reconciler)

	if err := pipeline.Stop(); err != nil {
		t.Fatalf("stop pipeline: %v", err)
	}
	if discovery.stopCalls != 1 {
		t.Fatalf("discovery stop calls = %d, want 1", discovery.stopCalls)
	}
	if reconciler.stopCalls != 1 {
		t.Fatalf("reconciler stop calls = %d, want 1", reconciler.stopCalls)
	}
}
