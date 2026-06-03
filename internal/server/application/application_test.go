package application

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	applicationapiclient "github.com/useryege/athena/internal/application/apiclient"
	"github.com/useryege/athena/internal/server/rbacpolicy"
	applicationpkg "github.com/useryege/athena/pkg/apiclient/application"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/assets"
	utilio "github.com/useryege/athena/util/io"
	"github.com/useryege/athena/util/rbac"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func newTestApplicationServer(client *fakeApplicationServiceClient) *Server {
	enf := rbac.NewEnforcer(nil)
	policyEnf := rbacpolicy.NewRBACPolicyEnforcer(enf)
	enf.SetClaimsEnforcerFunc(policyEnf.EnforceClaims)
	if err := enf.SetBuiltinPolicy(assets.BuiltinPolicyCSV); err != nil {
		panic(err)
	}
	return NewServer(&fakeApplicationClientset{client: client}, enf)
}

func contextWithSubject(subject string) context.Context {
	return context.WithValue(context.Background(), "claims", jwt.MapClaims{"sub": subject})
}

func TestGetProjectDiscoveryStatusProxiesToApplication(t *testing.T) {
	client := &fakeApplicationServiceClient{}
	resp, err := newTestApplicationServer(client).GetProjectDiscoveryStatus(context.Background(), &applicationpkg.GetProjectDiscoveryStatusRequest{})
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
	resp, err := newTestApplicationServer(client).StartProjectDiscovery(contextWithSubject("admin"), &applicationpkg.StartProjectDiscoveryRequest{})
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
	resp, err := newTestApplicationServer(client).StopProjectDiscovery(contextWithSubject("admin"), &applicationpkg.StopProjectDiscoveryRequest{})
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

func TestProjectDiscoveryControlRequiresRBACPermission(t *testing.T) {
	client := &fakeApplicationServiceClient{}
	server := newTestApplicationServer(client)

	if _, err := server.StartProjectDiscovery(contextWithSubject("viewer"), &applicationpkg.StartProjectDiscoveryRequest{}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("StartProjectDiscovery error = %v, want PermissionDenied", err)
	}
	if _, err := server.StopProjectDiscovery(contextWithSubject("viewer"), &applicationpkg.StopProjectDiscoveryRequest{}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("StopProjectDiscovery error = %v, want PermissionDenied", err)
	}
	if client.startCalls != 0 || client.stopCalls != 0 {
		t.Fatalf("internal client calls start=%d stop=%d, want none", client.startCalls, client.stopCalls)
	}
}
