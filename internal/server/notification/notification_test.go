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

func (f *fakeNotificationServiceClient) SendNotification(context.Context, *notificationapiclient.SendNotificationRequest, ...grpc.CallOption) (*notificationapiclient.SendNotificationResponse, error) {
	return &notificationapiclient.SendNotificationResponse{}, nil
}

func (f *fakeNotificationServiceClient) ListNotificationDeliveries(_ context.Context, req *notificationapiclient.ListNotificationDeliveriesRequest, _ ...grpc.CallOption) (*notificationapiclient.ListNotificationDeliveriesResponse, error) {
	return &notificationapiclient.ListNotificationDeliveriesResponse{
		Items: []*v1alpha1.NotificationDeliveryItem{
			{
				ID:       7,
				Source:   req.GetSource(),
				Severity: req.GetSeverity(),
				Status:   req.GetStatus(),
				Title:    req.GetKeyword(),
			},
		},
		Total:    1,
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	}, nil
}

func (f *fakeNotificationServiceClient) GetNotificationDelivery(_ context.Context, req *notificationapiclient.GetNotificationDeliveryRequest, _ ...grpc.CallOption) (*notificationapiclient.GetNotificationDeliveryResponse, error) {
	return &notificationapiclient.GetNotificationDeliveryResponse{
		Item: &v1alpha1.NotificationDeliveryDetail{ID: req.GetId(), Source: "worm", Status: "sent"},
	}, nil
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

func TestListNotificationDeliveriesMapsRequestAndResponse(t *testing.T) {
	resp, err := NewServer(&fakeNotificationClientset{client: &fakeNotificationServiceClient{}}).ListNotificationDeliveries(context.Background(), &notificationpkg.ListNotificationDeliveriesRequest{
		Page:     2,
		PageSize: 20,
		Status:   "sent",
		Severity: "warning",
		Source:   "worm",
		Keyword:  "scan",
	})
	if err != nil {
		t.Fatalf("ListNotificationDeliveries: %v", err)
	}
	if resp.Total != 1 || resp.Page != 2 || resp.PageSize != 20 {
		t.Fatalf("pagination = %#v, want total/page/pageSize mapped", resp)
	}
	if len(resp.Items) != 1 || resp.Items[0].Source != "worm" || resp.Items[0].Title != "scan" {
		t.Fatalf("items = %#v, want mapped filters in fake response", resp.Items)
	}
}

func TestGetNotificationDeliveryMapsResponse(t *testing.T) {
	resp, err := NewServer(&fakeNotificationClientset{client: &fakeNotificationServiceClient{}}).GetNotificationDelivery(context.Background(), &notificationpkg.GetNotificationDeliveryRequest{Id: 7})
	if err != nil {
		t.Fatalf("GetNotificationDelivery: %v", err)
	}
	if resp.GetItem().ID != 7 || resp.GetItem().Status != "sent" {
		t.Fatalf("item = %#v, want mapped detail", resp.GetItem())
	}
}
