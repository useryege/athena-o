package api

import (
	"context"
	"testing"

	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	"github.com/useryege/athena/internal/application/pipeline"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type projectDiscoveryIndexerFake struct {
	startCalls int
	stopCalls  int
}

func (f *projectDiscoveryIndexerFake) Start(context.Context) error {
	f.startCalls++
	return nil
}

func (f *projectDiscoveryIndexerFake) Stop() error {
	f.stopCalls++
	return nil
}

func TestProjectDiscoveryStatusDefaultsToStopped(t *testing.T) {
	service := &Service{
		started:      true,
		lifecycleCtx: context.Background(),
	}

	resp, err := service.GetProjectDiscoveryStatus(context.Background(), &applicationpkg.GetProjectDiscoveryStatusRequest{})
	if err != nil {
		t.Fatalf("GetProjectDiscoveryStatus: %v", err)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status = %#v, want stopped", resp)
	}
}

func TestStartProjectDiscoveryRequiresStartedApplication(t *testing.T) {
	service := &Service{}

	_, err := service.StartProjectDiscovery(context.Background(), &applicationpkg.StartProjectDiscoveryRequest{})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("StartProjectDiscovery error = %v, want FailedPrecondition", err)
	}
}

func TestProjectDiscoveryStartStopLifecycle(t *testing.T) {
	lifecycleCtx, lifecycleStop := context.WithCancel(context.Background())
	defer lifecycleStop()

	indexer := &projectDiscoveryIndexerFake{}
	service := &Service{
		started:       true,
		lifecycleCtx:  lifecycleCtx,
		lifecycleStop: lifecycleStop,
		discoveryIndexerFactory: func() (pipeline.ProjectDiscoveryIndexer, error) {
			return indexer, nil
		},
	}

	resp, err := service.StartProjectDiscovery(context.Background(), &applicationpkg.StartProjectDiscoveryRequest{})
	if err != nil {
		t.Fatalf("StartProjectDiscovery: %v", err)
	}
	if !resp.Started || resp.Status != "running" {
		t.Fatalf("status after start = %#v, want running", resp)
	}
	if indexer.startCalls != 1 {
		t.Fatalf("start calls = %d, want 1", indexer.startCalls)
	}

	if _, err := service.StartProjectDiscovery(context.Background(), &applicationpkg.StartProjectDiscoveryRequest{}); err != nil {
		t.Fatalf("second StartProjectDiscovery: %v", err)
	}
	if indexer.startCalls != 1 {
		t.Fatalf("start calls after second start = %d, want 1", indexer.startCalls)
	}

	resp, err = service.StopProjectDiscovery(context.Background(), &applicationpkg.StopProjectDiscoveryRequest{})
	if err != nil {
		t.Fatalf("StopProjectDiscovery: %v", err)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status after stop = %#v, want stopped", resp)
	}
	if indexer.stopCalls != 1 {
		t.Fatalf("stop calls = %d, want 1", indexer.stopCalls)
	}
}

func TestServiceStopStopsProjectDiscovery(t *testing.T) {
	lifecycleCtx, lifecycleStop := context.WithCancel(context.Background())
	indexer := &projectDiscoveryIndexerFake{}
	service := &Service{
		started:          true,
		lifecycleCtx:     lifecycleCtx,
		lifecycleStop:    lifecycleStop,
		discoveryIndexer: indexer,
		discoveryStop:    func() {},
		discoveryStarted: true,
	}

	if err := service.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if indexer.stopCalls != 1 {
		t.Fatalf("discovery stop calls = %d, want 1", indexer.stopCalls)
	}
	resp, err := service.GetProjectDiscoveryStatus(context.Background(), &applicationpkg.GetProjectDiscoveryStatusRequest{})
	if err != nil {
		t.Fatalf("GetProjectDiscoveryStatus: %v", err)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status after service stop = %#v, want stopped", resp)
	}
}
