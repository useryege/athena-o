package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/operationlog/event"
	"github.com/useryege/athena/internal/operationlog/ingest"
	"github.com/useryege/athena/internal/operationlog/record"
	accountpkg "github.com/useryege/athena/pkg/apiclient/account"
	util_session "github.com/useryege/athena/util/session"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type operationLogTestSink struct {
	mu     sync.Mutex
	events []event.Event
}

type operationLogFailingSink struct{}

func (operationLogFailingSink) Append(context.Context, event.Event) error {
	return errors.New("operation log sink unavailable")
}
func (operationLogFailingSink) PublishStatus(context.Context, ingest.Status) error {
	return errors.New("status sink unavailable")
}

func (s *operationLogTestSink) Append(_ context.Context, e event.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
	return nil
}

func (s *operationLogTestSink) PublishStatus(context.Context, ingest.Status) error { return nil }

func (s *operationLogTestSink) snapshot() []event.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]event.Event(nil), s.events...)
}

type operationLogAuthOverride struct{}

func (operationLogAuthOverride) AuthFuncOverride(ctx context.Context, _ string) (context.Context, error) {
	return withDisabledAuthClaims(ctx, "00000000-0000-4000-8000-000000000001"), nil
}

type operationLogAPIKeyOverride struct{}

func (operationLogAPIKeyOverride) AuthFuncOverride(ctx context.Context, _ string) (context.Context, error) {
	ctx = withDisabledAuthClaims(ctx, "00000000-0000-4000-8000-000000000001")
	return util_session.WithAuthenticatedCredential(ctx, accountcredentials.AuthenticatedCredential{AccountID: "00000000-0000-4000-8000-000000000001", Capability: accountcredentials.CapabilityAPIKey, JTI: "api-key-jti", AccessRevision: 1}), nil
}

type operationLogAuthFailureOverride struct{}

func (operationLogAuthFailureOverride) AuthFuncOverride(ctx context.Context, _ string) (context.Context, error) {
	// Deliberately leave a credential in the failed context. The interceptor
	// must rely on the authentication success marker rather than status.Code.
	return withDisabledAuthClaims(ctx, "00000000-0000-4000-8000-000000000001"), status.Error(codes.Internal, "authentication backend unavailable")
}

func newOperationLogTestServer(sink *operationLogTestSink) (*AthenaServer, func()) {
	producer := ingest.New(sink)
	server := &AthenaServer{operationLogProducer: producer}
	return server, func() { _ = producer.Close(context.Background()) }
}

func TestUnaryAuthInterceptorRecordsCataloguedOperationAndDoesNotBlockBusiness(t *testing.T) {
	sink := &operationLogTestSink{}
	server, cleanup := newOperationLogTestServer(sink)
	defer cleanup()

	called := false
	result, err := server.unaryAuthInterceptor(context.Background(), struct{}{}, &grpc.UnaryServerInfo{
		FullMethod: "/account.AccountService/CreateToken",
		Server:     operationLogAuthOverride{},
	}, func(ctx context.Context, _ any) (any, error) {
		called = true
		if credential, ok := util_session.AuthenticatedCredentialFromContext(ctx); !ok || credential.AccountID != "00000000-0000-4000-8000-000000000001" {
			t.Fatal("handler lost authenticated context")
		}
		if operationID := operationLogRecorder(ctx); operationID == "" {
			t.Fatal("handler did not receive operation recorder")
		}
		return "ok", nil
	})
	if err != nil || result != "ok" || !called {
		t.Fatalf("result=%v err=%v called=%v", result, err, called)
	}
	events := sink.snapshot()
	if len(events) != 2 || events[0].Phase != event.Start || events[1].Phase != event.Finish {
		t.Fatalf("events=%+v", events)
	}
	if events[1].ActionCode != "account.api_key.create" || events[1].Actor.AccountID == nil || !events[1].Actor.IdentityVerified {
		t.Fatalf("event=%+v", events[1])
	}
}

func TestUnaryAuthInterceptorDoesNotBlockBusinessWhenLogSinkFails(t *testing.T) {
	server := &AthenaServer{operationLogProducer: ingest.New(operationLogFailingSink{})}
	defer func() { _ = server.operationLogProducer.Close(context.Background()) }()
	called := false
	result, err := server.unaryAuthInterceptor(context.Background(), struct{}{}, &grpc.UnaryServerInfo{
		FullMethod: "/account.AccountService/CreateToken",
		Server:     operationLogAuthOverride{},
	}, func(context.Context, any) (any, error) {
		called = true
		return "business-result", nil
	})
	if err != nil || result != "business-result" || !called {
		t.Fatalf("result=%v err=%v called=%v", result, err, called)
	}
}

func TestUnaryAuthInterceptorRecordsAuthorizationFailureAndRethrowsPanic(t *testing.T) {
	t.Run("authorization failure", func(t *testing.T) {
		sink := &operationLogTestSink{}
		server, cleanup := newOperationLogTestServer(sink)
		defer cleanup()

		called := false
		_, err := server.unaryAuthInterceptor(context.Background(), struct{}{}, &grpc.UnaryServerInfo{
			FullMethod: "/account.AccountService/UpdateAccountAccess",
			Server:     operationLogAuthOverride{},
		}, func(context.Context, any) (any, error) {
			called = true
			return nil, nil
		})
		if err == nil || called || status.Code(err) != codes.Internal {
			t.Fatalf("err=%v called=%v", err, called)
		}
		events := sink.snapshot()
		if len(events) != 1 || events[0].Outcome != event.Unknown || events[0].Phase != event.Finish {
			t.Fatalf("events=%+v", events)
		}
	})

	t.Run("panic", func(t *testing.T) {
		sink := &operationLogTestSink{}
		server, cleanup := newOperationLogTestServer(sink)
		defer cleanup()

		panicValue := errors.New("handler panic")
		defer func() {
			if recovered := recover(); recovered != panicValue {
				t.Fatalf("recovered=%v", recovered)
			}
			events := sink.snapshot()
			if len(events) != 2 || events[1].GRPCCode == nil || *events[1].GRPCCode != codes.Internal.String() {
				t.Fatalf("events=%+v", events)
			}
		}()
		_, _ = server.unaryAuthInterceptor(context.Background(), struct{}{}, &grpc.UnaryServerInfo{
			FullMethod: "/account.AccountService/CreateToken",
			Server:     operationLogAuthOverride{},
		}, func(context.Context, any) (any, error) {
			panic(panicValue)
		})
	})

	t.Run("authentication failure does not trust stale credential", func(t *testing.T) {
		sink := &operationLogTestSink{}
		server, cleanup := newOperationLogTestServer(sink)
		defer cleanup()

		_, err := server.unaryAuthInterceptor(context.Background(), struct{}{}, &grpc.UnaryServerInfo{
			FullMethod: "/account.AccountService/CreateToken",
			Server:     operationLogAuthFailureOverride{},
		}, func(context.Context, any) (any, error) {
			t.Fatal("handler must not run")
			return nil, nil
		})
		if status.Code(err) != codes.Internal {
			t.Fatalf("err=%v", err)
		}
		events := sink.snapshot()
		if len(events) != 1 || events[0].Actor.AccountID != nil || events[0].Actor.IdentityVerified {
			t.Fatalf("events=%+v", events)
		}
	})
}

func TestOperationLogQueryRequiresInteractiveAdministratorCredential(t *testing.T) {
	sink := &operationLogTestSink{}
	server, cleanup := newOperationLogTestServer(sink)
	defer cleanup()
	accountID := "00000000-0000-4000-8000-000000000001"
	controller, err := accountaccess.NewController(context.Background(), traderAuthStore{accountID: accountaccess.Access{Administrator: true, LoginEnabled: true, Modules: accountaccess.NoModuleAccess(), Revision: 1}})
	if err != nil {
		t.Fatal(err)
	}
	server.accessController = controller
	called := false
	_, err = server.unaryAuthInterceptor(context.Background(), struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/operationlog.OperationLogService/ListOperationLogs", Server: operationLogAPIKeyOverride{}}, func(context.Context, any) (any, error) {
		called = true
		return nil, nil
	})
	if status.Code(err) != codes.PermissionDenied || called {
		t.Fatalf("err=%v called=%v", err, called)
	}
	// Query methods are administrator-only but deliberately excluded from the
	// mutating operation catalog, so authorization is verified without an
	// operation event assertion here.
}

func TestBeginOperationLogOnlyUsesApprovedGRPCCatalogEntries(t *testing.T) {
	for _, entry := range event.Catalog() {
		if entry.Transport != "grpc" {
			continue
		}
		if _, ok := event.Lookup(entry.Entry); !ok {
			t.Fatalf("catalog entry not resolvable: %s", entry.Entry)
		}
	}
	if got := len(event.Catalog()); got != 98 {
		t.Fatalf("catalog size=%d", got)
	}
	if _, ok := event.Lookup("/not-an-approved-method"); ok {
		t.Fatal("unknown method entered catalog")
	}
}

func TestObserveOperationLogResultUsesTypedPathsAndCommitEvidence(t *testing.T) {
	sink := &operationLogTestSink{}
	server, cleanup := newOperationLogTestServer(sink)
	defer cleanup()

	ctx, recorder := server.beginOperationLog(context.Background(), "/account.AccountService/CreateToken")
	if recorder == nil {
		t.Fatal("recorder is nil")
	}
	recorder.Start()
	observeOperationLogResult(recorder, "/account.AccountService/CreateToken",
		&accountpkg.CreateTokenRequest{Id: "ops-key", ExpiresIn: 3600},
		&accountpkg.CreateTokenResponse{Token: "secret-must-not-be-captured"})
	recorder.Finish()
	events := sink.snapshot()
	if len(events) != 2 {
		t.Fatalf("events=%+v", events)
	}
	finish := events[1]
	if finish.Outcome != event.Succeeded || len(finish.Effect) != 1 || finish.Effect[0] != "ACCOUNT_API_KEY_CREATE" {
		t.Fatalf("finish=%+v", finish)
	}
	displayID, _ := json.Marshal(finish.Details["displayId"])
	expiresIn, _ := json.Marshal(finish.Details["expiresIn"])
	if string(displayID) != `"ops-key"` || string(expiresIn) != `"3600"` {
		t.Fatalf("details=%+v", finish.Details)
	}
	if _, ok := finish.Details["token"]; ok || strings.Contains(fmt.Sprint(finish), "secret-must-not-be-captured") {
		t.Fatalf("sensitive token leaked: %+v", finish)
	}
	if err := event.Validate(finish); err != nil {
		t.Fatalf("finish event is invalid: %v (%+v)", err, finish)
	}
	_ = ctx

	sink = &operationLogTestSink{}
	server, cleanup = newOperationLogTestServer(sink)
	defer cleanup()
	_, recorder = server.beginOperationLog(context.Background(), "/account.AccountService/CreateToken")
	recorder.Start()
	// An identically named field nested under an unrelated object is not a
	// capture position and cannot establish either a resource or a commit.
	observeOperationLogResult(recorder, "/account.AccountService/CreateToken", map[string]any{}, map[string]any{
		"nested": map[string]any{"token": "forbidden", "id": "nested-id"},
	})
	recorder.Finish()
	events = sink.snapshot()
	if len(events) != 2 || events[1].Outcome != event.Unknown || len(events[1].Resources) != 0 {
		t.Fatalf("nested fields fabricated evidence: %+v", events)
	}
}

func TestObserveOperationLogResultParsesAccountAccessEnumJSON(t *testing.T) {
	sink := &operationLogTestSink{}
	server, cleanup := newOperationLogTestServer(sink)
	defer cleanup()

	modules := []accountpkg.AccountDataModule{
		accountpkg.AccountDataModule_ACCOUNT_DATA_MODULE_MARKET_RADAR,
		accountpkg.AccountDataModule_ACCOUNT_DATA_MODULE_MANAGED_OO,
		accountpkg.AccountDataModule_ACCOUNT_DATA_MODULE_TOKEN,
		accountpkg.AccountDataModule_ACCOUNT_DATA_MODULE_WALLET,
		accountpkg.AccountDataModule_ACCOUNT_DATA_MODULE_WORM_TRADING,
		accountpkg.AccountDataModule_ACCOUNT_DATA_MODULE_TRADER_SYNC,
		accountpkg.AccountDataModule_ACCOUNT_DATA_MODULE_SOLANA,
	}
	grants := make([]*accountpkg.AccountModuleAccess, 0, len(modules))
	for _, module := range modules {
		grants = append(grants, &accountpkg.AccountModuleAccess{
			Module:     module,
			DataAccess: accountpkg.AccountDataAccess_ACCOUNT_DATA_ACCESS_READ,
		})
	}
	access := &accountpkg.AccountAccess{LoginEnabled: true, Revision: 7, ModuleAccess: grants}
	request := &accountpkg.UpdateAccountAccessRequest{Id: "00000000-0000-4000-8000-000000000001", Access: access}
	response := &accountpkg.Account{Id: request.Id, Access: access}
	_, recorder := server.beginOperationLog(context.Background(), "/account.AccountService/UpdateAccountAccess")
	recorder.Start()
	observeOperationLogResult(recorder, "/account.AccountService/UpdateAccountAccess", request, response)
	recorder.Finish()
	events := sink.snapshot()
	if len(events) != 2 {
		t.Fatalf("events=%+v", events)
	}
	finish := events[1]
	if err := event.Validate(finish); err != nil {
		t.Fatalf("finish event is invalid: %v (%+v)", err, finish)
	}
	for _, key := range []string{"requestedAccess", "confirmedAccess"} {
		if _, ok := finish.Details[key]; !ok {
			t.Fatalf("missing %s in details: %+v", key, finish.Details)
		}
	}
}

func operationLogRecorder(ctx context.Context) string {
	if recorder := record.FromContext(ctx); recorder != nil {
		return recorder.OperationID()
	}
	return ""
}
