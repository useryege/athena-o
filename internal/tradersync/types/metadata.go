package types

import "fmt"

type MarketRef struct {
	Evidence
	ID, Title, URL, ConditionID, PositionID, Outcome string
}
type ComboLeg struct {
	PositionID string
	Market     MarketRef
}
type TradeMetadata struct {
	Market       MarketRef
	LegsEvidence Evidence
	Legs         []ComboLeg
	Relationship string
}

// DirectoryError describes only metadata collection. It is not evidence of a
// failed trade observation epoch. A later locked state read resolves unknown
// directory commits; callers must not use an unconfirmed next-page timestamp.
type DirectoryError struct {
	Phase                      string
	HTTPAttempted, CommitKnown bool
	Err, CleanupErr            error
}

func (e *DirectoryError) Error() string {
	return fmt.Sprintf("directory %s (http_attempted=%t commit_known=%t): %v; cleanup: %v", e.Phase, e.HTTPAttempted, e.CommitKnown, e.Err, e.CleanupErr)
}
func (e *DirectoryError) Unwrap() []error { return []error{e.Err, e.CleanupErr} }
