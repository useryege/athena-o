package account

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/pkg/apiclient/account"
	"github.com/useryege/athena/util/password"
	"github.com/useryege/athena/util/rbac"
	"github.com/useryege/athena/util/session"
	"github.com/useryege/athena/util/settings"
)

// Server provides a Session service
type Server struct {
	sessionMgr  *session.SessionManager
	settingsMgr *settings.SettingsManager
	enf         *rbac.Enforcer
}

// NewServer returns a new instance of the Session service
func NewServer(sessionMgr *session.SessionManager, settingsMgr *settings.SettingsManager, enf *rbac.Enforcer) *Server {
	return &Server{sessionMgr, settingsMgr, enf}
}

// UpdatePassword updates the password of the currently authenticated account or the account specified in the request.
func (s *Server) UpdatePassword(ctx context.Context, q *account.UpdatePasswordRequest) (*account.UpdatePasswordResponse, error) {
	// get the user identifier from the context
	username := session.GetUserIdentifier(ctx)

	updatedUsername := username
	if q.Name != "" {
		updatedUsername = q.Name
	}

	// check for permission is user is trying to change someone else's password
	// assuming user is trying to update someone else if username is different or issuer is not Athena
	issuer := session.Iss(ctx)
	if updatedUsername != username || issuer != session.SessionManagerClaimsIssuer {
		if err := s.enf.EnforceErr(ctx.Value("claims"), rbac.ResourceAccounts, rbac.ActionUpdate, q.Name); err != nil {
			return nil, fmt.Errorf("permission denied: %w", err)
		}
	}

	if updatedUsername == username && issuer == session.SessionManagerClaimsIssuer {
		err := s.sessionMgr.VerifyUsernamePassword(username, q.CurrentPassword)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "current password does not match")
		}
	}

	// Need to validate password complexity with regular expression
	passwordPattern, err := s.settingsMgr.GetPasswordPattern()
	if err != nil {
		return nil, fmt.Errorf("failed to get password pattern: %w", err)
	}

	validPasswordRegexp, err := regexp.Compile(passwordPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile password regex: %w", err)
	}

	if !validPasswordRegexp.MatchString(q.NewPassword) {
		err := fmt.Errorf("new password does not match the following expression: %s", passwordPattern)
		return nil, err
	}

	hashedPassword, err := password.HashPassword(q.NewPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	err = s.settingsMgr.UpdateAccount(updatedUsername, func(acc *settings.Account) error {
		acc.PasswordHash = hashedPassword
		now := time.Now().UTC()
		acc.PasswordMtime = &now
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update account password: %w", err)
	}

	if updatedUsername == username {
		log.Infof("user '%s' updated password", username)
	} else {
		log.Infof("user '%s' updated password of user '%s'", username, updatedUsername)
	}
	return &account.UpdatePasswordResponse{}, nil
}

func toAPIAccount(name string, a settings.Account) *account.Account {
	var capabilities []string
	for _, c := range a.Capabilities {
		capabilities = append(capabilities, string(c))
	}
	var tokens []*account.Token
	for _, t := range a.Tokens {
		tokens = append(tokens, &account.Token{Id: t.ID, ExpiresAt: t.ExpiresAt, IssuedAt: t.IssuedAt})
	}
	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i].IssuedAt > tokens[j].IssuedAt
	})
	return &account.Account{
		Name:         name,
		Enabled:      a.Enabled,
		Capabilities: capabilities,
		Tokens:       tokens,
	}
}

func canViewAccount(ctx context.Context, name string) bool {
	id := session.GetUserIdentifier(ctx)
	if id == common.AthenaAdminUsername {
		return true
	}
	return name == id || name == common.AthenaAdminUsername
}

func accountForViewer(ctx context.Context, name string, a settings.Account) *account.Account {
	apiAccount := toAPIAccount(name, a)
	if session.GetUserIdentifier(ctx) != common.AthenaAdminUsername && name == common.AthenaAdminUsername {
		apiAccount.Tokens = nil
	}
	return apiAccount
}

func (s *Server) ensureHasAccountPermission(ctx context.Context, action string, account string) error {
	id := session.GetUserIdentifier(ctx)

	// account has always has access to itself
	if id == account && session.Iss(ctx) == session.SessionManagerClaimsIssuer {
		return nil
	}
	if err := s.enf.EnforceErr(ctx.Value("claims"), rbac.ResourceAccounts, action, account); err != nil {
		return fmt.Errorf("permission denied for account %s with action %s: %w", account, action, err)
	}
	return nil
}

// ListAccounts returns the list of accounts
func (s *Server) ListAccounts(ctx context.Context, _ *account.ListAccountRequest) (*account.AccountsList, error) {
	resp := account.AccountsList{}
	accounts, err := s.settingsMgr.GetAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}
	for name, a := range accounts {
		if canViewAccount(ctx, name) {
			resp.Items = append(resp.Items, accountForViewer(ctx, name, a))
		}
	}
	sort.Slice(resp.Items, func(i, j int) bool {
		return resp.Items[i].Name < resp.Items[j].Name
	})
	return &resp, nil
}

// GetAccount returns an account
func (s *Server) GetAccount(ctx context.Context, r *account.GetAccountRequest) (*account.Account, error) {
	if !canViewAccount(ctx, r.Name) {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied to get account %s", r.Name)
	}
	a, err := s.settingsMgr.GetAccount(r.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get account %s: %w", r.Name, err)
	}
	return accountForViewer(ctx, r.Name, *a), nil
}

// CreateToken creates a token
func (s *Server) CreateToken(ctx context.Context, r *account.CreateTokenRequest) (*account.CreateTokenResponse, error) {
	if err := s.ensureHasAccountPermission(ctx, rbac.ActionUpdate, r.Name); err != nil {
		return nil, fmt.Errorf("permission denied to create token for account %s: %w", r.Name, err)
	}

	id := r.Id
	if id == "" {
		uniqueId, err := uuid.NewRandom()
		if err != nil {
			return nil, fmt.Errorf("failed to generate unique ID: %w", err)
		}
		id = uniqueId.String()
	}

	var tokenString string
	err := s.settingsMgr.UpdateAccount(r.Name, func(account *settings.Account) error {
		if account.TokenIndex(id) > -1 {
			return fmt.Errorf("account already has token with id '%s'", id)
		}
		if !account.HasCapability(settings.AccountCapabilityApiKey) {
			return fmt.Errorf("account '%s' does not have %s capability", r.Name, settings.AccountCapabilityApiKey)
		}

		now := time.Now()
		var err error
		tokenString, err = s.sessionMgr.Create(fmt.Sprintf("%s:%s", r.Name, settings.AccountCapabilityApiKey), r.ExpiresIn, id)
		if err != nil {
			return err
		}

		var expiresAt int64
		if r.ExpiresIn > 0 {
			expiresAt = now.Add(time.Duration(r.ExpiresIn) * time.Second).Unix()
		}
		account.Tokens = append(account.Tokens, settings.Token{
			ID:        id,
			IssuedAt:  now.Unix(),
			ExpiresAt: expiresAt,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update account with new token: %w", err)
	}
	return &account.CreateTokenResponse{Token: tokenString}, nil
}

// DeleteToken deletes a token
func (s *Server) DeleteToken(ctx context.Context, r *account.DeleteTokenRequest) (*account.EmptyResponse, error) {
	if err := s.ensureHasAccountPermission(ctx, rbac.ActionUpdate, r.Name); err != nil {
		return nil, fmt.Errorf("permission denied to delete account %s: %w", r.Name, err)
	}

	err := s.settingsMgr.UpdateAccount(r.Name, func(account *settings.Account) error {
		if index := account.TokenIndex(r.Id); index > -1 {
			account.Tokens = append(account.Tokens[:index], account.Tokens[index+1:]...)
			return nil
		}
		return status.Errorf(codes.NotFound, "token with id '%s' does not exist", r.Id)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to delete account %s: %w", r.Name, err)
	}
	return &account.EmptyResponse{}, nil
}
