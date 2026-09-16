package accountaccess

import "errors"

// ErrInvalidPersistedAccess identifies corrupt durable permissions, distinct
// from a failed database read or an account that does not exist.
var ErrInvalidPersistedAccess = errors.New("invalid persisted account access")
