package application

import (
	"context"
	"testing"

	applicationapiclient "github.com/useryege/athena/internal/application/apiclient"
	applicationpkg "github.com/useryege/athena/pkg/apiclient/application"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilio "github.com/useryege/athena/util/io"
	"google.golang.org/grpc"
)

type fakeApplicationClientset struct {
	client *fakeApplicationServiceClient
}

func (f *fakeApplicationClientset) NewApplicationServiceClient() (utilio.Closer, applicationapiclient.ApplicationServiceClient, error) {
	return utilio.NopCloser, f.client, nil
}

type fakeApplicationServiceClient struct {
	applicationapiclient.ApplicationServiceClient

	statusCalls int
	startCalls  int
	stopCalls   int
}

func (f *fakeApplicationServiceClient) GetProjectDiscoveryStatus(context.Context, *applicationapiclient.GetProjectDiscoveryStatusRequest, ...grpc.CallOption) (*v1alpha1.ProjectDiscoveryStatus, error) {
	f.statusCalls++
	return &v1alpha1.ProjectDiscoveryStatus{Started: true, Status: "running"}, nil
}

func (f *fakeApplicationServiceClient) StartProjectDiscovery(context.Context, *applicationapiclient.StartProjectDiscoveryRequest, ...grpc.CallOption) (*v1alpha1.ProjectDiscoveryStatus, error) {
	f.startCalls++
	return &v1alpha1.ProjectDiscoveryStatus{Started: true, Status: "running"}, nil
}

func (f *fakeApplicationServiceClient) StopProjectDiscovery(context.Context, *applicationapiclient.StopProjectDiscoveryRequest, ...grpc.CallOption) (*v1alpha1.ProjectDiscoveryStatus, error) {
	f.stopCalls++
	return &v1alpha1.ProjectDiscoveryStatus{Started: false, Status: "stopped"}, nil
}

func TestGetProjectDiscoveryStatusProxiesToApplication(t *testing.T) {
	client := &fakeApplicationServiceClient{}
	resp, err := NewServer(&fakeApplicationClientset{client: client}).GetProjectDiscoveryStatus(context.Background(), &applicationpkg.GetProjectDiscoveryStatusRequest{})
	if err != nil {
		t.Fatalf("GetProjectDiscoveryStatus: %v", err)
	}
	if client.statusCalls != 1 {
		t.Fatalf("status calls = %d, want 1", client.statusCalls)
	}
	if !resp.Started || resp.Status != "running" {
		t.Fatalf("status = %#v, want running", resp)
	}
}

func TestStartProjectDiscoveryProxiesToApplication(t *testing.T) {
	client := &fakeApplicationServiceClient{}
	resp, err := NewServer(&fakeApplicationClientset{client: client}).StartProjectDiscovery(context.Background(), &applicationpkg.StartProjectDiscoveryRequest{})
	if err != nil {
		t.Fatalf("StartProjectDiscovery: %v", err)
	}
	if client.startCalls != 1 {
		t.Fatalf("start calls = %d, want 1", client.startCalls)
	}
	if !resp.Started || resp.Status != "running" {
		t.Fatalf("status = %#v, want running", resp)
	}
}

func TestStopProjectDiscoveryProxiesToApplication(t *testing.T) {
	client := &fakeApplicationServiceClient{}
	resp, err := NewServer(&fakeApplicationClientset{client: client}).StopProjectDiscovery(context.Background(), &applicationpkg.StopProjectDiscoveryRequest{})
	if err != nil {
		t.Fatalf("StopProjectDiscovery: %v", err)
	}
	if client.stopCalls != 1 {
		t.Fatalf("stop calls = %d, want 1", client.stopCalls)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status = %#v, want stopped", resp)
	}
}
