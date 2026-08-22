package accountaccess

import (
	"errors"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DataAccess is the complete business-data access level assigned to an account.
type DataAccess string

const (
	DataAccessNone      DataAccess = "none"
	DataAccessRead      DataAccess = "read"
	DataAccessReadWrite DataAccess = "read_write"
)

// Access is the effective access aggregate for one environment-defined account.
// Revision is zero while the account is using its environment baseline and is
// positive after the first persisted administrator update.
type Access struct {
	LoginEnabled bool
	DataAccess   DataAccess
	Revision     uint64
}

func (a Access) validate() error {
	switch a.DataAccess {
	case DataAccessNone, DataAccessRead, DataAccessReadWrite:
		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "unsupported account data access %q", a.DataAccess)
	}
}

// Requirement is an authorization category assigned to a protected RPC.
type Requirement int

const (
	RequirementAdministrator Requirement = iota
	RequirementDataRead
	RequirementDataWrite
)

const (
	DataAccessDeniedReason          = "ACCOUNT_DATA_ACCESS_DENIED"
	AdministratorAccessDeniedReason = "ACCOUNT_ADMIN_REQUIRED"
	RevisionConflictReason          = "ACCOUNT_ACCESS_REVISION_CONFLICT"
	ErrorDomain                     = "athena.account_access"
)

var (
	ErrDataAccessDenied          = stableError(codes.PermissionDenied, DataAccessDeniedReason)
	ErrAdministratorAccessDenied = stableError(codes.PermissionDenied, AdministratorAccessDeniedReason)
	ErrRevisionConflict          = stableError(codes.Aborted, RevisionConflictReason)
)

func stableError(code codes.Code, reason string) error {
	errStatus := status.New(code, reason)
	withDetails, err := errStatus.WithDetails(&errdetails.ErrorInfo{
		Reason: reason,
		Domain: ErrorDomain,
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
