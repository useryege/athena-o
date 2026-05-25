package notification

import (
	"context"
	"testing"

	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	notificationpkg "github.com/useryege/athena/pkg/apiclient/notification"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilio "github.com/useryege/athena/util/io"
	"google.golang.org/grpc"
)

type fakeNotificationClientset struct {
	client *fakeNotificationServiceClient
}

func (f *fakeNotificationClientset) NewNotificationServiceClient() (utilio.Closer, notificationapiclient.NotificationServiceClient, error) {
	return utilio.NopCloser, f.client, nil
}

type fakeNotificationServiceClient struct{}

func (f *fakeNotificationServiceClient) GetNotificationStatus(context.Context, *notificationapiclient.GetNotificationStatusRequest, ...grpc.CallOption) (*v1alpha1.NotificationStatus, error) {
	return &v1alpha1.NotificationStatus{Started: true, Status: "running"}, nil
}

func TestGetNotificationStatusMapsResponse(t *testing.T) {
	resp, err := NewServer(&fakeNotificationClientset{client: &fakeNotificationServiceClient{}}).GetNotificationStatus(context.Background(), &notificationpkg.GetNotificationStatusRequest{})
	if err != nil {
		t.Fatalf("GetNotificationStatus: %v", err)
	}
	if !resp.Started || resp.Status != "running" {
		t.Fatalf("status = %#v, want running", resp)
	}
}
