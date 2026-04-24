package account

import (
	"context"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/useryege/athena/pkg/apiclient/account"
	"github.com/useryege/athena/test/e2e/fixture"
	utilio "github.com/useryege/athena/util/io"
)

// this implements the "when" part of given/when/then
//
// none of the func implement error checks, and that is complete intended, you should check for errors
// using the Then()
type Actions struct {
	context    *Context
	lastOutput string
	lastError  error
	lastCanI   *account.CanIResponse
}

// func (a *Actions) prepareCanIGetLogsArgs() []string {
// 	a.context.T().Helper()
// 	return []string{
// 		"account", "can-i", "get", "logs", a.context.project + "/*",
// 	}
// }

func (a *Actions) CanIGetLogs() *Actions {
	a.context.T().Helper()
	// TODO: implement this
	// a.runCli(a.prepareCanIGetLogsArgs()...)

	closer, accountClient, err := fixture.AthenaClientset.NewAccountClient()
	require.NoError(a.context.T(), err)
	defer utilio.Close(closer)
	canIResponse, err := accountClient.CanI(context.Background(), &account.CanIRequest{
		Resource:    "logs",
		Action:      "get",
		Subresource: "*/*",
	})
	a.lastCanI = canIResponse
	a.lastError = err
	if canIResponse != nil {
		a.lastOutput = canIResponse.Value
	} else {
		a.lastOutput = ""
	}

	return a
}

// func (a *Actions) prepareSetPasswordArgs(account string) []string {
// 	a.context.T().Helper()
// 	return []string{
// 		"account", "update-password", "--account", account, "--current-password", fixture.AdminPassword, "--new-password", fixture.DefaultTestUserPassword,
// 	}
// }

func (a *Actions) Create() *Actions {
	a.context.T().Helper()
	require.NoError(a.context.T(), fixture.SetAccounts(map[string][]string{
		a.context.GetName(): {"login"},
	}))
	// TODO: implement this
	// _, _ = fixture.RunCli(a.prepareSetPasswordArgs(a.context.GetName())...)

	closer, accountClient, err := fixture.AthenaClientset.NewAccountClient()
	require.NoError(a.context.T(), err)
	defer utilio.Close(closer)
	_, err = accountClient.UpdatePassword(context.Background(), &account.UpdatePasswordRequest{
		Name:            a.context.GetName(),
		CurrentPassword: fixture.AdminPassword,
		NewPassword:     fixture.DefaultTestUserPassword,
	})
	require.NoError(a.context.T(), err)

	return a
}

func (a *Actions) SetPermissions(permissions []fixture.ACL, roleName string) *Actions {
	a.context.T().Helper()
	require.NoError(a.context.T(), fixture.SetPermissions(permissions, a.context.GetName(), roleName))
	return a
}

func (a *Actions) SetParamInSettingConfigMap(key, value string) *Actions {
	a.context.T().Helper()
	require.NoError(a.context.T(), fixture.SetParamInSettingConfigMap(key, value))
	return a
}

func (a *Actions) Login() *Actions {
	a.context.T().Helper()
	require.NoError(a.context.T(), fixture.LoginAs(a.context.GetName()))
	return a
}

func (a *Actions) Then() *Consequences {
	a.context.T().Helper()
	time.Sleep(fixture.WhenThenSleepInterval)
	return &Consequences{a.context, a}
}
