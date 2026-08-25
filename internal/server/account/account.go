package account

import (
	"context"
	"fmt"
	"net/url"
	"sort"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/common"
	accountaccesscore "github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcenter"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/pkg/apiclient/account"
	"github.com/useryege/athena/util/session"
)

// Server provides the Account service.
type Server struct {
	credentials      *accountcredentials.CredentialManager
	accessController *accountaccesscore.Controller
	accountCenter    *accountcenter.Manager
}

// NewServer returns a new Account service.
func NewServer(credentials *accountcredentials.CredentialManager, accessController *accountaccesscore.Controller, accountCenter *accountcenter.Manager) *Server {
	return &Server{
		credentials:      credentials,
		accessController: accessController,
		accountCenter:    accountCenter,
	}
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

func toAPIAccount(name string, a accountcredentials.Account, access accountaccesscore.Access, profile accountcenter.Profile) *account.Account {
	var capabilities []string
	for _, c := range a.Capabilities {
		capabilities = append(capabilities, string(c))
	}
	return &account.Account{
		Name:          name,
		Administrator: name == common.AthenaAdminUsername,
		Access:        ToAPIAccountAccess(access),
		Capabilities:  capabilities,
		Profile:       ToAPIAccountProfile(name, profile),
	}
}

// ToAPIAccountProfile is the canonical profile projection shared with raw avatar HTTP handlers.
func ToAPIAccountProfile(name string, profile accountcenter.Profile) *account.AccountProfile {
	tier := account.AccountTier_ACCOUNT_TIER_UNSPECIFIED
	switch profile.Tier {
	case accountcenter.TierStandard:
		tier = account.AccountTier_ACCOUNT_TIER_STANDARD
	case accountcenter.TierPro:
		tier = account.AccountTier_ACCOUNT_TIER_PRO
	}
	avatarURL := ""
	if !profile.Avatar.Empty() {
		avatarURL = fmt.Sprintf("/api/v1/account/%s/avatar?v=%d", url.PathEscape(name), profile.Revision)
	}
	return &account.AccountProfile{
		DisplayName: profile.DisplayName,
		Tier:        tier,
		AvatarUrl:   avatarURL,
		Revision:    profile.Revision,
	}
}

// ToAPIAccountPreferences projects the current account's private UI preferences.
func ToAPIAccountPreferences(preferences accountcenter.Preferences) *account.AccountPreferences {
	theme := account.AccountThemeMode_ACCOUNT_THEME_MODE_UNSPECIFIED
	switch preferences.Theme {
	case accountcenter.ThemeModeSystem:
		theme = account.AccountThemeMode_ACCOUNT_THEME_MODE_SYSTEM
	case accountcenter.ThemeModeLight:
		theme = account.AccountThemeMode_ACCOUNT_THEME_MODE_LIGHT
	case accountcenter.ThemeModeDark:
		theme = account.AccountThemeMode_ACCOUNT_THEME_MODE_DARK
	}
	return &account.AccountPreferences{Theme: theme, Revision: preferences.Revision}
}

func canViewAccount(ctx context.Context, name string) bool {
	id := session.GetUserIdentifier(ctx)
	if id == common.AthenaAdminUsername {
		return true
	}
	return name == id
}

func (s *Server) accountForViewer(ctx context.Context, name string, a accountcredentials.Account) (*account.Account, error) {
	access, err := s.accessController.Get(name)
	if err != nil {
		return nil, err
	}
	profile, err := s.accountCenter.GetProfile(ctx, name)
	if err != nil {
		return nil, err
	}
	return toAPIAccount(name, a, access, profile), nil
}

// ListAccounts returns the list of accounts
func (s *Server) ListAccounts(ctx context.Context, _ *account.ListAccountsRequest) (*account.AccountsList, error) {
	resp := account.AccountsList{}
	accounts := s.credentials.List()
	for name, a := range accounts {
		if canViewAccount(ctx, name) {
			apiAccount, err := s.accountForViewer(ctx, name, a)
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
	return s.accountForViewer(ctx, r.Name, a)
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
	profile, err := s.accountCenter.GetProfile(ctx, r.Name)
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
	return toAPIAccount(r.Name, configuredAccount, updatedAccess, profile), nil
}

func (s *Server) UpdateAccountProfile(ctx context.Context, r *account.UpdateAccountProfileRequest) (*account.AccountProfile, error) {
	if !canViewAccount(ctx, r.Name) {
		if err := s.accessController.Authorize(session.GetUserIdentifier(ctx), accountaccesscore.RequirementAdministrator); err != nil {
			return nil, err
		}
	}
	profile, err := s.accountCenter.UpdateDisplayName(ctx, r.Name, r.DisplayName, r.ExpectedRevision)
	if err != nil {
		return nil, err
	}
	return ToAPIAccountProfile(r.Name, profile), nil
}

func (s *Server) UpdateAccountTier(ctx context.Context, r *account.UpdateAccountTierRequest) (*account.AccountProfile, error) {
	if err := s.accessController.Authorize(session.GetUserIdentifier(ctx), accountaccesscore.RequirementAdministrator); err != nil {
		return nil, err
	}
	var tier accountcenter.Tier
	switch r.Tier {
	case account.AccountTier_ACCOUNT_TIER_STANDARD:
		tier = accountcenter.TierStandard
	case account.AccountTier_ACCOUNT_TIER_PRO:
		tier = accountcenter.TierPro
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported account tier %d", r.Tier)
	}
	profile, err := s.accountCenter.UpdateTier(ctx, r.Name, tier, r.ExpectedRevision)
	if err != nil {
		return nil, err
	}
	return ToAPIAccountProfile(r.Name, profile), nil
}

func (s *Server) UpdateAccountPreferences(ctx context.Context, r *account.UpdateAccountPreferencesRequest) (*account.AccountPreferences, error) {
	name := session.GetUserIdentifier(ctx)
	var theme accountcenter.ThemeMode
	switch r.Theme {
	case account.AccountThemeMode_ACCOUNT_THEME_MODE_SYSTEM:
		theme = accountcenter.ThemeModeSystem
	case account.AccountThemeMode_ACCOUNT_THEME_MODE_LIGHT:
		theme = accountcenter.ThemeModeLight
	case account.AccountThemeMode_ACCOUNT_THEME_MODE_DARK:
		theme = accountcenter.ThemeModeDark
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported account theme %d", r.Theme)
	}
	preferences, err := s.accountCenter.UpdatePreferences(ctx, name, theme, r.ExpectedRevision)
	if err != nil {
		return nil, err
	}
	return ToAPIAccountPreferences(preferences), nil
}

// ListTokens returns API Key metadata only for the current authenticated account.
func (s *Server) ListTokens(ctx context.Context, _ *account.ListTokensRequest) (*account.TokensList, error) {
	a, err := s.credentials.Get(session.GetUserIdentifier(ctx))
	if err != nil {
		return nil, err
	}
	tokens := make([]*account.Token, 0, len(a.Tokens))
	for _, token := range a.Tokens {
		tokens = append(tokens, &account.Token{Id: token.ID, ExpiresAt: token.ExpiresAt, IssuedAt: token.IssuedAt})
	}
	sort.Slice(tokens, func(i, j int) bool { return tokens[i].IssuedAt > tokens[j].IssuedAt })
	return &account.TokensList{Items: tokens}, nil
}

// CreateToken creates a token
func (s *Server) CreateToken(ctx context.Context, r *account.CreateTokenRequest) (*account.CreateTokenResponse, error) {
	name := session.GetUserIdentifier(ctx)
	id := r.Id
	if id == "" {
		uniqueId, err := uuid.NewRandom()
		if err != nil {
			return nil, fmt.Errorf("failed to generate unique ID: %w", err)
		}
		id = uniqueId.String()
	} else if !accountcredentials.IsValidAPIKeyDisplayID(id) {
		return nil, status.Error(codes.InvalidArgument, "API key ID must be 1-64 ASCII letters, digits, dots, underscores, or hyphens and start with a letter or digit")
	}

	tokenString, err := s.credentials.IssueAPIKey(name, id, r.ExpiresIn)
	if err != nil {
		return nil, fmt.Errorf("failed to update account with new token: %w", err)
	}
	return &account.CreateTokenResponse{Token: tokenString}, nil
}

// DeleteToken deletes a token
func (s *Server) DeleteToken(ctx context.Context, r *account.DeleteTokenRequest) (*account.EmptyResponse, error) {
	name := session.GetUserIdentifier(ctx)
	err := s.credentials.DeleteAPIKey(name, r.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete token from account %s: %w", name, err)
	}
	return &account.EmptyResponse{}, nil
}
