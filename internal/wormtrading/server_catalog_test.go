package wormtrading

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountaccess"
	walletapi "github.com/useryege/athena/internal/wallet/apiclient"
	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	utilworm "github.com/useryege/athena/util/worm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type catalogReaderFunc func(context.Context, *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error)

func (f catalogReaderFunc) GetOrderEventCatalog(ctx context.Context, req *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
	return f(ctx, req)
}
func newCatalogRPCClient(t *testing.T, s *Service) apiclient.WormTradingServiceClient {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := (&Server{healthService: health.NewServer(), service: s, internalAuthTokenHash: sha256.Sum256([]byte("catalog-test-token"))}).CreateGRPC()
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///catalog", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(); _ = listener.Close() })
	return apiclient.NewWormTradingServiceClient(conn)
}
func TestCatalogRPCAuthAndCurrentAccess(t *testing.T) {
	access := catalogAccess(accountaccess.AccessLevelRead)
	var storeErr error
	var calls atomic.Int32
	s := &Service{accountAccessReader: accountAccessFunc(func(context.Context, string) (accountaccess.Access, error) { return access, storeErr }), catalogReader: catalogReaderFunc(func(_ context.Context, req *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
		calls.Add(1)
		return &apiclient.GetOrderEventCatalogResponse{Event: &apiclient.OrderEventCatalog{EventConditionId: req.EventConditionId, Title: "event"}, FetchedAt: 123}, nil
	})}
	client := newCatalogRPCClient(t, s)
	for _, tc := range []struct {
		name, bearer             string
		ids                      []string
		level                    accountaccess.AccessLevel
		disabled, admin, markets bool
		err                      error
		want                     codes.Code
	}{
		{name: "missing bearer", ids: []string{catalogAccountID}, want: codes.Unauthenticated},
		{name: "wrong bearer", bearer: "Bearer wrong", ids: []string{catalogAccountID}, want: codes.Unauthenticated},
		{name: "missing identity", bearer: "Bearer catalog-test-token", want: codes.Unauthenticated},
		{name: "duplicate", bearer: "Bearer catalog-test-token", ids: []string{catalogAccountID, catalogAccountID}, want: codes.Unauthenticated},
		{name: "noncanonical", bearer: "Bearer catalog-test-token", ids: []string{strings.ToUpper(catalogAccountID)}, want: codes.Unauthenticated},
		{name: "markets only", markets: true, want: codes.PermissionDenied},
		{name: "read", level: accountaccess.AccessLevelRead, want: codes.OK},
		{name: "revoked", want: codes.PermissionDenied},
		{name: "read write", level: accountaccess.AccessLevelReadWrite, want: codes.OK},
		{name: "disabled", level: accountaccess.AccessLevelRead, disabled: true, want: codes.PermissionDenied},
		{name: "admin", admin: true, want: codes.PermissionDenied},
		{name: "missing account", err: pgx.ErrNoRows, want: codes.PermissionDenied},
		{name: "db failure", err: errors.New("offline"), want: codes.Unavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			access = catalogAccess(tc.level)
			if tc.level == "" {
				access.Modules[accountaccess.ModuleWormTrading] = accountaccess.AccessLevelNone
			}
			access.LoginEnabled = !tc.disabled
			access.Administrator = tc.admin
			if tc.markets {
				access.Modules[accountaccess.ModuleWormMarkets] = accountaccess.AccessLevelRead
			}
			storeErr = tc.err
			md := metadata.MD{}
			if tc.bearer != "" {
				md.Set("authorization", tc.bearer)
			}
			if tc.ids != nil {
				md[AccountIDMetadataKey] = tc.ids
			}
			if tc.bearer == "" && tc.ids == nil {
				md.Set("authorization", "Bearer catalog-test-token")
				md.Set(AccountIDMetadataKey, catalogAccountID)
			}
			before := calls.Load()
			ctx, cancel := context.WithTimeout(metadata.NewOutgoingContext(context.Background(), md), time.Second)
			defer cancel()
			resp, err := client.GetOrderEventCatalog(ctx, &apiclient.GetOrderEventCatalogRequest{EventConditionId: "event-id"})
			require.Equal(t, tc.want, status.Code(err))
			if tc.want == codes.OK {
				require.Equal(t, "event-id", resp.Event.EventConditionId)
				require.Equal(t, before+1, calls.Load())
			} else {
				require.Equal(t, before, calls.Load())
			}
		})
	}
}
func TestCatalogServiceRequiresAccessAndBudget(t *testing.T) {
	reader := accountAccessFunc(func(context.Context, string) (accountaccess.Access, error) {
		return catalogAccess(accountaccess.AccessLevelRead), nil
	})
	for _, tc := range []struct {
		name   string
		reader AccountAccessReader
		budget time.Duration
		want   string
	}{
		{"missing reader", nil, 45 * time.Second, "account access reader"},
		{"zero budget", reader, 0, "catalog budget"}, {"negative budget", reader, -time.Second, "catalog budget"}, {"short budget", reader, time.Second, "catalog budget"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewServiceWithOptions(ServiceOptions{AccountAccessReader: tc.reader, WormCatalogBudget: tc.budget})
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestCatalogServerWiresDefaultAndInjectedReaders(t *testing.T) {
	signer, err := walletapi.NewWormExecutionSignerClientset("127.0.0.1:1", strings.Repeat("s", 32))
	require.NoError(t, err)
	t.Cleanup(func() { _ = signer.Close() })
	for _, injected := range []bool{false, true} {
		t.Run(fmt.Sprint(injected), func(t *testing.T) {
			reads := 0
			opts := ServerOpts{AccountAccessReader: accountAccessFunc(func(_ context.Context, id string) (accountaccess.Access, error) {
				reads++
				require.Equal(t, catalogAccountID, id)
				return catalogAccess(accountaccess.AccessLevelRead), nil
			}), WormCatalogBudget: 45 * time.Second, CredentialStore: &wormstore.SQLStore{}, CredentialEncryptionKey: []byte(strings.Repeat("k", 32)), WalletSignerClientset: signer, WormWebClient: utilworm.NewWebClient(utilworm.WebClientConfig{}), InternalAuthToken: strings.Repeat("t", 32)}
			if injected {
				opts.CatalogReader = catalogReaderFunc(func(context.Context, *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
					return &apiclient.GetOrderEventCatalogResponse{FetchedAt: 123}, nil
				})
			}
			server, err := NewServer(opts)
			require.NoError(t, err)
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(AccountIDMetadataKey, catalogAccountID))
			result, err := server.service.GetOrderEventCatalog(ctx, &apiclient.GetOrderEventCatalogRequest{EventConditionId: "invalid"})
			if injected {
				require.NoError(t, err)
				require.EqualValues(t, 123, result.FetchedAt)
			} else {
				require.Equal(t, codes.InvalidArgument, status.Code(err))
			}
			require.Equal(t, 1, reads)
		})
	}
}
