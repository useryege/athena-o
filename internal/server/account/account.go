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
	accountaccesscore "github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/pkg/apiclient/account"
	"github.com/useryege/athena/util/password"
	"github.com/useryege/athena/util/session"
	"github.com/useryege/athena/util/settings"
)

// Server provides the Account service.
type Server struct {
	sessionMgr       *session.SessionManager
	settingsMgr      *settings.SettingsManager
	accessController *accountaccesscore.Controller
}

// NewServer returns a new Account service.
func NewServer(sessionMgr *session.SessionManager, settingsMgr *settings.SettingsManager, accessController *accountaccesscore.Controller) *Server {
	return &Server{
		sessionMgr:       sessionMgr,
		settingsMgr:      settingsMgr,
		accessController: accessController,
	}
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
	if updatedUsername != username {
		if err := s.accessController.Authorize(username, accountaccesscore.RequirementAdministrator); err != nil {
			return nil, err
		}
	}

	if updatedUsername == username {
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

func toAPIDataAccess(dataAccess accountaccesscore.DataAccess) account.AccountDataAccess {
	switch dataAccess {
	case accountaccesscore.DataAccessRead:
		return account.AccountDataAccess_ACCOUNT_DATA_ACCESS_READ
	case accountaccesscore.DataAccessReadWrite:
		return account.AccountDataAccess_ACCOUNT_DATA_ACCESS_READ_WRITE
	default:
		return account.AccountDataAccess_ACCOUNT_DATA_ACCESS_NONE
	}
}

func fromAPIDataAccess(dataAccess account.AccountDataAccess) (accountaccesscore.DataAccess, error) {
	switch dataAccess {
	case account.AccountDataAccess_ACCOUNT_DATA_ACCESS_NONE:
		return accountaccesscore.DataAccessNone, nil
	case account.AccountDataAccess_ACCOUNT_DATA_ACCESS_READ:
		return accountaccesscore.DataAccessRead, nil
	case account.AccountDataAccess_ACCOUNT_DATA_ACCESS_READ_WRITE:
		return accountaccesscore.DataAccessReadWrite, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "unsupported account data access %d", dataAccess)
	}
}

func toAPIAccount(name string, a settings.Account, access accountaccesscore.Access) *account.Account {
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
		Name:          name,
		Administrator: name == common.AthenaAdminUsername,
		Access: &account.AccountAccess{
			LoginEnabled: access.LoginEnabled,
			DataAccess:   toAPIDataAccess(access.DataAccess),
			Revision:     access.Revision,
		},
		Capabilities: capabilities,
		Tokens:       tokens,
	}
}

func canViewAccount(ctx context.Context, name string) bool {
	id := session.GetUserIdentifier(ctx)
	if id == common.AthenaAdminUsername {
		return true
	}
	return name == id
}

func (s *Server) accountForViewer(name string, a settings.Account) (*account.Account, error) {
	access, err := s.accessController.Get(name)
	if err != nil {
		return nil, err
	}
	return toAPIAccount(name, a, access), nil
}

func (s *Server) ensureCanManageAccount(ctx context.Context, account string) error {
	id := session.GetUserIdentifier(ctx)

	// An account can always manage its own self-service resources.
	if id == account {
		return nil
	}
	return s.accessController.Authorize(id, accountaccesscore.RequirementAdministrator)
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
			apiAccount, err := s.accountForViewer(name, a)
			if err != nil {
				return nil, fmt.Errorf("failed to get access for account %s: %w", name, err)
			}
			resp.Items = append(resp.Items, apiAccount)
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
		if err := s.accessController.Authorize(session.GetUserIdentifier(ctx), accountaccesscore.RequirementAdministrator); err != nil {
			return nil, err
		}
	}
	a, err := s.settingsMgr.GetAccount(r.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get account %s: %w", r.Name, err)
	}
	return s.accountForViewer(r.Name, *a)
}

// UpdateAccountAccess replaces a non-administrator account's complete access state.
func (s *Server) UpdateAccountAccess(ctx context.Context, r *account.UpdateAccountAccessRequest) (*account.Account, error) {
	if err := s.accessController.Authorize(session.GetUserIdentifier(ctx), accountaccesscore.RequirementAdministrator); err != nil {
		return nil, err
	}
	configuredAccount, err := s.settingsMgr.GetAccount(r.Name)
	if err != nil {
		return nil, err
	}
	if r.Access == nil {
		return nil, status.Error(codes.InvalidArgument, "account access is required")
	}
	dataAccess, err := fromAPIDataAccess(r.Access.DataAccess)
	if err != nil {
		return nil, err
	}
	updatedAccess, err := s.accessController.Update(ctx, r.Name, accountaccesscore.Access{
		LoginEnabled: r.Access.LoginEnabled,
		DataAccess:   dataAccess,
		Revision:     r.Access.Revision,
	}, r.Access.Revision)
	if err != nil {
		return nil, err
	}
	return toAPIAccount(r.Name, *configuredAccount, updatedAccess), nil
}

// CreateToken creates a token
func (s *Server) CreateToken(ctx context.Context, r *account.CreateTokenRequest) (*account.CreateTokenResponse, error) {
	if err := s.ensureCanManageAccount(ctx, r.Name); err != nil {
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
	if err := s.ensureCanManageAccount(ctx, r.Name); err != nil {
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
