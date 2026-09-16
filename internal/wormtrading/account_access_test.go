package wormtrading

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountaccess"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type accountAccessFunc func(context.Context, string) (accountaccess.Access, error)

func (f accountAccessFunc) GetAccountAccess(ctx context.Context, id string) (accountaccess.Access, error) {
	return f(ctx, id)
}

const catalogAccountID = "d30fa35d-78a6-43d6-8faf-ed5b1b9e63d1"

func catalogAccess(level accountaccess.AccessLevel) accountaccess.Access {
	a := accountaccess.Access{LoginEnabled: true, Revision: 1, Modules: accountaccess.NoModuleAccess()}
	a.Modules[accountaccess.ModuleWormTrading] = level
	return a
}
func TestCatalogAccountRejectsDuplicateIdentity(t *testing.T) {
	s := &Service{accountAccessReader: accountAccessFunc(func(context.Context, string) (accountaccess.Access, error) {
		t.Fatal("ambiguous identity reached account store")
		return accountaccess.Access{}, nil
	})}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(AccountIDMetadataKey, catalogAccountID, AccountIDMetadataKey, catalogAccountID))
	_, err := s.authorizeCatalogAccount(ctx, "", accountaccess.AccessLevelRead)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}
func TestCatalogAccountAuthorization(t *testing.T) {
	for _, tc := range []struct {
		name       string
		ids        []string
		owner      string
		access     accountaccess.Access
		err        error
		required   accountaccess.AccessLevel
		want       codes.Code
		storeCalls int
	}{
		{name: "missing", want: codes.Unauthenticated},
		{name: "empty", ids: []string{""}, want: codes.Unauthenticated},
		{name: "uppercase", ids: []string{strings.ToUpper(catalogAccountID)}, want: codes.Unauthenticated},
		{name: "zero", ids: []string{"00000000-0000-0000-0000-000000000000"}, want: codes.Unauthenticated},
		{name: "compact", ids: []string{strings.ReplaceAll(catalogAccountID, "-", "")}, want: codes.Unauthenticated},
		{name: "owner mismatch", ids: []string{catalogAccountID}, owner: "another", want: codes.PermissionDenied},
		{name: "read", ids: []string{catalogAccountID}, access: catalogAccess(accountaccess.AccessLevelRead), want: codes.OK, storeCalls: 1},
		{name: "read write", ids: []string{catalogAccountID}, access: catalogAccess(accountaccess.AccessLevelReadWrite), want: codes.OK, storeCalls: 1},
		{name: "none", ids: []string{catalogAccountID}, access: catalogAccess(accountaccess.AccessLevelNone), want: codes.PermissionDenied, storeCalls: 1},
		{name: "read cannot write", ids: []string{catalogAccountID}, owner: catalogAccountID, access: catalogAccess(accountaccess.AccessLevelRead), required: accountaccess.AccessLevelReadWrite, want: codes.PermissionDenied, storeCalls: 1},
		{name: "write", ids: []string{catalogAccountID}, owner: catalogAccountID, access: catalogAccess(accountaccess.AccessLevelReadWrite), required: accountaccess.AccessLevelReadWrite, want: codes.OK, storeCalls: 1},
		{name: "invalid matrix", ids: []string{catalogAccountID}, access: accountaccess.Access{LoginEnabled: true}, want: codes.Internal, storeCalls: 1},
		{name: "missing row", ids: []string{catalogAccountID}, err: pgx.ErrNoRows, want: codes.PermissionDenied, storeCalls: 1},
		{name: "not found", ids: []string{catalogAccountID}, err: status.Error(codes.NotFound, "missing"), want: codes.PermissionDenied, storeCalls: 1},
		{name: "invalid persisted matrix from SQL reader", ids: []string{catalogAccountID}, err: fmt.Errorf("invalid persisted access: %w", status.Error(codes.InvalidArgument, "missing module")), want: codes.Internal, storeCalls: 1},
		{name: "database failure", ids: []string{catalogAccountID}, err: errors.New("offline"), want: codes.Unavailable, storeCalls: 1},
		{name: "cancel", ids: []string{catalogAccountID}, err: context.Canceled, want: codes.Canceled, storeCalls: 1},
		{name: "deadline", ids: []string{catalogAccountID}, err: context.DeadlineExceeded, want: codes.DeadlineExceeded, storeCalls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			s := &Service{accountAccessReader: accountAccessFunc(func(_ context.Context, id string) (accountaccess.Access, error) {
				calls++
				require.Equal(t, catalogAccountID, id)
				return tc.access, tc.err
			})}
			ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{AccountIDMetadataKey: tc.ids})
			required := tc.required
			if required == "" {
				required = accountaccess.AccessLevelRead
			}
			id, err := s.authorizeCatalogAccount(ctx, tc.owner, required)
			require.Equal(t, tc.want, status.Code(err))
			require.Equal(t, tc.storeCalls, calls)
			if err == nil {
				require.Equal(t, catalogAccountID, id)
			}
		})
	}
}
