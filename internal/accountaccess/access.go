package accountaccess

import (
	"errors"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Module identifies one independently authorized product module.
type Module string

const (
	ModuleMarketRadar         Module = "market_radar"
	ModuleSportsLive          Module = "sports_live"
	ModuleSportsHistory       Module = "sports_history"
	ModuleManagedOO           Module = "managed_oo"
	ModuleWormMarkets         Module = "worm_markets"
	ModuleFIFAMarketDashboard Module = "fifa_market_dashboard"
	ModuleWorldCupCorners     Module = "world_cup_corners"
	ModuleToken               Module = "token"
	ModuleWallet              Module = "wallet"
	ModuleNotifications       Module = "notifications"
)

var allModules = [...]Module{
	ModuleMarketRadar,
	ModuleSportsLive,
	ModuleSportsHistory,
	ModuleManagedOO,
	ModuleWormMarkets,
	ModuleFIFAMarketDashboard,
	ModuleWorldCupCorners,
	ModuleToken,
	ModuleWallet,
	ModuleNotifications,
}

// AccessLevel is the hierarchical data-access level for one module.
type AccessLevel string

const (
	AccessLevelNone      AccessLevel = "none"
	AccessLevelRead      AccessLevel = "read"
	AccessLevelReadWrite AccessLevel = "read_write"
)

// AllModules returns the canonical product-module order.
func AllModules() []Module {
	modules := make([]Module, len(allModules))
	copy(modules, allModules[:])
	return modules
}

// MaxAccessLevel returns the highest meaningful level for a known module.
func MaxAccessLevel(module Module) (AccessLevel, bool) {
	switch module {
	case ModuleMarketRadar, ModuleSportsLive, ModuleWormMarkets, ModuleWorldCupCorners:
		return AccessLevelRead, true
	case ModuleSportsHistory, ModuleManagedOO, ModuleFIFAMarketDashboard,
		ModuleToken, ModuleWallet, ModuleNotifications:
		return AccessLevelReadWrite, true
	default:
		return "", false
	}
}

// NoModuleAccess returns a complete module matrix with every level set to NONE.
func NoModuleAccess() map[Module]AccessLevel {
	modules := make(map[Module]AccessLevel, len(allModules))
	for _, module := range allModules {
		modules[module] = AccessLevelNone
	}
	return modules
}

// MaximumModuleAccess returns a complete module matrix at each module's maximum.
func MaximumModuleAccess() map[Module]AccessLevel {
	modules := make(map[Module]AccessLevel, len(allModules))
	for _, module := range allModules {
		maximum, _ := MaxAccessLevel(module)
		modules[module] = maximum
	}
	return modules
}

// Access is the effective access aggregate for one environment-defined account.
// Revision is zero while the account is using its environment baseline and is
// positive after the first persisted administrator update.
type Access struct {
	LoginEnabled bool
	Modules      map[Module]AccessLevel
	Revision     uint64
}

// Clone returns an aggregate that shares no mutable module map with the source.
func (a Access) Clone() Access {
	clone := Access{
		LoginEnabled: a.LoginEnabled,
		Revision:     a.Revision,
	}
	if a.Modules != nil {
		clone.Modules = make(map[Module]AccessLevel, len(a.Modules))
		for module, level := range a.Modules {
			clone.Modules[module] = level
		}
	}
	return clone
}

// Validate requires exactly one valid level for every known product module.
func (a Access) Validate() error {
	if len(a.Modules) != len(allModules) {
		return status.Errorf(codes.InvalidArgument, "account module access must contain exactly %d modules", len(allModules))
	}
	for module := range a.Modules {
		if _, known := MaxAccessLevel(module); !known {
			return status.Errorf(codes.InvalidArgument, "unsupported account access module %q", module)
		}
	}
	for _, module := range allModules {
		level, exists := a.Modules[module]
		if !exists {
			return status.Errorf(codes.InvalidArgument, "account module access is missing module %q", module)
		}
		if err := validateModuleAccessLevel(module, level); err != nil {
			return err
		}
	}
	return nil
}

func validateModuleAccessLevel(module Module, level AccessLevel) error {
	maximum, known := MaxAccessLevel(module)
	if !known {
		return status.Errorf(codes.InvalidArgument, "unsupported account access module %q", module)
	}
	switch level {
	case AccessLevelNone, AccessLevelRead:
		return nil
	case AccessLevelReadWrite:
		if maximum == AccessLevelReadWrite {
			return nil
		}
		return status.Errorf(codes.InvalidArgument, "module %q does not support read-write access", module)
	default:
		return status.Errorf(codes.InvalidArgument, "unsupported account access level %q for module %q", level, module)
	}
}

// Requirement describes either the administrator boundary or a module level.
type Requirement struct {
	Administrator bool
	Module        Module
	AccessLevel   AccessLevel
}

// RequirementAdministrator is the fixed built-in-administrator boundary.
var RequirementAdministrator = Requirement{Administrator: true}

// RequireModule builds a product-module authorization requirement.
func RequireModule(module Module, level AccessLevel) Requirement {
	return Requirement{Module: module, AccessLevel: level}
}

func (r Requirement) validate() error {
	if r.Administrator {
		if r.Module != "" || r.AccessLevel != "" {
			return status.Error(codes.Internal, "administrator requirement cannot include a product module")
		}
		return nil
	}
	if r.AccessLevel != AccessLevelRead && r.AccessLevel != AccessLevelReadWrite {
		return status.Errorf(codes.Internal, "invalid required account access level %q", r.AccessLevel)
	}
	if err := validateModuleAccessLevel(r.Module, r.AccessLevel); err != nil {
		return status.Errorf(codes.Internal, "invalid account access requirement: %v", err)
	}
	return nil
}

func accessLevelSatisfies(effective, required AccessLevel) bool {
	if required == AccessLevelRead {
		return effective == AccessLevelRead || effective == AccessLevelReadWrite
	}
	return required == AccessLevelReadWrite && effective == AccessLevelReadWrite
}

const (
	DataAccessDeniedReason          = "ACCOUNT_DATA_ACCESS_DENIED"
	AdministratorAccessDeniedReason = "ACCOUNT_ADMIN_REQUIRED"
	RevisionConflictReason          = "ACCOUNT_ACCESS_REVISION_CONFLICT"
	ErrorDomain                     = "athena.account_access"
)

var (
	ErrAdministratorAccessDenied = stableError(codes.PermissionDenied, AdministratorAccessDeniedReason, nil)
	ErrRevisionConflict          = stableError(codes.Aborted, RevisionConflictReason, nil)
)

func moduleAccessDeniedError(module Module, required, effective AccessLevel) error {
	return stableError(codes.PermissionDenied, DataAccessDeniedReason, map[string]string{
		"module":           string(module),
		"required_access":  string(required),
		"effective_access": string(effective),
	})
}

func stableError(code codes.Code, reason string, metadata map[string]string) error {
	errStatus := status.New(code, reason)
	withDetails, err := errStatus.WithDetails(&errdetails.ErrorInfo{
		Reason:   reason,
		Domain:   ErrorDomain,
		Metadata: metadata,
	})
	if err != nil {
		return errStatus.Err()
	}
	return withDetails.Err()
}

// IsRevisionConflict identifies the stable optimistic-concurrency failure.
func IsRevisionConflict(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrRevisionConflict) {
		return true
	}
	errStatus, ok := status.FromError(err)
	return ok && errStatus.Code() == codes.Aborted && errStatus.Message() == RevisionConflictReason
}
