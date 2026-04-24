package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/pkg/apiclient/account"
	"github.com/useryege/athena/pkg/apiclient/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// "testing"
	// "github.com/spf13/cobra"
	// "github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify/require"
	// "google.golang.org/grpc/codes"
	// "google.golang.org/grpc/status"
	// "github.com/useryege/athena/v3/pkg/apiclient/account"
	// "github.com/useryege/athena/v3/pkg/apiclient/session"
	accountFixture "github.com/useryege/athena/test/e2e/fixture/account"
	// "github.com/useryege/athena/v3/util/errors"
	. "github.com/useryege/athena/test/e2e/fixture"

	utilio "github.com/useryege/athena/util/io"
)

func TestCreateAndUseAccount(t *testing.T) {
	ctx := accountFixture.Given(t)
	ctx.
		Name("test").
		When().
		Create().
		Then().
		And(func(account *account.Account, _ error) {
			assert.Equal(t, ctx.GetName(), account.Name)
			assert.Equal(t, []string{"login"}, account.Capabilities)
		}).
		When().
		Login().
		Then().
		CurrentUser(func(user *session.GetUserInfoResponse, _ error) {
			assert.True(t, user.LoggedIn)
			assert.Equal(t, user.Username, ctx.GetName())
		})
}

func TestCanIGetLogsAllow(t *testing.T) {
	ctx := accountFixture.Given(t)
	ctx.
		Name("test").
		When().
		Create().
		Login().
		SetPermissions([]ACL{
			{
				Resource: "logs",
				Action:   "get",
				Scope:    ProjectName + "/*",
			},
			{
				Resource: "apps",
				Action:   "get",
				Scope:    ProjectName + "/*",
			},
		}, "log-viewer").
		CanIGetLogs().
		Then().
		CanIResult(func(response *account.CanIResponse, err error) {
			assert.NoError(t, err)
			if assert.NotNil(t, response) {
				assert.Equal(t, "no", response.Value)
			}
		})
}

func TestCanIGetLogsDeny(t *testing.T) {
	ctx := accountFixture.Given(t)
	ctx.
		Name("test").
		When().
		Create().
		Login().
		CanIGetLogs().
		Then().
		CanIResult(func(response *account.CanIResponse, err error) {
			assert.NoError(t, err)
			if assert.NotNil(t, response) {
				assert.Equal(t, "no", response.Value)
			}
		})
}

func TestLoginBadCredentials(t *testing.T) {
	EnsureCleanState(t)

	closer, sessionClient := AthenaClientset.NewSessionClientOrDie()
	defer utilio.Close(closer)

	requests := []session.SessionCreateRequest{{
		Username: "user-does-not-exist", Password: "some-password",
	}, {
		Username: "admin", Password: "bad-password",
	}}

	for _, r := range requests {
		_, err := sessionClient.Create(t.Context(), &r)
		require.Error(t, err)
		errStatus, ok := status.FromError(err)
		if !assert.True(t, ok) {
			return
		}
		assert.Equal(t, codes.Unauthenticated, errStatus.Code())
		assert.Equal(t, "Invalid username or password", errStatus.Message())
	}
}
