package notification

import (
	"context"
	"errors"
	"strings"
	"testing"

	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	notificationpkg "github.com/useryege/athena/pkg/apiclient/notification"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilio "github.com/useryege/athena/util/io"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeNotificationClientset struct {
	client *fakeNotificationServiceClient
}

func (f *fakeNotificationClientset) NewNotificationServiceClient() (utilio.Closer, notificationapiclient.NotificationServiceClient, error) {
	return utilio.NopCloser, f.client, nil
}

type fakeNotificationServiceClient struct {
	sendReq  *notificationapiclient.SendNotificationRequest
	sendResp *notificationapiclient.SendNotificationResponse
	sendErr  error
}

func (f *fakeNotificationServiceClient) GetNotificationStatus(context.Context, *notificationapiclient.GetNotificationStatusRequest, ...grpc.CallOption) (*v1alpha1.NotificationStatus, error) {
	return &v1alpha1.NotificationStatus{Started: true, Status: "running"}, nil
}

func (f *fakeNotificationServiceClient) SendNotification(_ context.Context, req *notificationapiclient.SendNotificationRequest, _ ...grpc.CallOption) (*notificationapiclient.SendNotificationResponse, error) {
	f.sendReq = req
	if f.sendErr != nil {
		return nil, f.sendErr
	}
	if f.sendResp != nil {
		return f.sendResp, nil
	}
	return &notificationapiclient.SendNotificationResponse{}, nil
}

func (f *fakeNotificationServiceClient) ListNotificationDeliveries(_ context.Context, req *notificationapiclient.ListNotificationDeliveriesRequest, _ ...grpc.CallOption) (*notificationapiclient.ListNotificationDeliveriesResponse, error) {
	return &notificationapiclient.ListNotificationDeliveriesResponse{
		Items: []*v1alpha1.NotificationDeliveryItem{
			{
				ID:       7,
				Source:   req.GetSource(),
				Severity: req.GetSeverity(),
				Topic:    req.GetTopic(),
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
		Topic:    "poly",
		Source:   "worm",
		Keyword:  "scan",
	})
	if err != nil {
		t.Fatalf("ListNotificationDeliveries: %v", err)
	}
	if resp.Total != 1 || resp.Page != 2 || resp.PageSize != 20 {
		t.Fatalf("pagination = %#v, want total/page/pageSize mapped", resp)
	}
	if len(resp.Items) != 1 || resp.Items[0].Source != "worm" || resp.Items[0].Topic != "poly" || resp.Items[0].Title != "scan" {
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

func TestSendTestNotificationMapsFixedRequestAndResponse(t *testing.T) {
	client := &fakeNotificationServiceClient{
		sendResp: &notificationapiclient.SendNotificationResponse{
			NotificationId:    11,
			Status:            notificationapiclient.NotificationDeliveryStatus_NOTIFICATION_DELIVERY_STATUS_SENT,
			ProviderMessageId: "123",
		},
	}
	resp, err := NewServer(&fakeNotificationClientset{client: client}).SendTestNotification(context.Background(), &notificationpkg.SendTestNotificationRequest{Topic: "token"})
	if err != nil {
		t.Fatalf("SendTestNotification: %v", err)
	}
	if resp.NotificationId != 11 || resp.Status != "sent" || resp.ProviderMessageId != "123" {
		t.Fatalf("response = %#v, want mapped sent response", resp)
	}
	if client.sendReq == nil {
		t.Fatalf("send request was not captured")
	}
	if client.sendReq.Source != "ui" {
		t.Fatalf("source = %q, want ui", client.sendReq.Source)
	}
	if client.sendReq.Severity != notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_INFO {
		t.Fatalf("severity = %s, want info", client.sendReq.Severity)
	}
	if client.sendReq.Topic != notificationapiclient.NotificationTopic_NOTIFICATION_TOPIC_TOKEN {
		t.Fatalf("topic = %s, want token", client.sendReq.Topic)
	}
	if client.sendReq.Title != "ATHENA TOKEN test notification" {
		t.Fatalf("title = %q, want fixed test title", client.sendReq.Title)
	}
	if !strings.HasPrefix(client.sendReq.Body, "Manual TOKEN test notification sent from ATHENA UI at ") {
		t.Fatalf("body = %q, want fixed test body prefix", client.sendReq.Body)
	}
}

func TestSendTestNotificationMapsPolyTopic(t *testing.T) {
	client := &fakeNotificationServiceClient{}
	_, err := NewServer(&fakeNotificationClientset{client: client}).SendTestNotification(context.Background(), &notificationpkg.SendTestNotificationRequest{Topic: "poly"})
	if err != nil {
		t.Fatalf("SendTestNotification: %v", err)
	}
	if client.sendReq == nil || client.sendReq.Topic != notificationapiclient.NotificationTopic_NOTIFICATION_TOPIC_POLY {
		t.Fatalf("send request = %#v, want poly topic", client.sendReq)
	}
	if client.sendReq.Title != "ATHENA POLY test notification" {
		t.Fatalf("title = %q, want POLY title", client.sendReq.Title)
	}
}

func TestSendTestNotificationRejectsInvalidTopic(t *testing.T) {
	client := &fakeNotificationServiceClient{}
	_, err := NewServer(&fakeNotificationClientset{client: client}).SendTestNotification(context.Background(), &notificationpkg.SendTestNotificationRequest{Topic: "bad"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("error = %v, want InvalidArgument", err)
	}
	if client.sendReq != nil {
		t.Fatalf("send request = %#v, want no internal send", client.sendReq)
	}
}

func TestSendTestNotificationReturnsClientError(t *testing.T) {
	client := &fakeNotificationServiceClient{sendErr: errors.New("notification unavailable")}
	_, err := NewServer(&fakeNotificationClientset{client: client}).SendTestNotification(context.Background(), &notificationpkg.SendTestNotificationRequest{Topic: "token"})
	if err == nil || err.Error() != "notification unavailable" {
		t.Fatalf("error = %v, want internal client error", err)
	}
}
