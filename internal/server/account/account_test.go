package account

import (
	"context"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	accountpkg "github.com/useryege/athena/pkg/apiclient/account"
	"github.com/useryege/athena/util/settings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func contextWithSubject(subject string) context.Context {
	return context.WithValue(context.Background(), "claims", jwt.MapClaims{"sub": subject, "iss": "athena"})
}

func newVisibilityTestServer(t *testing.T) *Server {
	t.Helper()

	originalEnv := os.Environ()
	os.Clearenv()
	t.Cleanup(func() {
		os.Clearenv()
		for _, item := range originalEnv {
			key, value, ok := strings.Cut(item, "=")
			if ok {
				_ = os.Setenv(key, value)
			}
		}
	})

	env := map[string]string{
		"ATHENA_JWT_SECRET":                   "test-secret",
		"ATHENA_ADMIN_TOKENS":                 `[{"id":"admin-token","iat":30,"exp":300}]`,
		"ATHENA_ACCOUNT_LINGJIE_ENABLED":      "true",
		"ATHENA_ACCOUNT_LINGJIE_CAPABILITIES": string(settings.AccountCapabilityLogin),
		"ATHENA_ACCOUNT_LINGJIE_TOKENS":       `[{"id":"lingjie-token","iat":20,"exp":200}]`,
		"ATHENA_ACCOUNT_YUDIAN_ENABLED":       "true",
		"ATHENA_ACCOUNT_YUDIAN_CAPABILITIES":  string(settings.AccountCapabilityLogin),
		"ATHENA_ACCOUNT_YUDIAN_TOKENS":        `[{"id":"yudian-token","iat":10,"exp":100}]`,
	}
	for key, value := range env {
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("set %s: %v", key, err)
		}
	}

	settingsMgr, err := settings.NewSettingsManagerFromEnv(context.Background())
	if err != nil {
		t.Fatalf("NewSettingsManagerFromEnv: %v", err)
	}
	return NewServer(nil, settingsMgr, nil)
}

func accountNames(items []*accountpkg.Account) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	sort.Strings(names)
	return names
}

func assertNames(t *testing.T, got []*accountpkg.Account, want []string) {
	t.Helper()
	if names := accountNames(got); !reflect.DeepEqual(names, want) {
		t.Fatalf("account names = %v, want %v", names, want)
	}
}

func TestListAccountsAdminSeesAllAccounts(t *testing.T) {
	resp, err := newVisibilityTestServer(t).ListAccounts(contextWithSubject("admin"), &accountpkg.ListAccountRequest{})
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	assertNames(t, resp.Items, []string{"LINGJIE", "YUDIAN", "admin"})
}

func TestListAccountsRegularUserSeesSelfAndAdmin(t *testing.T) {
	resp, err := newVisibilityTestServer(t).ListAccounts(contextWithSubject("LINGJIE"), &accountpkg.ListAccountRequest{})
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	assertNames(t, resp.Items, []string{"LINGJIE", "admin"})

	for _, item := range resp.Items {
		if item.Name == "admin" && len(item.Tokens) != 0 {
			t.Fatalf("regular user received admin tokens: %#v", item.Tokens)
		}
		if item.Name == "LINGJIE" && len(item.Tokens) != 1 {
			t.Fatalf("regular user self tokens = %d, want 1", len(item.Tokens))
		}
	}
}

func TestGetAccountVisibilityRules(t *testing.T) {
	server := newVisibilityTestServer(t)

	adminForRegular, err := server.GetAccount(contextWithSubject("LINGJIE"), &accountpkg.GetAccountRequest{Name: "admin"})
	if err != nil {
		t.Fatalf("GetAccount admin as regular user: %v", err)
	}
	if len(adminForRegular.Tokens) != 0 {
		t.Fatalf("regular user received admin tokens: %#v", adminForRegular.Tokens)
	}

	self, err := server.GetAccount(contextWithSubject("LINGJIE"), &accountpkg.GetAccountRequest{Name: "LINGJIE"})
	if err != nil {
		t.Fatalf("GetAccount self: %v", err)
	}
	if len(self.Tokens) != 1 || self.Tokens[0].Id != "lingjie-token" {
		t.Fatalf("self tokens = %#v, want lingjie-token", self.Tokens)
	}

	_, err = server.GetAccount(contextWithSubject("LINGJIE"), &accountpkg.GetAccountRequest{Name: "YUDIAN"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("GetAccount other regular user code = %v, want PermissionDenied (err=%v)", status.Code(err), err)
	}

	other, err := server.GetAccount(contextWithSubject("admin"), &accountpkg.GetAccountRequest{Name: "YUDIAN"})
	if err != nil {
		t.Fatalf("GetAccount other as admin: %v", err)
	}
	if len(other.Tokens) != 1 || other.Tokens[0].Id != "yudian-token" {
		t.Fatalf("admin view tokens = %#v, want yudian-token", other.Tokens)
	}
}
