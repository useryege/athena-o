//go:build integration

package account

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	api "github.com/useryege/athena/pkg/apiclient/account"
	"github.com/useryege/athena/util/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Public documentation retirement must not change persistent ordinary API keys,
// including keys named by the former browser convenience workflow.
func TestDocumentationRetirementPreservesAPIKeyLifecycle(t *testing.T) {
	ctx := context.Background()
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	store := accountstore.NewSQLStore(db.Pool)
	codec, err := accountcredentials.NewJWTCodec([]byte(strings.Repeat("retirement-key", 4)))
	require.NoError(t, err)
	credentials, err := accountcredentials.NewCredentialManager(ctx, store, codec)
	require.NoError(t, err)
	ids := map[string]string{}
	for _, name := range []string{"member", "other", "admin"} {
		realm := accountcredentials.ApplicationRealmMember
		if name == "admin" {
			realm = accountcredentials.ApplicationRealmAdmin
		}
		account, _, err := credentials.RegisterExternalAccount(ctx, accountcredentials.IdentityProviderGoogle, name+"-retirement", name+"@example.test", name+"-retirement", realm)
		require.NoError(t, err)
		ids[name] = account.ID
	}
	access, err := accountaccess.NewController(ctx, store)
	require.NoError(t, err)
	for _, name := range []string{"member", "other"} {
		value, err := access.Get(ids[name])
		require.NoError(t, err)
		value.LoginEnabled, value.APIKeyEnabled = true, true
		_, err = access.Update(ctx, ids[name], value, value.Revision)
		require.NoError(t, err)
	}
	accountContext := func(name string) context.Context {
		return context.WithValue(ctx, "claims", jwt.MapClaims{"sub": ids[name]}) //nolint:staticcheck
	}
	server := NewServer(credentials, access, nil, nil)
	key, err := server.CreateToken(accountContext("member"), &api.CreateTokenRequest{Id: "ai-existing", ExpiresIn: 3600})
	require.NoError(t, err)
	require.NotEmpty(t, key.Token)
	// Reload the real persistent registry: pre-existing ai- metadata survives.
	credentials, err = accountcredentials.NewCredentialManager(ctx, store, codec)
	require.NoError(t, err)
	server = NewServer(credentials, access, nil, nil)
	sessions := session.NewSessionManager(credentials, codec, session.NewUserStateStorage(nil), access)
	authenticate := func() error {
		_, credential, err := sessions.AuthenticateToken(key.Token)
		if err == nil {
			require.Equal(t, ids["member"], credential.AccountID)
			require.Equal(t, accountcredentials.CapabilityAPIKey, credential.Capability)
		}
		return err
	}
	require.NoError(t, authenticate())
	listed, err := server.ListTokens(accountContext("member"), &api.ListTokensRequest{})
	require.NoError(t, err)
	require.Len(t, listed.Items, 1)
	require.Equal(t, "ai-existing", listed.Items[0].Id)
	require.Equal(t, int64(3600), listed.Items[0].ExpiresAt-listed.Items[0].IssuedAt)
	other, err := server.ListTokens(accountContext("other"), &api.ListTokensRequest{})
	require.NoError(t, err)
	require.Empty(t, other.Items)
	_, err = server.DeleteToken(accountContext("other"), &api.DeleteTokenRequest{Id: "ai-existing"})
	require.Error(t, err)
	require.NoError(t, authenticate())
	_, err = server.CreateToken(accountContext("admin"), &api.CreateTokenRequest{Id: "forbidden"})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	_, err = server.CreateToken(accountContext("member"), &api.CreateTokenRequest{Id: "bad id"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	_, err = server.CreateToken(accountContext("member"), &api.CreateTokenRequest{Id: "negative", ExpiresIn: -1})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	for _, field := range []string{"api-key", "login"} {
		value, err := access.Get(ids["member"])
		require.NoError(t, err)
		if field == "api-key" {
			value.APIKeyEnabled = false
		} else {
			value.LoginEnabled = false
		}
		value, err = access.Update(ctx, ids["member"], value, value.Revision)
		require.NoError(t, err)
		require.Error(t, authenticate(), field)
		_, err = server.CreateToken(accountContext("member"), &api.CreateTokenRequest{Id: "disabled"})
		require.Equal(t, codes.PermissionDenied, status.Code(err))
		value.LoginEnabled, value.APIKeyEnabled = true, true
		_, err = access.Update(ctx, ids["member"], value, value.Revision)
		require.NoError(t, err)
		require.NoError(t, authenticate(), "re-enabling permission preserves the key")
	}
	// A correctly signed but expired key is rejected even with live metadata.
	expired, metadata, err := codec.Issue(ids["member"], accountcredentials.CapabilityAPIKey, "expired-retirement", 60, time.Now().Add(-time.Hour), "")
	require.NoError(t, err)
	metadata.ID = "expired"
	require.NoError(t, store.CreateAPIKeyMetadata(ctx, ids["member"], metadata))
	loaded, err := accountcredentials.NewCredentialManager(ctx, store, codec)
	require.NoError(t, err)
	expiredSessions := session.NewSessionManager(loaded, codec, session.NewUserStateStorage(nil), access)
	_, _, err = expiredSessions.AuthenticateToken(expired)
	require.ErrorContains(t, err, "expired")
	_, err = server.DeleteToken(accountContext("member"), &api.DeleteTokenRequest{Id: "ai-existing"})
	require.NoError(t, err)
	require.Error(t, authenticate(), "revoked key must immediately stop authenticating")
	listed, err = server.ListTokens(accountContext("member"), &api.ListTokensRequest{})
	require.NoError(t, err)
	require.Empty(t, listed.Items)
}
