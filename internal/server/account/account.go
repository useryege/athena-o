package account

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/common"
	accountaccesscore "github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/pkg/apiclient/account"
	"github.com/useryege/athena/util/password"
	"github.com/useryege/athena/util/session"
)

// Server provides the Account service.
type Server struct {
	credentials      *accountcredentials.CredentialManager
	passwordPattern  string
	accessController *accountaccesscore.Controller
}

// NewServer returns a new Account service.
func NewServer(credentials *accountcredentials.CredentialManager, passwordPattern string, accessController *accountaccesscore.Controller) *Server {
	return &Server{
		credentials:      credentials,
		passwordPattern:  passwordPattern,
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

	// Need to validate password complexity with regular expression
	passwordPattern := s.passwordPattern
	if passwordPattern == "" {
		passwordPattern = common.PasswordPatten
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

	if updatedUsername == username {
		err = s.credentials.ChangePassword(updatedUsername, q.CurrentPassword, hashedPassword)
		if errors.Is(err, accountcredentials.ErrInvalidCredentials) {
			return nil, status.Error(codes.InvalidArgument, "current password does not match")
		}
	} else {
		err = s.credentials.ResetPassword(updatedUsername, hashedPassword)
	}
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

type accountDataModuleMapping struct {
	module accountaccesscore.Module
	api    account.AccountDataModule
}

var canonicalAccountDataModules = []accountDataModuleMapping{
	{module: accountaccesscore.ModuleMarketRadar, api: account.AccountDataModule_ACCOUNT_DATA_MODULE_MARKET_RADAR},
	{module: accountaccesscore.ModuleSportsLive, api: account.AccountDataModule_ACCOUNT_DATA_MODULE_SPORTS_LIVE},
	{module: accountaccesscore.ModuleSportsHistory, api: account.AccountDataModule_ACCOUNT_DATA_MODULE_SPORTS_HISTORY},
	{module: accountaccesscore.ModuleManagedOO, api: account.AccountDataModule_ACCOUNT_DATA_MODULE_MANAGED_OO},
	{module: accountaccesscore.ModuleWormMarkets, api: account.AccountDataModule_ACCOUNT_DATA_MODULE_WORM_MARKETS},
	{module: accountaccesscore.ModuleFIFAMarketDashboard, api: account.AccountDataModule_ACCOUNT_DATA_MODULE_FIFA_MARKET_DASHBOARD},
	{module: accountaccesscore.ModuleWorldCupCorners, api: account.AccountDataModule_ACCOUNT_DATA_MODULE_WORLD_CUP_CORNERS},
	{module: accountaccesscore.ModuleToken, api: account.AccountDataModule_ACCOUNT_DATA_MODULE_TOKEN},
	{module: accountaccesscore.ModuleWallet, api: account.AccountDataModule_ACCOUNT_DATA_MODULE_WALLET},
	{module: accountaccesscore.ModuleNotifications, api: account.AccountDataModule_ACCOUNT_DATA_MODULE_NOTIFICATIONS},
}

func toAPIDataAccess(dataAccess accountaccesscore.AccessLevel) account.AccountDataAccess {
	switch dataAccess {
	case accountaccesscore.AccessLevelRead:
		return account.AccountDataAccess_ACCOUNT_DATA_ACCESS_READ
	case accountaccesscore.AccessLevelReadWrite:
		return account.AccountDataAccess_ACCOUNT_DATA_ACCESS_READ_WRITE
	default:
		return account.AccountDataAccess_ACCOUNT_DATA_ACCESS_NONE
	}
}

func fromAPIDataAccess(dataAccess account.AccountDataAccess) (accountaccesscore.AccessLevel, error) {
	switch dataAccess {
	case account.AccountDataAccess_ACCOUNT_DATA_ACCESS_NONE:
		return accountaccesscore.AccessLevelNone, nil
	case account.AccountDataAccess_ACCOUNT_DATA_ACCESS_READ:
		return accountaccesscore.AccessLevelRead, nil
	case account.AccountDataAccess_ACCOUNT_DATA_ACCESS_READ_WRITE:
		return accountaccesscore.AccessLevelReadWrite, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "unsupported account data access %d", dataAccess)
	}
}

func fromAPIDataModule(dataModule account.AccountDataModule) (accountaccesscore.Module, error) {
	for _, mapping := range canonicalAccountDataModules {
		if mapping.api == dataModule {
			return mapping.module, nil
		}
	}
	return "", status.Errorf(codes.InvalidArgument, "unsupported account data module %d", dataModule)
}

func fromAPIModuleAccess(moduleAccess []*account.AccountModuleAccess) (map[accountaccesscore.Module]accountaccesscore.AccessLevel, error) {
	if len(moduleAccess) != len(canonicalAccountDataModules) {
		return nil, status.Errorf(codes.InvalidArgument, "account access must contain all %d data modules", len(canonicalAccountDataModules))
	}

	modules := make(map[accountaccesscore.Module]accountaccesscore.AccessLevel, len(canonicalAccountDataModules))
	for _, item := range moduleAccess {
		module, err := fromAPIDataModule(item.GetModule())
		if err != nil {
			return nil, err
		}
		if _, exists := modules[module]; exists {
			return nil, status.Errorf(codes.InvalidArgument, "account data module %q is duplicated", module)
		}
		accessLevel, err := fromAPIDataAccess(item.GetDataAccess())
		if err != nil {
			return nil, err
		}
		maximum, supported := accountaccesscore.MaxAccessLevel(module)
		if !supported {
			return nil, status.Errorf(codes.InvalidArgument, "unsupported account data module %q", module)
		}
		if maximum == accountaccesscore.AccessLevelRead && accessLevel == accountaccesscore.AccessLevelReadWrite {
			return nil, status.Errorf(codes.InvalidArgument, "account data module %q is read-only", module)
		}
		modules[module] = accessLevel
	}

	for _, mapping := range canonicalAccountDataModules {
		if _, exists := modules[mapping.module]; !exists {
			return nil, status.Errorf(codes.InvalidArgument, "account data module %q is required", mapping.module)
		}
	}
	return modules, nil
}

// ToAPIAccountAccess projects a complete access aggregate in stable module order.
func ToAPIAccountAccess(access accountaccesscore.Access) *account.AccountAccess {
	moduleAccess := make([]*account.AccountModuleAccess, 0, len(canonicalAccountDataModules))
	for _, mapping := range canonicalAccountDataModules {
		moduleAccess = append(moduleAccess, &account.AccountModuleAccess{
			Module:     mapping.api,
			DataAccess: toAPIDataAccess(access.Modules[mapping.module]),
		})
	}
	return &account.AccountAccess{
		LoginEnabled: access.LoginEnabled,
		Revision:     access.Revision,
		ModuleAccess: moduleAccess,
	}
}

func toAPIAccount(name string, a accountcredentials.Account, access accountaccesscore.Access) *account.Account {
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
		Access:        ToAPIAccountAccess(access),
		Capabilities:  capabilities,
		Tokens:        tokens,
	}
}

func canViewAccount(ctx context.Context, name string) bool {
	id := session.GetUserIdentifier(ctx)
	if id == common.AthenaAdminUsername {
		return true
	}
	return name == id
}

func (s *Server) accountForViewer(name string, a accountcredentials.Account) (*account.Account, error) {
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
	accounts := s.credentials.List()
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
	a, err := s.credentials.Get(r.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get account %s: %w", r.Name, err)
	}
	return s.accountForViewer(r.Name, a)
}

// UpdateAccountAccess replaces a non-administrator account's complete access state.
func (s *Server) UpdateAccountAccess(ctx context.Context, r *account.UpdateAccountAccessRequest) (*account.Account, error) {
	if err := s.accessController.Authorize(session.GetUserIdentifier(ctx), accountaccesscore.RequirementAdministrator); err != nil {
		return nil, err
	}
	configuredAccount, err := s.credentials.Get(r.Name)
	if err != nil {
		return nil, err
	}
	if r.Access == nil {
		return nil, status.Error(codes.InvalidArgument, "account access is required")
	}
	modules, err := fromAPIModuleAccess(r.Access.ModuleAccess)
	if err != nil {
		return nil, err
	}
	updatedAccess, err := s.accessController.Update(ctx, r.Name, accountaccesscore.Access{
		LoginEnabled: r.Access.LoginEnabled,
		Modules:      modules,
		Revision:     r.Access.Revision,
	}, r.Access.Revision)
	if err != nil {
		return nil, err
	}
	return toAPIAccount(r.Name, configuredAccount, updatedAccess), nil
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

	tokenString, err := s.credentials.IssueAPIKey(r.Name, id, r.ExpiresIn)
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

	err := s.credentials.DeleteAPIKey(r.Name, r.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete account %s: %w", r.Name, err)
	}
	return &account.EmptyResponse{}, nil
}
