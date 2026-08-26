package account

import (
	"context"
	"errors"
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
	directory        AccountDirectory
}

// AccountDirectory provides the server-side filtered durable account index.
type AccountDirectory interface {
	ListAccountDirectory(ctx context.Context, query, status string, page, pageSize int32, profitSharingEligibleOnly bool) ([]string, int64, error)
}

// NewServer returns a new Account service.
func NewServer(credentials *accountcredentials.CredentialManager, accessController *accountaccesscore.Controller, accountCenter *accountcenter.Manager, directory AccountDirectory) *Server {
	return &Server{
		credentials:      credentials,
		accessController: accessController,
		accountCenter:    accountCenter,
		directory:        directory,
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
		LoginEnabled:         access.LoginEnabled,
		Revision:             access.Revision,
		ModuleAccess:         moduleAccess,
		ApiKeyEnabled:        access.APIKeyEnabled,
		ProfitSharingEnabled: access.ProfitSharingEnabled,
	}
}

func toAPIAccount(name string, a accountcredentials.Account, access accountaccesscore.Access, profile accountcenter.Profile) *account.Account {
	return &account.Account{
		Name:          name,
		Administrator: a.Administrator,
		Access:        ToAPIAccountAccess(access),
		Profile:       ToAPIAccountProfile(name, profile),
		Identity:      ToAPIAccountIdentity(a),
		Status:        toAPIAccountStatus(access),
	}
}

// ToAPIAccountIdentity projects safe Google identity metadata without its subject.
func ToAPIAccountIdentity(a accountcredentials.Account) *account.AccountIdentity {
	identity := &account.AccountIdentity{
		VerifiedEmail: a.VerifiedEmail,
	}
	if a.HasGoogleBinding() {
		identity.Provider = account.AccountIdentityProvider_ACCOUNT_IDENTITY_PROVIDER_GOOGLE
	}
	if !a.CreatedAt.IsZero() {
		identity.CreatedAt = a.CreatedAt.Unix()
	}
	if !a.LastLoginAt.IsZero() {
		identity.LastLoginAt = a.LastLoginAt.Unix()
	}
	return identity
}

func toAPIAccountStatus(access accountaccesscore.Access) account.AccountStatus {
	if !access.LoginEnabled {
		return account.AccountStatus_ACCOUNT_STATUS_BLOCKED
	}
	if access.IsPending() {
		return account.AccountStatus_ACCOUNT_STATUS_PENDING
	}
	return account.AccountStatus_ACCOUNT_STATUS_ACTIVE
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

// ListAccounts returns a server-side filtered account directory for the
// administrator, while ordinary users can project only themselves.
func (s *Server) ListAccounts(ctx context.Context, r *account.ListAccountsRequest) (*account.AccountsList, error) {
	viewer := session.GetUserIdentifier(ctx)
	if viewer != common.AthenaAdminUsername {
		a, err := s.credentials.Get(viewer)
		if err != nil {
			return nil, err
		}
		projected, err := s.accountForViewer(ctx, viewer, a)
		if err != nil {
			return nil, err
		}
		if r.ProfitSharingEligibleOnly && (!projected.Access.LoginEnabled || !projected.Access.ProfitSharingEnabled || projected.Administrator) {
			return &account.AccountsList{}, nil
		}
		return &account.AccountsList{Items: []*account.Account{projected}, TotalSize: 1}, nil
	}
	if s.directory == nil {
		return nil, status.Error(codes.Internal, "account directory is not configured")
	}
	page := r.Page
	if page == 0 {
		page = 1
	}
	pageSize := r.PageSize
	if pageSize == 0 {
		pageSize = 50
	}
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return nil, status.Error(codes.InvalidArgument, "page must be positive and pageSize must be between 1 and 100")
	}
	statusFilter, err := accountStatusFilter(r.Status)
	if err != nil {
		return nil, err
	}
	names, total, err := s.directory.ListAccountDirectory(ctx, r.Query, statusFilter, page, pageSize, r.ProfitSharingEligibleOnly)
	if err != nil {
		return nil, err
	}
	response := &account.AccountsList{Items: make([]*account.Account, 0, len(names)), TotalSize: total}
	for _, name := range names {
		a, err := s.credentials.Get(name)
		if err != nil {
			return nil, fmt.Errorf("get directory account %q: %w", name, err)
		}
		projected, err := s.accountForViewer(ctx, name, a)
		if err != nil {
			return nil, fmt.Errorf("project directory account %q: %w", name, err)
		}
		response.Items = append(response.Items, projected)
	}
	return response, nil
}

func accountStatusFilter(value account.AccountStatus) (string, error) {
	switch value {
	case account.AccountStatus_ACCOUNT_STATUS_UNSPECIFIED:
		return "all", nil
	case account.AccountStatus_ACCOUNT_STATUS_PENDING:
		return "pending", nil
	case account.AccountStatus_ACCOUNT_STATUS_ACTIVE:
		return "active", nil
	case account.AccountStatus_ACCOUNT_STATUS_BLOCKED:
		return "blocked", nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "unsupported account status %d", value)
	}
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
		LoginEnabled:         r.Access.LoginEnabled,
		APIKeyEnabled:        r.Access.ApiKeyEnabled,
		ProfitSharingEnabled: r.Access.ProfitSharingEnabled,
		Modules:              modules,
		Revision:             r.Access.Revision,
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
	name := session.GetUserIdentifier(ctx)
	if err := s.requireAPIKeyAccess(name); err != nil {
		return nil, err
	}
	a, err := s.credentials.Get(name)
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
	if err := s.requireAPIKeyAccess(name); err != nil {
		return nil, err
	}
	if r.ExpiresIn < 0 {
		return nil, status.Error(codes.InvalidArgument, "API Key expiration must not be negative")
	}
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

	tokenString, err := s.credentials.IssueAPIKey(ctx, name, id, r.ExpiresIn)
	if err != nil {
		if errors.Is(err, accountcredentials.ErrAPIKeyAccessDisabled) {
			return nil, status.Error(codes.PermissionDenied, "API Key access is disabled")
		}
		return nil, fmt.Errorf("failed to update account with new token: %w", err)
	}
	return &account.CreateTokenResponse{Token: tokenString}, nil
}

// DeleteToken deletes a token
func (s *Server) DeleteToken(ctx context.Context, r *account.DeleteTokenRequest) (*account.EmptyResponse, error) {
	name := session.GetUserIdentifier(ctx)
	if err := s.requireAPIKeyAccess(name); err != nil {
		return nil, err
	}
	err := s.credentials.DeleteAPIKey(ctx, name, r.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete token from account %s: %w", name, err)
	}
	return &account.EmptyResponse{}, nil
}

func (s *Server) requireAPIKeyAccess(name string) error {
	access, err := s.accessController.Get(name)
	if err != nil {
		return err
	}
	if !access.LoginEnabled || !access.APIKeyEnabled {
		return status.Error(codes.PermissionDenied, "API Key access is disabled")
	}
	return nil
}
