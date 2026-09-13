package tradersync

import (
	"context"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	"github.com/useryege/athena/internal/tradersync/transport"
	api "github.com/useryege/athena/pkg/apiclient/tradersync"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"reflect"
	"testing"
)

var facadeActor = &trpc.Actor{AccountId: "11111111-1111-4111-8111-111111111111", Realm: trpc.ApplicationRealm_APPLICATION_REALM_ADMIN}

type recordingInternal struct {
	trpc.UnimplementedTraderSyncServiceServer
	request  any
	response any
	err      error
}

func (s *recordingInternal) ResolveTarget(ctx context.Context, r *trpc.ResolveTargetRequest) (*trpc.ResolveTargetResponse, error) {
	s.request = r
	if s.err != nil {
		return nil, s.err
	}
	return s.response.(*trpc.ResolveTargetResponse), nil
}
func (s *recordingInternal) CreateSubscription(ctx context.Context, r *trpc.CreateSubscriptionRequest) (*trpc.CreateSubscriptionResponse, error) {
	s.request = r
	if s.err != nil {
		return nil, s.err
	}
	return s.response.(*trpc.CreateSubscriptionResponse), nil
}
func (s *recordingInternal) ListSubscriptions(ctx context.Context, r *trpc.ListSubscriptionsRequest) (*trpc.ListSubscriptionsResponse, error) {
	s.request = r
	if s.err != nil {
		return nil, s.err
	}
	return s.response.(*trpc.ListSubscriptionsResponse), nil
}
func (s *recordingInternal) GetSubscription(ctx context.Context, r *trpc.GetSubscriptionRequest) (*trpc.GetSubscriptionResponse, error) {
	s.request = r
	if s.err != nil {
		return nil, s.err
	}
	return s.response.(*trpc.GetSubscriptionResponse), nil
}
func (s *recordingInternal) PauseSubscription(ctx context.Context, r *trpc.PauseSubscriptionRequest) (*trpc.PauseSubscriptionResponse, error) {
	s.request = r
	if s.err != nil {
		return nil, s.err
	}
	return s.response.(*trpc.PauseSubscriptionResponse), nil
}
func (s *recordingInternal) ResumeSubscription(ctx context.Context, r *trpc.ResumeSubscriptionRequest) (*trpc.ResumeSubscriptionResponse, error) {
	s.request = r
	if s.err != nil {
		return nil, s.err
	}
	return s.response.(*trpc.ResumeSubscriptionResponse), nil
}
func (s *recordingInternal) CancelSubscription(ctx context.Context, r *trpc.CancelSubscriptionRequest) (*trpc.CancelSubscriptionResponse, error) {
	s.request = r
	if s.err != nil {
		return nil, s.err
	}
	return s.response.(*trpc.CancelSubscriptionResponse), nil
}
func (s *recordingInternal) UpdateTargetNote(ctx context.Context, r *trpc.UpdateTargetNoteRequest) (*trpc.UpdateTargetNoteResponse, error) {
	s.request = r
	if s.err != nil {
		return nil, s.err
	}
	return s.response.(*trpc.UpdateTargetNoteResponse), nil
}
func (s *recordingInternal) ListActivities(ctx context.Context, r *trpc.ListActivitiesRequest) (*trpc.ListActivitiesResponse, error) {
	s.request = r
	if s.err != nil {
		return nil, s.err
	}
	return s.response.(*trpc.ListActivitiesResponse), nil
}
func (s *recordingInternal) GetActivity(ctx context.Context, r *trpc.GetActivityRequest) (*trpc.GetActivityResponse, error) {
	s.request = r
	if s.err != nil {
		return nil, s.err
	}
	return s.response.(*trpc.GetActivityResponse), nil
}
func (s *recordingInternal) ListSubscriptionHistory(ctx context.Context, r *trpc.ListSubscriptionHistoryRequest) (*trpc.ListSubscriptionHistoryResponse, error) {
	s.request = r
	if s.err != nil {
		return nil, s.err
	}
	return s.response.(*trpc.ListSubscriptionHistoryResponse), nil
}
func (s *recordingInternal) GetSummaryBatch(ctx context.Context, r *trpc.GetSummaryBatchRequest) (*trpc.GetSummaryBatchResponse, error) {
	s.request = r
	if s.err != nil {
		return nil, s.err
	}
	return s.response.(*trpc.GetSummaryBatchResponse), nil
}
func (s *recordingInternal) ListSummaryParts(ctx context.Context, r *trpc.ListSummaryPartsRequest) (*trpc.ListSummaryPartsResponse, error) {
	s.request = r
	if s.err != nil {
		return nil, s.err
	}
	return s.response.(*trpc.ListSummaryPartsResponse), nil
}
func (s *recordingInternal) ListSubscriptionSummaries(ctx context.Context, r *trpc.ListSubscriptionSummariesRequest) (*trpc.ListSubscriptionSummariesResponse, error) {
	s.request = r
	if s.err != nil {
		return nil, s.err
	}
	return s.response.(*trpc.ListSubscriptionSummariesResponse), nil
}
func (s *recordingInternal) GetSubscriptionSummary(ctx context.Context, r *trpc.GetSubscriptionSummaryRequest) (*trpc.GetSubscriptionSummaryResponse, error) {
	s.request = r
	if s.err != nil {
		return nil, s.err
	}
	return s.response.(*trpc.GetSubscriptionSummaryResponse), nil
}
func (s *recordingInternal) GetTraderSyncRuntimeStatus(ctx context.Context, r *trpc.GetTraderSyncRuntimeStatusRequest) (*trpc.GetTraderSyncRuntimeStatusResponse, error) {
	s.request = r
	if s.err != nil {
		return nil, s.err
	}
	return s.response.(*trpc.GetTraderSyncRuntimeStatusResponse), nil
}
func internalConnection(t *testing.T, s trpc.TraderSyncServiceServer) *grpc.ClientConn {
	t.Helper()
	l := bufconn.Listen(1024 * 1024)
	g := grpc.NewServer()
	trpc.RegisterTraderSyncServiceServer(g, s)
	go g.Serve(l)
	t.Cleanup(g.Stop)
	c, e := grpc.DialContext(context.Background(), "passthrough:///internal", grpc.WithInsecure(), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return l.Dial() }))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

// Fill complete wire fixtures; individual expectations below are literal request facts.
// This fixture is not used to derive expected public values.
func fillWire(v reflect.Value) {
	if v.Kind() == reflect.Pointer {
		v.Set(reflect.New(v.Type().Elem()))
		fillWire(v.Elem())
		return
	}
	if v.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		switch f.Kind() {
		case reflect.Pointer:
			fillWire(f)
		case reflect.Slice:
			if f.Type().Elem().Kind() == reflect.Pointer {
				f.Set(reflect.MakeSlice(f.Type(), 1, 1))
				fillWire(f.Index(0))
			}
		}
	}
}
func completeWire[T any]() *T { v := new(T); fillWire(reflect.ValueOf(v).Elem()); return v }
func TestFacadeAllMethodsForwardExactRequests(t *testing.T) {
	t.Run("ResolveTarget", func(t *testing.T) {
		remote := &recordingInternal{response: completeWire[trpc.ResolveTargetResponse]()}
		fillSentinels(reflect.ValueOf(remote.response), "ResolveTarget")
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.ResolveTarget(context.Background(), &api.ResolveTargetRequest{Input: "  unchanged input  "})
		if e != nil || got == nil {
			t.Fatalf("response: %v %v", got, e)
		}
		compareWireFields(t, reflect.ValueOf(remote.response), reflect.ValueOf(got), "response")
		want := &trpc.ResolveTargetRequest{Actor: facadeActor, Input: "  unchanged input  "}
		if !reflect.DeepEqual(remote.request, want) {
			t.Fatalf("request got %#v want %#v", remote.request, want)
		}
	})
	t.Run("CreateSubscription", func(t *testing.T) {
		remote := &recordingInternal{response: completeWire[trpc.CreateSubscriptionResponse]()}
		fillSentinels(reflect.ValueOf(remote.response), "CreateSubscription")
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.CreateSubscription(context.Background(), &api.CreateSubscriptionRequest{ConfirmationToken: "opaque-token", RequestId: "intent", Note: &api.TraderSyncNoteInput{Value: ""}})
		if e != nil || got == nil {
			t.Fatalf("response: %v %v", got, e)
		}
		compareWireFields(t, reflect.ValueOf(remote.response), reflect.ValueOf(got), "response")
		want := &trpc.CreateSubscriptionRequest{Actor: facadeActor, ConfirmationToken: "opaque-token", RequestId: "intent", Note: &trpc.NoteInput{Value: ""}}
		if !reflect.DeepEqual(remote.request, want) {
			t.Fatalf("request got %#v want %#v", remote.request, want)
		}
	})
	t.Run("ListSubscriptions", func(t *testing.T) {
		remote := &recordingInternal{response: completeWire[trpc.ListSubscriptionsResponse]()}
		fillSentinels(reflect.ValueOf(remote.response), "ListSubscriptions")
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.ListSubscriptions(context.Background(), &api.ListSubscriptionsRequest{Page: &api.PageInput{PageSize: 17, Cursor: "opaque+/="}, View: "history", State: "paused"})
		if e != nil || got == nil {
			t.Fatalf("response: %v %v", got, e)
		}
		compareWireFields(t, reflect.ValueOf(remote.response), reflect.ValueOf(got), "response")
		want := &trpc.ListSubscriptionsRequest{Actor: facadeActor, Page: &trpc.PageInput{PageSize: 17, Cursor: "opaque+/="}, View: "history", State: "paused"}
		if !reflect.DeepEqual(remote.request, want) {
			t.Fatalf("request got %#v want %#v", remote.request, want)
		}
	})
	t.Run("GetSubscription", func(t *testing.T) {
		remote := &recordingInternal{response: completeWire[trpc.GetSubscriptionResponse]()}
		fillSentinels(reflect.ValueOf(remote.response), "GetSubscription")
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.GetSubscription(context.Background(), &api.GetSubscriptionRequest{SubscriptionId: "sub"})
		if e != nil || got == nil {
			t.Fatalf("response: %v %v", got, e)
		}
		compareWireFields(t, reflect.ValueOf(remote.response), reflect.ValueOf(got), "response")
		want := &trpc.GetSubscriptionRequest{Actor: facadeActor, SubscriptionId: "sub"}
		if !reflect.DeepEqual(remote.request, want) {
			t.Fatalf("request got %#v want %#v", remote.request, want)
		}
	})
	t.Run("PauseSubscription", func(t *testing.T) {
		remote := &recordingInternal{response: completeWire[trpc.PauseSubscriptionResponse]()}
		fillSentinels(reflect.ValueOf(remote.response), "PauseSubscription")
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.PauseSubscription(context.Background(), &api.PauseSubscriptionRequest{SubscriptionId: "sub", ExpectedRevision: 18446744073709551615, RequestId: "PauseSubscription"})
		if e != nil || got == nil {
			t.Fatalf("response: %v %v", got, e)
		}
		compareWireFields(t, reflect.ValueOf(remote.response), reflect.ValueOf(got), "response")
		want := &trpc.PauseSubscriptionRequest{Actor: facadeActor, SubscriptionId: "sub", ExpectedRevision: 18446744073709551615, RequestId: "PauseSubscription"}
		if !reflect.DeepEqual(remote.request, want) {
			t.Fatalf("request got %#v want %#v", remote.request, want)
		}
	})
	t.Run("ResumeSubscription", func(t *testing.T) {
		remote := &recordingInternal{response: completeWire[trpc.ResumeSubscriptionResponse]()}
		fillSentinels(reflect.ValueOf(remote.response), "ResumeSubscription")
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.ResumeSubscription(context.Background(), &api.ResumeSubscriptionRequest{SubscriptionId: "sub", ExpectedRevision: 18446744073709551615, RequestId: "ResumeSubscription"})
		if e != nil || got == nil {
			t.Fatalf("response: %v %v", got, e)
		}
		compareWireFields(t, reflect.ValueOf(remote.response), reflect.ValueOf(got), "response")
		want := &trpc.ResumeSubscriptionRequest{Actor: facadeActor, SubscriptionId: "sub", ExpectedRevision: 18446744073709551615, RequestId: "ResumeSubscription"}
		if !reflect.DeepEqual(remote.request, want) {
			t.Fatalf("request got %#v want %#v", remote.request, want)
		}
	})
	t.Run("CancelSubscription", func(t *testing.T) {
		remote := &recordingInternal{response: completeWire[trpc.CancelSubscriptionResponse]()}
		fillSentinels(reflect.ValueOf(remote.response), "CancelSubscription")
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.CancelSubscription(context.Background(), &api.CancelSubscriptionRequest{SubscriptionId: "sub", ExpectedRevision: 18446744073709551615, RequestId: "CancelSubscription"})
		if e != nil || got == nil {
			t.Fatalf("response: %v %v", got, e)
		}
		compareWireFields(t, reflect.ValueOf(remote.response), reflect.ValueOf(got), "response")
		want := &trpc.CancelSubscriptionRequest{Actor: facadeActor, SubscriptionId: "sub", ExpectedRevision: 18446744073709551615, RequestId: "CancelSubscription"}
		if !reflect.DeepEqual(remote.request, want) {
			t.Fatalf("request got %#v want %#v", remote.request, want)
		}
	})
	t.Run("UpdateTargetNote", func(t *testing.T) {
		remote := &recordingInternal{response: completeWire[trpc.UpdateTargetNoteResponse]()}
		fillSentinels(reflect.ValueOf(remote.response), "UpdateTargetNote")
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.UpdateTargetNote(context.Background(), &api.UpdateTargetNoteRequest{Wallet: "wallet", Note: " note ", ExpectedRevision: 18446744073709551615, RequestId: "note-intent"})
		if e != nil || got == nil {
			t.Fatalf("response: %v %v", got, e)
		}
		compareWireFields(t, reflect.ValueOf(remote.response), reflect.ValueOf(got), "response")
		want := &trpc.UpdateTargetNoteRequest{Actor: facadeActor, Wallet: "wallet", Note: " note ", ExpectedRevision: 18446744073709551615, RequestId: "note-intent"}
		if !reflect.DeepEqual(remote.request, want) {
			t.Fatalf("request got %#v want %#v", remote.request, want)
		}
	})
	t.Run("ListActivities", func(t *testing.T) {
		remote := &recordingInternal{response: completeWire[trpc.ListActivitiesResponse]()}
		fillSentinels(reflect.ValueOf(remote.response), "ListActivities")
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.ListActivities(context.Background(), &api.ListActivitiesRequest{Page: &api.PageInput{PageSize: 19, Cursor: "cursor"}, SubscriptionId: "sub", From: "from", To: "to", SummaryBatchId: "9007199254740993", RefreshCursor: "refresh"})
		if e != nil || got == nil {
			t.Fatalf("response: %v %v", got, e)
		}
		compareWireFields(t, reflect.ValueOf(remote.response), reflect.ValueOf(got), "response")
		want := &trpc.ListActivitiesRequest{Actor: facadeActor, Page: &trpc.PageInput{PageSize: 19, Cursor: "cursor"}, SubscriptionId: "sub", From: "from", To: "to", SummaryBatchId: "9007199254740993", RefreshCursor: "refresh"}
		if !reflect.DeepEqual(remote.request, want) {
			t.Fatalf("request got %#v want %#v", remote.request, want)
		}
	})
	t.Run("GetActivity", func(t *testing.T) {
		remote := &recordingInternal{response: completeWire[trpc.GetActivityResponse]()}
		fillSentinels(reflect.ValueOf(remote.response), "GetActivity")
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.GetActivity(context.Background(), &api.GetActivityRequest{ActivityId: "9007199254740993"})
		if e != nil || got == nil {
			t.Fatalf("response: %v %v", got, e)
		}
		compareWireFields(t, reflect.ValueOf(remote.response), reflect.ValueOf(got), "response")
		want := &trpc.GetActivityRequest{Actor: facadeActor, ActivityId: "9007199254740993"}
		if !reflect.DeepEqual(remote.request, want) {
			t.Fatalf("request got %#v want %#v", remote.request, want)
		}
	})
	t.Run("ListSubscriptionHistory", func(t *testing.T) {
		remote := &recordingInternal{response: completeWire[trpc.ListSubscriptionHistoryResponse]()}
		fillSentinels(reflect.ValueOf(remote.response), "ListSubscriptionHistory")
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.ListSubscriptionHistory(context.Background(), &api.ListSubscriptionHistoryRequest{SubscriptionId: "sub", Page: &api.PageInput{PageSize: 23, Cursor: "history"}})
		if e != nil || got == nil {
			t.Fatalf("response: %v %v", got, e)
		}
		compareWireFields(t, reflect.ValueOf(remote.response), reflect.ValueOf(got), "response")
		want := &trpc.ListSubscriptionHistoryRequest{Actor: facadeActor, SubscriptionId: "sub", Page: &trpc.PageInput{PageSize: 23, Cursor: "history"}}
		if !reflect.DeepEqual(remote.request, want) {
			t.Fatalf("request got %#v want %#v", remote.request, want)
		}
	})
	t.Run("GetSummaryBatch", func(t *testing.T) {
		remote := &recordingInternal{response: completeWire[trpc.GetSummaryBatchResponse]()}
		fillSentinels(reflect.ValueOf(remote.response), "GetSummaryBatch")
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.GetSummaryBatch(context.Background(), &api.GetSummaryBatchRequest{BatchId: "9007199254740995"})
		if e != nil || got == nil {
			t.Fatalf("response: %v %v", got, e)
		}
		compareWireFields(t, reflect.ValueOf(remote.response), reflect.ValueOf(got), "response")
		want := &trpc.GetSummaryBatchRequest{Actor: facadeActor, BatchId: "9007199254740995"}
		if !reflect.DeepEqual(remote.request, want) {
			t.Fatalf("request got %#v want %#v", remote.request, want)
		}
	})
	t.Run("ListSummaryParts", func(t *testing.T) {
		remote := &recordingInternal{response: completeWire[trpc.ListSummaryPartsResponse]()}
		fillSentinels(reflect.ValueOf(remote.response), "ListSummaryParts")
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.ListSummaryParts(context.Background(), &api.ListSummaryPartsRequest{BatchId: "batch", ActivityId: "activity", Page: &api.PageInput{PageSize: 29, Cursor: "parts"}})
		if e != nil || got == nil {
			t.Fatalf("response: %v %v", got, e)
		}
		compareWireFields(t, reflect.ValueOf(remote.response), reflect.ValueOf(got), "response")
		want := &trpc.ListSummaryPartsRequest{Actor: facadeActor, BatchId: "batch", ActivityId: "activity", Page: &trpc.PageInput{PageSize: 29, Cursor: "parts"}}
		if !reflect.DeepEqual(remote.request, want) {
			t.Fatalf("request got %#v want %#v", remote.request, want)
		}
	})
	t.Run("ListSubscriptionSummaries", func(t *testing.T) {
		remote := &recordingInternal{response: completeWire[trpc.ListSubscriptionSummariesResponse]()}
		fillSentinels(reflect.ValueOf(remote.response), "ListSubscriptionSummaries")
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.ListSubscriptionSummaries(context.Background(), &api.ListSubscriptionSummariesRequest{Page: &api.PageInput{PageSize: 31, Cursor: "admin"}, AccountId: "", State: "paused", Wallet: "wallet", IncludeCancelled: true})
		if e != nil || got == nil {
			t.Fatalf("response: %v %v", got, e)
		}
		compareWireFields(t, reflect.ValueOf(remote.response), reflect.ValueOf(got), "response")
		want := &trpc.ListSubscriptionSummariesRequest{Actor: facadeActor, Page: &trpc.PageInput{PageSize: 31, Cursor: "admin"}, AccountId: "", State: "paused", Wallet: "wallet", IncludeCancelled: true}
		if !reflect.DeepEqual(remote.request, want) {
			t.Fatalf("request got %#v want %#v", remote.request, want)
		}
	})
	t.Run("GetSubscriptionSummary", func(t *testing.T) {
		remote := &recordingInternal{response: completeWire[trpc.GetSubscriptionSummaryResponse]()}
		fillSentinels(reflect.ValueOf(remote.response), "GetSubscriptionSummary")
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.GetSubscriptionSummary(context.Background(), &api.GetSubscriptionSummaryRequest{SubscriptionId: "admin-sub"})
		if e != nil || got == nil {
			t.Fatalf("response: %v %v", got, e)
		}
		compareWireFields(t, reflect.ValueOf(remote.response), reflect.ValueOf(got), "response")
		want := &trpc.GetSubscriptionSummaryRequest{Actor: facadeActor, SubscriptionId: "admin-sub"}
		if !reflect.DeepEqual(remote.request, want) {
			t.Fatalf("request got %#v want %#v", remote.request, want)
		}
	})
	t.Run("GetTraderSyncRuntimeStatus", func(t *testing.T) {
		remote := &recordingInternal{response: completeWire[trpc.GetTraderSyncRuntimeStatusResponse]()}
		fillSentinels(reflect.ValueOf(remote.response), "GetTraderSyncRuntimeStatus")
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.GetTraderSyncRuntimeStatus(context.Background(), &api.GetTraderSyncRuntimeStatusRequest{})
		if e != nil || got == nil {
			t.Fatalf("response: %v %v", got, e)
		}
		compareWireFields(t, reflect.ValueOf(remote.response), reflect.ValueOf(got), "response")
		want := &trpc.GetTraderSyncRuntimeStatusRequest{Actor: facadeActor}
		if !reflect.DeepEqual(remote.request, want) {
			t.Fatalf("request got %#v want %#v", remote.request, want)
		}
	})
}
func TestFacadeMissingRequiredResponsesAreUnavailable(t *testing.T) {
	t.Run("ResolveTarget", func(t *testing.T) {
		remote := &recordingInternal{response: &trpc.ResolveTargetResponse{}}
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.ResolveTarget(context.Background(), &api.ResolveTargetRequest{})
		if status.Code(e) != codes.Unavailable || got != nil {
			t.Fatalf("got %v %v", got, e)
		}
	})
	t.Run("CreateSubscription", func(t *testing.T) {
		remote := &recordingInternal{response: &trpc.CreateSubscriptionResponse{}}
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.CreateSubscription(context.Background(), &api.CreateSubscriptionRequest{})
		if status.Code(e) != codes.Unavailable || got != nil {
			t.Fatalf("got %v %v", got, e)
		}
	})
	t.Run("ListSubscriptions", func(t *testing.T) {
		remote := &recordingInternal{response: &trpc.ListSubscriptionsResponse{}}
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.ListSubscriptions(context.Background(), &api.ListSubscriptionsRequest{})
		if status.Code(e) != codes.Unavailable || got != nil {
			t.Fatalf("got %v %v", got, e)
		}
	})
	t.Run("GetSubscription", func(t *testing.T) {
		remote := &recordingInternal{response: &trpc.GetSubscriptionResponse{}}
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.GetSubscription(context.Background(), &api.GetSubscriptionRequest{})
		if status.Code(e) != codes.Unavailable || got != nil {
			t.Fatalf("got %v %v", got, e)
		}
	})
	t.Run("PauseSubscription", func(t *testing.T) {
		remote := &recordingInternal{response: &trpc.PauseSubscriptionResponse{}}
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.PauseSubscription(context.Background(), &api.PauseSubscriptionRequest{})
		if status.Code(e) != codes.Unavailable || got != nil {
			t.Fatalf("got %v %v", got, e)
		}
	})
	t.Run("ResumeSubscription", func(t *testing.T) {
		remote := &recordingInternal{response: &trpc.ResumeSubscriptionResponse{}}
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.ResumeSubscription(context.Background(), &api.ResumeSubscriptionRequest{})
		if status.Code(e) != codes.Unavailable || got != nil {
			t.Fatalf("got %v %v", got, e)
		}
	})
	t.Run("CancelSubscription", func(t *testing.T) {
		remote := &recordingInternal{response: &trpc.CancelSubscriptionResponse{}}
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.CancelSubscription(context.Background(), &api.CancelSubscriptionRequest{})
		if status.Code(e) != codes.Unavailable || got != nil {
			t.Fatalf("got %v %v", got, e)
		}
	})
	t.Run("UpdateTargetNote", func(t *testing.T) {
		remote := &recordingInternal{response: &trpc.UpdateTargetNoteResponse{}}
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.UpdateTargetNote(context.Background(), &api.UpdateTargetNoteRequest{})
		if status.Code(e) != codes.Unavailable || got != nil {
			t.Fatalf("got %v %v", got, e)
		}
	})
	t.Run("ListActivities", func(t *testing.T) {
		remote := &recordingInternal{response: &trpc.ListActivitiesResponse{}}
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.ListActivities(context.Background(), &api.ListActivitiesRequest{})
		if status.Code(e) != codes.Unavailable || got != nil {
			t.Fatalf("got %v %v", got, e)
		}
	})
	t.Run("GetActivity", func(t *testing.T) {
		remote := &recordingInternal{response: &trpc.GetActivityResponse{}}
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.GetActivity(context.Background(), &api.GetActivityRequest{})
		if status.Code(e) != codes.Unavailable || got != nil {
			t.Fatalf("got %v %v", got, e)
		}
	})
	t.Run("ListSubscriptionHistory", func(t *testing.T) {
		remote := &recordingInternal{response: &trpc.ListSubscriptionHistoryResponse{}}
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.ListSubscriptionHistory(context.Background(), &api.ListSubscriptionHistoryRequest{})
		if status.Code(e) != codes.Unavailable || got != nil {
			t.Fatalf("got %v %v", got, e)
		}
	})
	t.Run("GetSummaryBatch", func(t *testing.T) {
		remote := &recordingInternal{response: &trpc.GetSummaryBatchResponse{}}
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.GetSummaryBatch(context.Background(), &api.GetSummaryBatchRequest{})
		if status.Code(e) != codes.Unavailable || got != nil {
			t.Fatalf("got %v %v", got, e)
		}
	})
	t.Run("ListSummaryParts", func(t *testing.T) {
		remote := &recordingInternal{response: &trpc.ListSummaryPartsResponse{}}
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.ListSummaryParts(context.Background(), &api.ListSummaryPartsRequest{})
		if status.Code(e) != codes.Unavailable || got != nil {
			t.Fatalf("got %v %v", got, e)
		}
	})
	t.Run("ListSubscriptionSummaries", func(t *testing.T) {
		remote := &recordingInternal{response: &trpc.ListSubscriptionSummariesResponse{}}
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.ListSubscriptionSummaries(context.Background(), &api.ListSubscriptionSummariesRequest{})
		if status.Code(e) != codes.Unavailable || got != nil {
			t.Fatalf("got %v %v", got, e)
		}
	})
	t.Run("GetSubscriptionSummary", func(t *testing.T) {
		remote := &recordingInternal{response: &trpc.GetSubscriptionSummaryResponse{}}
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.GetSubscriptionSummary(context.Background(), &api.GetSubscriptionSummaryRequest{})
		if status.Code(e) != codes.Unavailable || got != nil {
			t.Fatalf("got %v %v", got, e)
		}
	})
	t.Run("GetTraderSyncRuntimeStatus", func(t *testing.T) {
		remote := &recordingInternal{response: &trpc.GetTraderSyncRuntimeStatusResponse{}}
		s := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		got, e := s.GetTraderSyncRuntimeStatus(context.Background(), &api.GetTraderSyncRuntimeStatusRequest{})
		if status.Code(e) != codes.Unavailable || got != nil {
			t.Fatalf("got %v %v", got, e)
		}
	})
}
func TestInternalAuthenticationFailureBecomesUnavailable(t *testing.T) {
	for _, tc := range []struct {
		code           codes.Code
		domain, reason string
		want           codes.Code
	}{{codes.Unauthenticated, "tradersync.internal.v1", "SERVICE_AUTH_INVALID", codes.Unavailable}, {codes.Unauthenticated, "tradersync.internal.v1", "SERVICE_AUTH_MISSING", codes.Unavailable}, {codes.InvalidArgument, "tradersync.internal.v1", "ACTOR_INVALID", codes.Unavailable}, {codes.PermissionDenied, "", "", codes.PermissionDenied}, {codes.InvalidArgument, "other", "ACTOR_INVALID", codes.InvalidArgument}} {
		s, e := status.New(tc.code, "private-detail").WithDetails(&errdetails.ErrorInfo{Domain: tc.domain, Reason: tc.reason})
		if e != nil {
			t.Fatal(e)
		}
		if got := status.Code(MapInternalError(s.Err())); got != tc.want {
			t.Fatalf("got %v want %v", got, tc.want)
		}
		remote := &recordingInternal{err: s.Err()}
		facade := New(trpc.NewTraderSyncServiceClient(internalConnection(t, remote)), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
		_, e = facade.ResolveTarget(context.Background(), &api.ResolveTargetRequest{Input: "wallet"})
		if status.Code(e) != tc.want {
			t.Fatal(e)
		}
	}
}
func TestPublicIdentityFailureRemainsUnauthenticated(t *testing.T) {
	s := New(trpc.NewUnavailableClient("dependency_unavailable"), func(context.Context) (*trpc.Actor, error) {
		return nil, status.Error(codes.Unauthenticated, "public identity required")
	})
	_, e := s.ResolveTarget(context.Background(), &api.ResolveTargetRequest{})
	if status.Code(e) != codes.Unauthenticated {
		t.Fatal(e)
	}
}

func TestActualInternalTokenGateNeverBecomesPublic401(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	remote := &recordingInternal{response: completeWire[trpc.ResolveTargetResponse]()}
	fillSentinels(reflect.ValueOf(remote.response), "ResolveTarget")
	server := grpc.NewServer(grpc.UnaryInterceptor(transport.NewUnaryInterceptor("valid-internal-token-01234567890123", func() bool { return true })))
	trpc.RegisterTraderSyncServiceServer(server, remote)
	go server.Serve(listener)
	defer server.Stop()
	conn, e := grpc.DialContext(context.Background(), "passthrough:///auth", grpc.WithInsecure(), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	facade := New(trpc.NewTraderSyncServiceClient(conn), func(context.Context) (*trpc.Actor, error) { return facadeActor, nil })
	for _, token := range []string{"", "wrong-internal-token-0123456789012"} {
		ctx := context.Background()
		if token != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		}
		_, e = facade.ResolveTarget(ctx, &api.ResolveTargetRequest{Input: "wallet"})
		if status.Code(e) != codes.Unavailable {
			t.Fatal("internal credentials became public failure", e)
		}
	}
	if remote.request != nil {
		t.Fatal("unauthenticated request reached business handler")
	}
}
