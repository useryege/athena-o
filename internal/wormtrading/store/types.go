package store

import (
	"context"
	"fmt"
	"time"
)

type ConnectionState string

const (
	ConnectionStateNotConnected       ConnectionState = "NOT_CONNECTED"
	ConnectionStateConnecting         ConnectionState = "CONNECTING"
	ConnectionStateConnected          ConnectionState = "CONNECTED"
	ConnectionStateReconnectRequired  ConnectionState = "RECONNECT_REQUIRED"
	ConnectionStateDisconnecting      ConnectionState = "DISCONNECTING"
	ConnectionStateRevocationRequired ConnectionState = "REVOCATION_REQUIRED"
)

type CredentialState string

const (
	CredentialStateActive             CredentialState = "ACTIVE"
	CredentialStatePendingRevocation  CredentialState = "PENDING_REVOCATION"
	CredentialStateRevoking           CredentialState = "REVOKING"
	CredentialStateRevocationRequired CredentialState = "REVOCATION_REQUIRED"
)

const (
	WarningCredentialRevocationPending  = "CREDENTIAL_REVOCATION_PENDING"
	WarningCredentialRevocationRequired = "REVOCATION_REQUIRED"
)

type ConnectionAttemptKind string

const (
	ConnectionAttemptKindConnect    ConnectionAttemptKind = "CONNECT"
	ConnectionAttemptKindReconnect  ConnectionAttemptKind = "RECONNECT"
	ConnectionAttemptKindRegenerate ConnectionAttemptKind = "REGENERATE"
)

type ConnectionAttemptState string

const (
	ConnectionAttemptStatePrepared       ConnectionAttemptState = "PREPARED"
	ConnectionAttemptStateCompleting     ConnectionAttemptState = "COMPLETING"
	ConnectionAttemptStateCompleted      ConnectionAttemptState = "COMPLETED"
	ConnectionAttemptStateFailed         ConnectionAttemptState = "FAILED"
	ConnectionAttemptStateCancelled      ConnectionAttemptState = "CANCELLED"
	ConnectionAttemptStateOutcomeUnknown ConnectionAttemptState = "OUTCOME_UNKNOWN"
)

type WalletReference struct {
	WalletID int64
	Address  string
}

const MaximumWalletSelectionItems = 20

// WalletSelectionInput is a Wallet reference whose owner and Solana type were
// verified by the API Server. Input order is persisted as selection order.
type WalletSelectionInput struct {
	WalletID int64
	Address  string
}

// WalletRetirementInput retains one pre-existing managed connection during
// first configuration. It never adds the Wallet to the selected set.
type WalletRetirementInput struct {
	WalletID     int64
	Address      string
	PriorOrdinal int32
}

type WalletSelectionItem struct {
	Ordinal  int32
	WalletID int64
	Address  string
}

type WalletRetirement struct {
	WalletID            int64
	Address             string
	PriorOrdinal        int32
	RetiredFromRevision int64
	RetiredAt           time.Time
}

type WalletSelection struct {
	OwnerAccountID string
	Configured     bool
	Revision       int64
	SelectedItems  []WalletSelectionItem
	Retirements    []WalletRetirement
	UpdatedAt      time.Time
}

func (s WalletSelection) Clone() WalletSelection {
	s.SelectedItems = append([]WalletSelectionItem(nil), s.SelectedItems...)
	s.Retirements = append([]WalletRetirement(nil), s.Retirements...)
	return s
}

type ReplaceWalletSelectionRequest struct {
	OwnerAccountID   string
	ExpectedRevision int64
	SelectedItems    []WalletSelectionInput
	RetiringWallets  []WalletRetirementInput
	Now              time.Time
}

type WalletRetirementBlocker struct {
	WalletID   int64
	ReasonCode string
}

type WalletSelectionRetirementBlockedError struct {
	WalletID   int64
	ReasonCode string
}

func (e *WalletSelectionRetirementBlockedError) Error() string {
	if e == nil {
		return ErrWalletSelectionRetirementBlocked.Error()
	}
	return fmt.Sprintf("Wallet %d has removal blocker %s", e.WalletID, e.ReasonCode)
}

func (e *WalletSelectionRetirementBlockedError) Unwrap() error {
	return ErrWalletSelectionRetirementBlocked
}

type StoredCredential struct {
	ID                  int64
	WalletID            int64
	Version             int64
	State               CredentialState
	APIKeyCiphertext    []byte
	APISecretCiphertext []byte
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (c StoredCredential) Clone() StoredCredential {
	c.APIKeyCiphertext = append([]byte(nil), c.APIKeyCiphertext...)
	c.APISecretCiphertext = append([]byte(nil), c.APISecretCiphertext...)
	return c
}

type WalletConnectionSnapshot struct {
	WalletID          int64
	RequestedAddress  string
	StoredAddress     string
	State             ConnectionState
	WarningCode       string
	ConnectedAt       time.Time
	ConnectionCreated time.Time
	ConnectionUpdated time.Time
	ActiveCredential  *StoredCredential
}

type ConnectionAttempt struct {
	ID                      string
	WalletID                int64
	Address                 string
	Kind                    ConnectionAttemptKind
	PreviousConnectionState ConnectionState
	PreviousConnectedAt     time.Time
	Nonce                   string
	ChallengeMessage        string
	MessageDigest           []byte
	State                   ConnectionAttemptState
	FailureCode             string
	ExpiresAt               time.Time
	CompletedAt             time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func (a ConnectionAttempt) Clone() ConnectionAttempt {
	a.MessageDigest = append([]byte(nil), a.MessageDigest...)
	return a
}

type PrepareConnectionAttemptRequest struct {
	AttemptID        string
	OwnerAccountID   string
	WalletID         int64
	Address          string
	Kind             ConnectionAttemptKind
	Nonce            string
	ChallengeMessage string
	MessageDigest    []byte
	ExpiresAt        time.Time
	Now              time.Time
}

type BeginConnectionAttemptCompletionRequest struct {
	OwnerAccountID string
	AttemptID      string
	WalletID       int64
	Address        string
	Now            time.Time
}

type ActivateCredentialRequest struct {
	OwnerAccountID      string
	AttemptID           string
	WalletID            int64
	Address             string
	APIKeyCiphertext    []byte
	APISecretCiphertext []byte
	Now                 time.Time
}

type MarketCombinationItemInput struct {
	EventConditionID  string
	EventTitle        string
	EventLogo         string
	MarketConditionID string
	MarketTitle       string
	MarketLogo        string
	IsYes             bool
	OutcomeLabel      string
}

type MarketCombinationItem struct {
	Ordinal int32
	MarketCombinationItemInput
}

type MarketCombination struct {
	ID             string
	OwnerAccountID string
	Name           string
	Revision       int64
	Items          []MarketCombinationItem
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ExecutionPlanState is the lifecycle of a durable read-only execution
// preview. A READY plan is never mutated again and can be consumed only before
// its expiry by the later execution capability.
type ExecutionPlanState string

const (
	ExecutionPlanStateBuilding ExecutionPlanState = "BUILDING"
	ExecutionPlanStateReady    ExecutionPlanState = "READY"
	ExecutionPlanStateFailed   ExecutionPlanState = "FAILED"
)

const (
	ExecutionPlanUsabilityCombinationDeleted     = "COMBINATION_DELETED"
	ExecutionPlanUsabilityCombinationChanged     = "COMBINATION_CHANGED"
	ExecutionPlanUsabilityExpired                = "EXPIRED"
	ExecutionPlanUsabilityNoActionableSteps      = "NO_ACTIONABLE_STEPS"
	ExecutionPlanUsabilityWalletSelectionChanged = "WALLET_SELECTION_CHANGED"
)

type ExecutionPlanStepDisposition string

const (
	ExecutionPlanStepDispositionReady   ExecutionPlanStepDisposition = "READY"
	ExecutionPlanStepDispositionSkipped ExecutionPlanStepDisposition = "SKIPPED"
)

// ExecutionPlanWalletInput is a caller-verified, non-secret wallet identity
// snapshot. Wallet order is the requested execution order.
type ExecutionPlanWalletInput struct {
	WalletID       int64
	Address        string
	Remark         string
	AvatarKind     string
	AvatarPresetID string
	AvatarURL      string
}

type ExecutionPlanWallet struct {
	Ordinal int32
	ExecutionPlanWalletInput
	ConnectionState       ConnectionState
	ConnectionWarningCode string
	ConnectedAt           time.Time
	CredentialVersion     int64
	SOLAtomicAmount       string
	SOLAmount             string
	SOLDecimals           int32
	SOLObservedSlot       uint64
	SOLAvailability       string
	SOLErrorCode          string
	USDCMint              string
	USDCAtomicAmount      string
	USDCAmount            string
	USDCDecimals          int32
	USDCObservedSlot      uint64
	USDCAvailability      string
	USDCErrorCode         string
	USDCTokenAccountCount int32
	Status                string
	ReasonCode            string
}

type ExecutionPlanEstimate struct {
	AveragePrice     string
	TotalShares      string
	TotalCost        string
	BestAsk          string
	WorstFillPrice   string
	IsFullyFilled    bool
	FeeAmount        string
	UserFundsNeeded  string
	LiquidationPrice string
}

// ExecutionPlanItem freezes a trusted combination item together with the
// current market and public estimate result observed by the build worker.
type ExecutionPlanItem struct {
	Ordinal int32
	MarketCombinationItemInput
	Backend    string
	Funds      string
	Leverage   string
	State      string
	ReasonCode string
	Estimate   ExecutionPlanEstimate
}

type ExecutionPlanStep struct {
	Ordinal             int64
	WalletOrdinal       int32
	ItemOrdinal         int32
	Disposition         ExecutionPlanStepDisposition
	ReasonCode          string
	ProjectedUSDCBefore string
	ProjectedUSDCAfter  string
}

// ExecutionPlanReasonCount is an authoritative aggregate of terminal preview
// step classifications. READY is the stable key for executable steps.
type ExecutionPlanReasonCount struct {
	ReasonCode string
	Count      int64
}

type ExecutionPlan struct {
	ID                      string
	OwnerAccountID          string
	CombinationID           string
	CombinationName         string
	CombinationRevision     int64
	WalletSelectionRevision int64
	State                   ExecutionPlanState
	BuildStage              string
	FailureCode             string
	UsabilityCode           string
	WorkerID                string
	LockedAt                time.Time
	LeaseExpiresAt          time.Time
	WalletCount             int64
	ItemCount               int64
	TotalStepCount          int64
	CompletedStepCount      int64
	ReadyStepCount          int64
	SkippedStepCount        int64
	TotalCollateral         string
	TotalOpeningFee         string
	TotalUserFundsNeeded    string
	RequestedAt             time.Time
	CompletedAt             time.Time
	ExpiresAt               time.Time
	RetentionUntil          time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
	Wallets                 []ExecutionPlanWallet
	Items                   []ExecutionPlanItem
	ReasonCounts            []ExecutionPlanReasonCount
}

type CreateExecutionPlanRequest struct {
	OwnerAccountID              string
	CombinationID               string
	ExpectedCombinationRevision int64
	WalletSelectionRevision     int64
	Wallets                     []ExecutionPlanWalletInput
	Now                         time.Time
}

type ExecutionPlanBuildProgress struct {
	PlanID             string
	WorkerID           string
	BuildStage         string
	CompletedStepCount int64
	LeaseExpiresAt     time.Time
	Now                time.Time
}

type ExecutionPlanWalletObservation struct {
	Ordinal               int32
	ConnectionState       ConnectionState
	ConnectionWarningCode string
	ConnectedAt           time.Time
	CredentialVersion     int64
	SOLAtomicAmount       string
	SOLAmount             string
	SOLDecimals           int32
	SOLObservedSlot       uint64
	SOLAvailability       string
	SOLErrorCode          string
	USDCMint              string
	USDCAtomicAmount      string
	USDCAmount            string
	USDCDecimals          int32
	USDCObservedSlot      uint64
	USDCAvailability      string
	USDCErrorCode         string
	USDCTokenAccountCount int32
	Status                string
	ReasonCode            string
}

type ExecutionPlanItemObservation struct {
	Ordinal    int32
	Backend    string
	Funds      string
	Leverage   string
	State      string
	ReasonCode string
	Estimate   ExecutionPlanEstimate
}

type MarkExecutionPlanReadyRequest struct {
	PlanID               string
	WorkerID             string
	Wallets              []ExecutionPlanWalletObservation
	Items                []ExecutionPlanItemObservation
	Steps                []ExecutionPlanStep
	TotalCollateral      string
	TotalOpeningFee      string
	TotalUserFundsNeeded string
	Now                  time.Time
}

type ExecutionRunState string

const (
	ExecutionRunStateAwaitingAuthorization  ExecutionRunState = "AWAITING_AUTHORIZATION"
	ExecutionRunStateAuthorized             ExecutionRunState = "AUTHORIZED"
	ExecutionRunStateRunning                ExecutionRunState = "RUNNING"
	ExecutionRunStatePauseRequested         ExecutionRunState = "PAUSE_REQUESTED"
	ExecutionRunStatePaused                 ExecutionRunState = "PAUSED"
	ExecutionRunStateTerminateRequested     ExecutionRunState = "TERMINATE_REQUESTED"
	ExecutionRunStateReconciliationRequired ExecutionRunState = "RECONCILIATION_REQUIRED"
	ExecutionRunStateCompleted              ExecutionRunState = "COMPLETED"
	ExecutionRunStateTerminated             ExecutionRunState = "TERMINATED"
	ExecutionRunStateFailed                 ExecutionRunState = "FAILED"
)

type ExecutionStepState string

const (
	ExecutionStepStatePending            ExecutionStepState = "PENDING"
	ExecutionStepStatePreflighting       ExecutionStepState = "PREFLIGHTING"
	ExecutionStepStateOpening            ExecutionStepState = "OPENING"
	ExecutionStepStateOpened             ExecutionStepState = "OPENED"
	ExecutionStepStateSigning            ExecutionStepState = "SIGNING"
	ExecutionStepStateFinalizing         ExecutionStepState = "FINALIZING"
	ExecutionStepStateAwaitingCompletion ExecutionStepState = "AWAITING_COMPLETION"
	ExecutionStepStateCompleted          ExecutionStepState = "COMPLETED"
	ExecutionStepStateSatisfied          ExecutionStepState = "SATISFIED"
	ExecutionStepStateSkipped            ExecutionStepState = "SKIPPED"
	ExecutionStepStateFailed             ExecutionStepState = "FAILED"
	ExecutionStepStateNotExecuted        ExecutionStepState = "NOT_EXECUTED"
	ExecutionStepStateOutcomeUnknown     ExecutionStepState = "OUTCOME_UNKNOWN"
)

type ExecutionCompletionSource string

const (
	ExecutionCompletionSourceOpenPosition ExecutionCompletionSource = "OPEN_POSITION"
)

type ExecutionMutationKind string

const (
	ExecutionMutationKindOpen     ExecutionMutationKind = "OPEN"
	ExecutionMutationKindFinalize ExecutionMutationKind = "FINALIZE"
)

type ExecutionMutationState string

const (
	ExecutionMutationStatePrepared        ExecutionMutationState = "PREPARED"
	ExecutionMutationStateDispatched      ExecutionMutationState = "DISPATCHED"
	ExecutionMutationStateSucceeded       ExecutionMutationState = "SUCCEEDED"
	ExecutionMutationStateDefiniteFailure ExecutionMutationState = "DEFINITE_FAILURE"
	ExecutionMutationStateOutcomeUnknown  ExecutionMutationState = "OUTCOME_UNKNOWN"
)

type ExecutionAuthorizationState string

const (
	ExecutionAuthorizationStateAuthorized ExecutionAuthorizationState = "AUTHORIZED"
	ExecutionAuthorizationStateConsumed   ExecutionAuthorizationState = "CONSUMED"
	ExecutionAuthorizationStateRevoked    ExecutionAuthorizationState = "REVOKED"
	ExecutionAuthorizationStateSuperseded ExecutionAuthorizationState = "SUPERSEDED"
)

type ExecutionCoordinatorState string

const (
	ExecutionCoordinatorStateActive   ExecutionCoordinatorState = "ACTIVE"
	ExecutionCoordinatorStateReleased ExecutionCoordinatorState = "RELEASED"
	ExecutionCoordinatorStateExpired  ExecutionCoordinatorState = "EXPIRED"
)

type ExecutionCommandKind string

const (
	ExecutionCommandKindAuthorize   ExecutionCommandKind = "AUTHORIZE"
	ExecutionCommandKindStart       ExecutionCommandKind = "START"
	ExecutionCommandKindPause       ExecutionCommandKind = "PAUSE"
	ExecutionCommandKindResume      ExecutionCommandKind = "RESUME"
	ExecutionCommandKindTerminate   ExecutionCommandKind = "TERMINATE"
	ExecutionCommandKindHeartbeat   ExecutionCommandKind = "HEARTBEAT"
	ExecutionCommandKindExecuteNext ExecutionCommandKind = "EXECUTE_NEXT"
	ExecutionCommandKindReconcile   ExecutionCommandKind = "RECONCILE"
)

type ExecutionCommandState string

const (
	ExecutionCommandStateAccepted   ExecutionCommandState = "ACCEPTED"
	ExecutionCommandStateInProgress ExecutionCommandState = "IN_PROGRESS"
	ExecutionCommandStateApplied    ExecutionCommandState = "APPLIED"
	ExecutionCommandStateFailed     ExecutionCommandState = "FAILED"
	ExecutionCommandStateAbandoned  ExecutionCommandState = "ABANDONED"
)

// ExecutionRunAction values are the sole public allowed_actions vocabulary.
// Continue deliberately maps to the internal RESUME command/RPC terminology.
type ExecutionRunAction string

const (
	ExecutionRunActionAuthorize   ExecutionRunAction = "AUTHORIZE"
	ExecutionRunActionStart       ExecutionRunAction = "START"
	ExecutionRunActionPause       ExecutionRunAction = "PAUSE"
	ExecutionRunActionContinue    ExecutionRunAction = "CONTINUE"
	ExecutionRunActionTerminate   ExecutionRunAction = "TERMINATE"
	ExecutionRunActionHeartbeat   ExecutionRunAction = "HEARTBEAT"
	ExecutionRunActionExecuteNext ExecutionRunAction = "EXECUTE_NEXT"
	ExecutionRunActionReconcile   ExecutionRunAction = "RECONCILE"
)

type ExecutionAuthorization struct {
	ID               string
	State            ExecutionAuthorizationState
	Scope            string
	ProofKind        string
	SessionJTIDigest []byte
	AccessRevision   int64
	PlanVersion      int64
	PlanDigestSHA256 []byte
	AuthorizedAt     time.Time
	EndedAt          time.Time
	EndReasonCode    string
}

type ExecutionCoordinator struct {
	ID             string
	Generation     int64
	State          ExecutionCoordinatorState
	AccessRevision int64
	AcquiredAt     time.Time
	HeartbeatAt    time.Time
	LeaseExpiresAt time.Time
	ReleasedAt     time.Time
}

type ExecutionMutationAttempt struct {
	ID                string
	CommandID         string
	Kind              ExecutionMutationKind
	State             ExecutionMutationState
	RequestSHA256     []byte
	PositionRequestID int64
	HTTPStatus        int32
	ProviderCode      int32
	ProviderSlug      string
	ErrorCode         string
	PreparedAt        time.Time
	DispatchedAt      time.Time
	CompletedAt       time.Time
}

type ExecutionStepIsolation struct {
	ID                string
	WalletID          int64
	MarketConditionID string
	ReasonCode        string
	CreatedAt         time.Time
	ResolvedAt        time.Time
	ResolutionCode    string
}

type ExecutionRunStep struct {
	ID                              string
	Ordinal                         int64
	PlanStepOrdinal                 int64
	WalletOrdinal                   int32
	ItemOrdinal                     int32
	SourceDisposition               ExecutionPlanStepDisposition
	SourceReasonCode                string
	ProjectedUSDCBefore             string
	ProjectedUSDCAfter              string
	State                           ExecutionStepState
	ReasonCode                      string
	PositionRequestID               int64
	FinalizeMode                    string
	TransactionMessageSHA256        []byte
	TransactionVersion              string
	RequiredSignatureCount          int32
	WalletSignerIndex               int32
	ProviderState                   string
	ProviderOrderState              string
	FundingTxID                     string
	RefundTxID                      string
	CompletionSource                ExecutionCompletionSource
	CompletionPositionPubkey        string
	CompletionPositionRequestPubkey string
	CompletionPositionCreatedAt     time.Time
	StartedAt                       time.Time
	OpenedAt                        time.Time
	FinalizedAt                     time.Time
	LastObservedAt                  time.Time
	CompletedAt                     time.Time
	CreatedAt                       time.Time
	UpdatedAt                       time.Time
	NextPollAt                      time.Time
	PollCount                       int32
	ClaimCommandID                  string
	ClaimID                         string
	ClaimOwner                      string
	ClaimExpiresAt                  time.Time
	ReconcileRequestedAt            time.Time
	Attempts                        []ExecutionMutationAttempt
	Isolation                       *ExecutionStepIsolation
}

type ExecutionRun struct {
	ID                   string
	OwnerAccountID       string
	PlanID               string
	PlanVersion          int64
	PlanDigestSHA256     []byte
	CombinationID        string
	CombinationName      string
	CombinationRevision  int64
	State                ExecutionRunState
	Revision             int64
	CurrentStepOrdinal   int64
	NextStepOrdinal      int64
	WalletCount          int64
	ItemCount            int64
	TotalStepCount       int64
	ActionableStepCount  int64
	TerminalStepCount    int64
	CompletedStepCount   int64
	SatisfiedStepCount   int64
	SkippedStepCount     int64
	FailedStepCount      int64
	NotExecutedStepCount int64
	PauseCode            string
	FailureCode          string
	BlockCode            string
	RequestedAt          time.Time
	AuthorizedAt         time.Time
	StartedAt            time.Time
	PausedAt             time.Time
	CompletedAt          time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
	Wallets              []ExecutionPlanWallet
	Items                []ExecutionPlanItem
	Authorization        *ExecutionAuthorization
	Coordinator          *ExecutionCoordinator
	CurrentStep          *ExecutionRunStep
	AllowedActions       []ExecutionRunAction
}

type CreateExecutionRunRequest struct {
	OwnerAccountID   string
	PlanID           string
	CommandID        string
	ExpectedRevision int64
	Now              time.Time
}

type ExecutionRunCommandRequest struct {
	OwnerAccountID   string
	RunID            string
	CommandID        string
	ExpectedRevision int64
	Now              time.Time
}

type AuthorizeExecutionRunRequest struct {
	ExecutionRunCommandRequest
	ProofKind        string
	SessionJTIDigest []byte
	AccessRevision   int64
}

type StartExecutionRunRequest struct {
	ExecutionRunCommandRequest
	SessionJTIDigest []byte
	AccessRevision   int64
}

type ResumeExecutionRunRequest = StartExecutionRunRequest

type HeartbeatExecutionCoordinatorRequest struct {
	ExecutionRunCommandRequest
	CoordinatorToken []byte
	SessionJTIDigest []byte
	AccessRevision   int64
}

type BeginExecutionStepRequest struct {
	ExecutionRunCommandRequest
	ExpectedStepOrdinal int64
	CoordinatorToken    []byte
	SessionJTIDigest    []byte
	AccessRevision      int64
}

type ReconcileExecutionStepRequest struct {
	ExecutionRunCommandRequest
	ExpectedStepOrdinal int64
	StepID              string
}

type ExecutionCoordinatorLease struct {
	Run   ExecutionRun
	Token []byte
}

type ExecutionStepScope string

const (
	ExecutionStepScopeCurrent         ExecutionStepScope = "CURRENT"
	ExecutionStepScopeRemainingWallet ExecutionStepScope = "REMAINING_WALLET"
	ExecutionStepScopeRemainingMarket ExecutionStepScope = "REMAINING_MARKET"
)

type ExecutionStepClaimRequest struct {
	OwnerAccountID      string
	RunID               string
	CommandID           string
	ExpectedRunRevision int64
	ExpectedStepOrdinal int64
	CoordinatorToken    []byte
	SessionJTIDigest    []byte
	AccessRevision      int64
	LeaseExpiresAt      time.Time
	Now                 time.Time
}

type RecoverExecutionStepRequest struct {
	RunID          string
	StepOrdinal    int64
	ClaimID        string
	WorkerID       string
	LeaseExpiresAt time.Time
	Now            time.Time
}

type CompleteExecutionPreflightRequest struct {
	RunID       string
	StepOrdinal int64
	CommandID   string
	ClaimID     string
	NextState   ExecutionStepState
	ReasonCode  string
	SkipScope   ExecutionStepScope
	Now         time.Time
}

type PrepareExecutionMutationRequest struct {
	AttemptID         string
	RunID             string
	StepOrdinal       int64
	CommandID         string
	ClaimID           string
	Kind              ExecutionMutationKind
	RequestSHA256     []byte
	PositionRequestID int64
	Now               time.Time
}

type ResolveExecutionMutationRequest struct {
	AttemptID         string
	State             ExecutionMutationState
	PositionRequestID int64
	HTTPStatus        int32
	ProviderCode      int32
	ProviderSlug      string
	ErrorCode         string
	Now               time.Time
}

type DispatchExecutionMutationRequest struct {
	AttemptID   string
	RunID       string
	StepOrdinal int64
	ClaimID     string
	Now         time.Time
}

type RenewExecutionStepClaimRequest struct {
	RunID          string
	StepOrdinal    int64
	ClaimID        string
	WorkerID       string
	LeaseExpiresAt time.Time
	Now            time.Time
}

type RecordExecutionStepOpenedRequest struct {
	AttemptID                string
	RunID                    string
	StepOrdinal              int64
	ClaimID                  string
	PositionRequestID        int64
	TransactionMessageSHA256 []byte
	ProviderState            string
	ProviderOrderState       string
	HTTPStatus               int32
	ProviderCode             int32
	ProviderSlug             string
	Now                      time.Time
}

type AdvanceExecutionStepRequest struct {
	RunID       string
	StepOrdinal int64
	ClaimID     string
	Now         time.Time
}

type RecordExecutionStepSignedRequest struct {
	AdvanceExecutionStepRequest
	FinalizeMode           string
	TransactionVersion     string
	RequiredSignatureCount int32
	WalletSignerIndex      int32
}

type RecordExecutionProviderObservationRequest struct {
	RunID                           string
	StepOrdinal                     int64
	CommandID                       string
	ClaimID                         string
	ExpectedState                   ExecutionStepState
	NextState                       ExecutionStepState
	ReasonCode                      string
	PositionRequestID               int64
	ProviderState                   string
	ProviderOrderState              string
	FundingTxID                     string
	RefundTxID                      string
	CompletionSource                ExecutionCompletionSource
	CompletionPositionPubkey        string
	CompletionPositionRequestPubkey string
	CompletionPositionCreatedAt     time.Time
	NextPollAt                      time.Time
	SkipScope                       ExecutionStepScope
	IsolationID                     string
	AttemptID                       string
	ResolveIsolationID              string
	IsolationResolutionCode         string
	Now                             time.Time
}

type PauseExecutionRunForFailureRequest struct {
	RunID     string
	PauseCode string
	Now       time.Time
}

type CreateExecutionStepIsolationRequest struct {
	IsolationID string
	RunID       string
	StepOrdinal int64
	AttemptID   string
	ReasonCode  string
	Now         time.Time
}

type ResolveExecutionStepIsolationRequest struct {
	IsolationID    string
	RunID          string
	StepOrdinal    int64
	ResolutionCode string
	Now            time.Time
}

type RecoverableExecutionStep struct {
	OwnerAccountID        string
	RunID                 string
	RunState              ExecutionRunState
	RunRevision           int64
	CommandID             string
	CoordinatorID         string
	CoordinatorGeneration int64
	Step                  ExecutionRunStep
}

// PositionCashOutState is the durable lifecycle of one owner-authorized,
// whole-position HMAC market Close. RECONCILIATION_REQUIRED remains active and
// continues to isolate its Wallet because a dispatched mutation may have taken
// effect remotely.
type PositionCashOutState string

const (
	PositionCashOutStateAwaitingAuthorization  PositionCashOutState = "AWAITING_AUTHORIZATION"
	PositionCashOutStateQueued                 PositionCashOutState = "QUEUED"
	PositionCashOutStatePreflighting           PositionCashOutState = "PREFLIGHTING"
	PositionCashOutStateClosing                PositionCashOutState = "CLOSING"
	PositionCashOutStateAwaitingCompletion     PositionCashOutState = "AWAITING_COMPLETION"
	PositionCashOutStateCompleted              PositionCashOutState = "COMPLETED"
	PositionCashOutStateFailed                 PositionCashOutState = "FAILED"
	PositionCashOutStateReconciliationRequired PositionCashOutState = "RECONCILIATION_REQUIRED"
	PositionCashOutStateExpired                PositionCashOutState = "EXPIRED"
)

type PositionCashOutAuthorizationState string

const (
	PositionCashOutAuthorizationStateAuthorized PositionCashOutAuthorizationState = "AUTHORIZED"
	PositionCashOutAuthorizationStateConsumed   PositionCashOutAuthorizationState = "CONSUMED"
	PositionCashOutAuthorizationStateRevoked    PositionCashOutAuthorizationState = "REVOKED"
)

type PositionCashOutAttemptState string

const (
	PositionCashOutAttemptStatePrepared       PositionCashOutAttemptState = "PREPARED"
	PositionCashOutAttemptStateDispatched     PositionCashOutAttemptState = "DISPATCHED"
	PositionCashOutAttemptStateAcknowledged   PositionCashOutAttemptState = "ACKNOWLEDGED"
	PositionCashOutAttemptStateRejected       PositionCashOutAttemptState = "REJECTED"
	PositionCashOutAttemptStateOutcomeUnknown PositionCashOutAttemptState = "OUTCOME_UNKNOWN"
)

type PositionCashOutCommandKind string

const (
	PositionCashOutCommandKindCreate    PositionCashOutCommandKind = "CREATE"
	PositionCashOutCommandKindAuthorize PositionCashOutCommandKind = "AUTHORIZE"
	PositionCashOutCommandKindReconcile PositionCashOutCommandKind = "RECONCILE"
)

type PositionCashOutAuthorization struct {
	ID                 string
	State              PositionCashOutAuthorizationState
	Scope              string
	ProofKind          string
	SessionJTIDigest   []byte
	AccessRevision     int64
	IntentDigestSHA256 []byte
	AuthorizedAt       time.Time
	EndedAt            time.Time
	EndReasonCode      string
}

type PositionCashOutAttempt struct {
	ID            string
	State         PositionCashOutAttemptState
	RequestSHA256 []byte
	HTTPStatus    int32
	ProviderCode  int32
	ProviderSlug  string
	ProviderState string
	ErrorCode     string
	PreparedAt    time.Time
	DispatchedAt  time.Time
	CompletedAt   time.Time
}

type PositionCashOut struct {
	ID                     string
	OwnerAccountID         string
	WalletID               int64
	WalletAddress          string
	CredentialVersion      int64
	PositionPubkey         string
	MarketConditionID      string
	IsYes                  bool
	PositionCreatedAt      time.Time
	PositionRequestPubkey  string
	Shares                 string
	IntentDigestSHA256     []byte
	State                  PositionCashOutState
	Revision               int64
	ReasonCode             string
	ProviderState          string
	ProviderIsClosed       bool
	ProviderIsLiquidated   bool
	AuthorizationExpiresAt time.Time
	ExecutionExpiresAt     time.Time
	AuthorizedAt           time.Time
	NextPollAt             time.Time
	PollCount              int32
	ReconcileRequestedAt   time.Time
	ClaimID                string
	ClaimOwner             string
	ClaimExpiresAt         time.Time
	CompletedAt            time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
	BatchID                string
	BatchItemID            string
	Authorization          *PositionCashOutAuthorization
	Attempt                *PositionCashOutAttempt
}

func (c PositionCashOut) Clone() PositionCashOut {
	c.IntentDigestSHA256 = append([]byte(nil), c.IntentDigestSHA256...)
	if c.Authorization != nil {
		authorization := *c.Authorization
		authorization.SessionJTIDigest = append([]byte(nil), authorization.SessionJTIDigest...)
		authorization.IntentDigestSHA256 = append([]byte(nil), authorization.IntentDigestSHA256...)
		c.Authorization = &authorization
	}
	if c.Attempt != nil {
		attempt := *c.Attempt
		attempt.RequestSHA256 = append([]byte(nil), attempt.RequestSHA256...)
		c.Attempt = &attempt
	}
	return c
}

type CreatePositionCashOutRequest struct {
	OwnerAccountID         string
	CommandID              string
	WalletID               int64
	WalletAddress          string
	CredentialVersion      int64
	PositionPubkey         string
	MarketConditionID      string
	IsYes                  bool
	PositionCreatedAt      time.Time
	PositionRequestPubkey  string
	Shares                 string
	ProviderState          string
	ProviderIsClosed       bool
	ProviderIsLiquidated   bool
	AuthorizationExpiresAt time.Time
	Now                    time.Time
}

// GetPositionCashOutCreationRequest is the stable browser-supplied identity
// of a CREATE command. It intentionally excludes provider observations and
// generated expiry timestamps so an acknowledged-but-lost response can be
// replayed without another Worm request.
type GetPositionCashOutCreationRequest struct {
	OwnerAccountID string
	CommandID      string
	WalletID       int64
	WalletAddress  string
	PositionPubkey string
}

type PositionCashOutCommandRequest struct {
	OwnerAccountID   string
	CashOutID        string
	CommandID        string
	ExpectedRevision int64
	Now              time.Time
}

type AuthorizePositionCashOutRequest struct {
	PositionCashOutCommandRequest
	ProofKind          string
	SessionJTIDigest   []byte
	AccessRevision     int64
	ExecutionExpiresAt time.Time
}

type PositionCashOutClaimRequest struct {
	CashOutID      string
	ClaimID        string
	WorkerID       string
	LeaseExpiresAt time.Time
	Now            time.Time
}

type BeginPositionCashOutClosingRequest struct {
	CashOutID     string
	ClaimID       string
	ProviderState string
	Now           time.Time
}

type PreparePositionCashOutAttemptRequest struct {
	AttemptID     string
	CashOutID     string
	ClaimID       string
	RequestSHA256 []byte
	Now           time.Time
}

type DispatchPositionCashOutAttemptRequest struct {
	AttemptID string
	CashOutID string
	ClaimID   string
	Now       time.Time
}

type ResolvePositionCashOutAttemptRequest struct {
	AttemptID     string
	CashOutID     string
	State         PositionCashOutAttemptState
	HTTPStatus    int32
	ProviderCode  int32
	ProviderSlug  string
	ProviderState string
	ErrorCode     string
	Now           time.Time
}

// CompletePositionCashOutDispatchRequest atomically records a validated
// closed response echo, resolves the one dispatched attempt, and completes
// the operation. This prevents a crash between those durable facts from
// discarding authoritative completion evidence.
type CompletePositionCashOutDispatchRequest struct {
	AttemptID     string
	CashOutID     string
	ClaimID       string
	HTTPStatus    int32
	ProviderCode  int32
	ProviderSlug  string
	ProviderState string
	ErrorCode     string
	Now           time.Time
}

type RecordPositionCashOutObservationRequest struct {
	CashOutID            string
	ClaimID              string
	ExpectedState        PositionCashOutState
	NextState            PositionCashOutState
	ReasonCode           string
	ProviderState        string
	ProviderIsClosed     bool
	ProviderIsLiquidated bool
	NextPollAt           time.Time
	Now                  time.Time
}

// PositionCashOutBatchState is the durable lifecycle of one owner-authorized,
// Wallet-major batch. PAUSED and RECONCILIATION_REQUIRED retain every selected
// Wallet lock. Terminal states release them.
type PositionCashOutBatchState string

const (
	PositionCashOutBatchStateBuilding               PositionCashOutBatchState = "BUILDING"
	PositionCashOutBatchStateAwaitingAuthorization  PositionCashOutBatchState = "AWAITING_AUTHORIZATION"
	PositionCashOutBatchStateQueued                 PositionCashOutBatchState = "QUEUED"
	PositionCashOutBatchStateRunning                PositionCashOutBatchState = "RUNNING"
	PositionCashOutBatchStatePauseRequested         PositionCashOutBatchState = "PAUSE_REQUESTED"
	PositionCashOutBatchStatePaused                 PositionCashOutBatchState = "PAUSED"
	PositionCashOutBatchStateTerminateRequested     PositionCashOutBatchState = "TERMINATE_REQUESTED"
	PositionCashOutBatchStateCompleted              PositionCashOutBatchState = "COMPLETED"
	PositionCashOutBatchStateFailed                 PositionCashOutBatchState = "FAILED"
	PositionCashOutBatchStateReconciliationRequired PositionCashOutBatchState = "RECONCILIATION_REQUIRED"
	PositionCashOutBatchStateTerminated             PositionCashOutBatchState = "TERMINATED"
	PositionCashOutBatchStateCancelled              PositionCashOutBatchState = "CANCELLED"
	PositionCashOutBatchStateExpired                PositionCashOutBatchState = "EXPIRED"
)

type PositionCashOutBatchItemState string

const (
	PositionCashOutBatchItemStatePending                PositionCashOutBatchItemState = "PENDING"
	PositionCashOutBatchItemStatePreflighting           PositionCashOutBatchItemState = "PREFLIGHTING"
	PositionCashOutBatchItemStateClosing                PositionCashOutBatchItemState = "CLOSING"
	PositionCashOutBatchItemStateAwaitingPosition       PositionCashOutBatchItemState = "AWAITING_POSITION"
	PositionCashOutBatchItemStateAwaitingBalance        PositionCashOutBatchItemState = "AWAITING_BALANCE"
	PositionCashOutBatchItemStateCompleted              PositionCashOutBatchItemState = "COMPLETED"
	PositionCashOutBatchItemStateFailed                 PositionCashOutBatchItemState = "FAILED"
	PositionCashOutBatchItemStateReconciliationRequired PositionCashOutBatchItemState = "RECONCILIATION_REQUIRED"
	PositionCashOutBatchItemStateNotExecuted            PositionCashOutBatchItemState = "NOT_EXECUTED"
)

type PositionCashOutBatchAuthorizationState string

const (
	PositionCashOutBatchAuthorizationStateAuthorized PositionCashOutBatchAuthorizationState = "AUTHORIZED"
	PositionCashOutBatchAuthorizationStateConsumed   PositionCashOutBatchAuthorizationState = "CONSUMED"
	PositionCashOutBatchAuthorizationStateRevoked    PositionCashOutBatchAuthorizationState = "REVOKED"
	PositionCashOutBatchAuthorizationStateSuperseded PositionCashOutBatchAuthorizationState = "SUPERSEDED"
)

type PositionCashOutBatchCommandKind string

const (
	PositionCashOutBatchCommandKindCreate      PositionCashOutBatchCommandKind = "CREATE"
	PositionCashOutBatchCommandKindAuthorize   PositionCashOutBatchCommandKind = "AUTHORIZE"
	PositionCashOutBatchCommandKindCancel      PositionCashOutBatchCommandKind = "CANCEL"
	PositionCashOutBatchCommandKindPause       PositionCashOutBatchCommandKind = "PAUSE"
	PositionCashOutBatchCommandKindContinue    PositionCashOutBatchCommandKind = "CONTINUE"
	PositionCashOutBatchCommandKindTerminate   PositionCashOutBatchCommandKind = "TERMINATE"
	PositionCashOutBatchCommandKindCheckStatus PositionCashOutBatchCommandKind = "CHECK_STATUS"
)

type PositionCashOutBatchBalanceEvidence struct {
	Mint         string
	Decimals     int32
	AtomicAmount string
	ObservedSlot uint64
}

type PositionCashOutBatchWalletInput struct {
	WalletID       int64
	Address        string
	Remark         string
	AvatarKind     string
	AvatarPresetID string
	AvatarURL      string
}

type PositionCashOutBatchWallet struct {
	Ordinal           int32
	WalletID          int64
	Address           string
	CredentialVersion int64
	Remark            string
	AvatarKind        string
	AvatarPresetID    string
	AvatarURL         string
	PositionCount     int64
	CompletedCount    int64
}

type PositionCashOutBatchItemInput struct {
	WalletOrdinal         int32
	PositionOrdinal       int32
	WalletID              int64
	WalletAddress         string
	WalletRemark          string
	CredentialVersion     int64
	PositionPubkey        string
	PositionRequestPubkey string
	MarketConditionID     string
	MarketTitle           string
	IsYes                 bool
	Shares                string
	PositionCreatedAt     time.Time
	ProviderState         string
}

type PositionCashOutBatchItem struct {
	ID                    string
	BatchID               string
	Ordinal               int64
	WalletOrdinal         int32
	PositionOrdinal       int32
	WalletID              int64
	WalletAddress         string
	WalletRemark          string
	CredentialVersion     int64
	PositionPubkey        string
	PositionRequestPubkey string
	MarketConditionID     string
	MarketTitle           string
	IsYes                 bool
	Shares                string
	PositionCreatedAt     time.Time
	ProviderState         string
	State                 PositionCashOutBatchItemState
	ReasonCode            string
	ChildCashOutID        string
	Baseline              *PositionCashOutBatchBalanceEvidence
	Observed              *PositionCashOutBatchBalanceEvidence
	DeltaUSDCAtomicAmount string
	BalanceStartedAt      time.Time
	BalanceDeadlineAt     time.Time
	BalanceConfirmedAt    time.Time
	CompletedAt           time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type PositionCashOutBatchAuthorization struct {
	ID                 string
	State              PositionCashOutBatchAuthorizationState
	Scope              string
	ProofKind          string
	SessionJTIDigest   []byte
	AccessRevision     int64
	IntentDigestSHA256 []byte
	AuthorizedAt       time.Time
	EndedAt            time.Time
	EndReasonCode      string
}

type PositionCashOutBatch struct {
	ID                     string
	OwnerAccountID         string
	SelectionDigestSHA256  []byte
	IntentDigestSHA256     []byte
	State                  PositionCashOutBatchState
	Revision               int64
	ReasonCode             string
	BuildStage             string
	WalletCount            int64
	PositionCount          int64
	CompletedCount         int64
	NotExecutedCount       int64
	NextItemOrdinal        int64
	CurrentItemOrdinal     int64
	AuthorizationExpiresAt time.Time
	AuthorizedAt           time.Time
	ExecutionStartedAt     time.Time
	NextPollAt             time.Time
	CheckRequestedAt       time.Time
	ClaimID                string
	ClaimOwner             string
	ClaimExpiresAt         time.Time
	CompletedAt            time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
	Wallets                []PositionCashOutBatchWallet
	CurrentItem            *PositionCashOutBatchItem
	Authorization          *PositionCashOutBatchAuthorization
}

func (b PositionCashOutBatch) Clone() PositionCashOutBatch {
	b.SelectionDigestSHA256 = append([]byte(nil), b.SelectionDigestSHA256...)
	b.IntentDigestSHA256 = append([]byte(nil), b.IntentDigestSHA256...)
	b.Wallets = append([]PositionCashOutBatchWallet(nil), b.Wallets...)
	if b.CurrentItem != nil {
		item := clonePositionCashOutBatchItem(*b.CurrentItem)
		b.CurrentItem = &item
	}
	if b.Authorization != nil {
		authorization := *b.Authorization
		authorization.SessionJTIDigest = append([]byte(nil), authorization.SessionJTIDigest...)
		authorization.IntentDigestSHA256 = append([]byte(nil), authorization.IntentDigestSHA256...)
		b.Authorization = &authorization
	}
	return b
}

func clonePositionCashOutBatchItem(item PositionCashOutBatchItem) PositionCashOutBatchItem {
	if item.Baseline != nil {
		baseline := *item.Baseline
		item.Baseline = &baseline
	}
	if item.Observed != nil {
		observed := *item.Observed
		item.Observed = &observed
	}
	return item
}

type CreatePositionCashOutBatchRequest struct {
	OwnerAccountID         string
	CommandID              string
	Wallets                []PositionCashOutBatchWalletInput
	AuthorizationExpiresAt time.Time
	Now                    time.Time
}

type CompletePositionCashOutBatchBuildRequest struct {
	BatchID                string
	ClaimID                string
	Items                  []PositionCashOutBatchItemInput
	AuthorizationExpiresAt time.Time
	Now                    time.Time
}

type FailPositionCashOutBatchBuildRequest struct {
	BatchID    string
	ClaimID    string
	BuildStage string
	ReasonCode string
	Now        time.Time
}

type AuthorizePositionCashOutBatchRequest struct {
	OwnerAccountID   string
	BatchID          string
	CommandID        string
	ExpectedRevision int64
	ProofKind        string
	SessionJTIDigest []byte
	AccessRevision   int64
	Now              time.Time
}

type PositionCashOutBatchCommandRequest struct {
	OwnerAccountID   string
	BatchID          string
	CommandID        string
	ExpectedRevision int64
	Kind             PositionCashOutBatchCommandKind
	Now              time.Time
}

type PositionCashOutBatchClaimRequest struct {
	BatchID        string
	ClaimID        string
	WorkerID       string
	LeaseExpiresAt time.Time
	Now            time.Time
}

type ActivatePositionCashOutBatchItemRequest struct {
	BatchID       string
	ItemID        string
	ClaimID       string
	CashOutID     string
	ProviderState string
	NextPollAt    time.Time
	Now           time.Time
}

type DeferPositionCashOutBatchPreflightRequest struct {
	BatchID string
	ItemID  string
	ClaimID string
	Now     time.Time
}

type DeferPositionCashOutBatchCheckRequest struct {
	BatchID string
	ClaimID string
	Now     time.Time
}

// DispatchPositionCashOutBatchAttemptRequest binds the confirmed USDC
// baseline to the same transaction that commits the child Close attempt as
// DISPATCHED. A caller must never send Close unless this operation succeeds.
type DispatchPositionCashOutBatchAttemptRequest struct {
	BatchID    string
	ItemID     string
	AttemptID  string
	CashOutID  string
	ClaimID    string
	Baseline   PositionCashOutBatchBalanceEvidence
	NextPollAt time.Time
	Now        time.Time
}

type RecordPositionCashOutBatchItemStateRequest struct {
	BatchID           string
	ItemID            string
	ClaimID           string
	ExpectedState     PositionCashOutBatchItemState
	NextState         PositionCashOutBatchItemState
	ReasonCode        string
	ProviderState     string
	BalanceDeadlineAt time.Time
	NextPollAt        time.Time
	Now               time.Time
}

type RecordPositionCashOutBatchBalanceRequest struct {
	BatchID  string
	ItemID   string
	ClaimID  string
	Observed PositionCashOutBatchBalanceEvidence
	TimedOut bool
	Now      time.Time
}

type DeferPositionCashOutBatchBalanceRequest struct {
	BatchID  string
	ItemID   string
	ClaimID  string
	TimedOut bool
	Now      time.Time
}

type RecordPositionCashOutBatchBlockingRequest struct {
	BatchID    string
	ItemID     string
	ClaimID    string
	ItemState  PositionCashOutBatchItemState
	BatchState PositionCashOutBatchState
	ReasonCode string
	Now        time.Time
}

type CompletePositionCashOutBatchBoundaryRequest struct {
	BatchID string
	ClaimID string
	Now     time.Time
}

func (l ExecutionCoordinatorLease) Clone() ExecutionCoordinatorLease {
	l.Run = l.Run.Clone()
	l.Token = append([]byte(nil), l.Token...)
	return l
}

func (r ExecutionRun) Clone() ExecutionRun {
	r.PlanDigestSHA256 = append([]byte(nil), r.PlanDigestSHA256...)
	r.Wallets = append([]ExecutionPlanWallet(nil), r.Wallets...)
	r.Items = append([]ExecutionPlanItem(nil), r.Items...)
	r.AllowedActions = append([]ExecutionRunAction(nil), r.AllowedActions...)
	if r.Authorization != nil {
		authorization := *r.Authorization
		authorization.SessionJTIDigest = append([]byte(nil), authorization.SessionJTIDigest...)
		authorization.PlanDigestSHA256 = append([]byte(nil), authorization.PlanDigestSHA256...)
		r.Authorization = &authorization
	}
	if r.Coordinator != nil {
		coordinator := *r.Coordinator
		r.Coordinator = &coordinator
	}
	if r.CurrentStep != nil {
		step := r.CurrentStep.Clone()
		r.CurrentStep = &step
	}
	return r
}

func (s ExecutionRunStep) Clone() ExecutionRunStep {
	s.TransactionMessageSHA256 = append([]byte(nil), s.TransactionMessageSHA256...)
	s.Attempts = append([]ExecutionMutationAttempt(nil), s.Attempts...)
	for index := range s.Attempts {
		s.Attempts[index].RequestSHA256 = append([]byte(nil), s.Attempts[index].RequestSHA256...)
	}
	if s.Isolation != nil {
		isolation := *s.Isolation
		s.Isolation = &isolation
	}
	return s
}

// Store is the durable persistence boundary used by Worm Trading. It stores
// Wallet correlation and credential lifecycle state, plus owner-account UUIDs
// and trusted display snapshots for saved market combinations. It never stores
// custodial private keys, login credentials, or Worm transaction signatures.
type Store interface {
	Ping(context.Context) error
	GetWalletSelection(context.Context, string) (*WalletSelection, error)
	ReplaceWalletSelection(context.Context, ReplaceWalletSelectionRequest) (*WalletSelection, error)
	ListWalletRetirementBlockers(context.Context, string, []int64) ([]WalletRetirementBlocker, error)
	CompleteWalletRetirement(context.Context, string, int64, string) error
	PrepareConnectionAttempt(context.Context, PrepareConnectionAttemptRequest) (*ConnectionAttempt, error)
	GetConnectionAttempt(context.Context, string) (*ConnectionAttempt, error)
	BeginConnectionAttemptCompletion(context.Context, BeginConnectionAttemptCompletionRequest) (*ConnectionAttempt, error)
	FailConnectionAttempt(context.Context, string, string, time.Time) error
	MarkConnectionAttemptOutcomeUnknown(context.Context, string, string, time.Time) error
	RecoverConnectionAttempts(context.Context, time.Time, int32) (int64, error)
	ExpireConnectionAttempts(context.Context, time.Time, int32) error
	ActivateCredential(context.Context, ActivateCredentialRequest) (*WalletConnectionSnapshot, error)
	ListWalletConnectionSnapshots(context.Context, []WalletReference) ([]WalletConnectionSnapshot, error)
	GetWalletConnectionSnapshot(context.Context, int64, string) (*WalletConnectionSnapshot, error)
	BeginDisconnect(context.Context, string, int64, string, time.Time) (*StoredCredential, error)
	BeginCredentialRevocation(context.Context, int64, int64, time.Time) (*StoredCredential, error)
	ListCredentialsNeedingRevocation(context.Context, time.Time, int32) ([]StoredCredential, error)
	MarkCredentialRevoked(context.Context, int64, int64, time.Time) error
	MarkCredentialRevocationFailed(context.Context, int64, int64, ConnectionState, CredentialState, string, time.Time) error
	MarkReconnectRequired(context.Context, int64, string, int64, string, time.Time) error
	CreateMarketCombination(context.Context, string, string, []MarketCombinationItemInput) (*MarketCombination, error)
	GetMarketCombination(context.Context, string, string) (*MarketCombination, error)
	ListMarketCombinations(context.Context, string, int32, int32) ([]MarketCombination, int64, error)
	UpdateMarketCombination(context.Context, string, string, string, int64, []MarketCombinationItemInput) (*MarketCombination, error)
	DeleteMarketCombination(context.Context, string, string, int64) error
	CreateExecutionPlan(context.Context, CreateExecutionPlanRequest) (*ExecutionPlan, error)
	GetExecutionPlan(context.Context, string, string, time.Time) (*ExecutionPlan, error)
	ListExecutionPlanSteps(context.Context, string, string, int32, int32) ([]ExecutionPlanStep, int64, error)
	ClaimExecutionPlan(context.Context, string, time.Duration, time.Time) (*ExecutionPlan, error)
	UpdateExecutionPlanBuildProgress(context.Context, ExecutionPlanBuildProgress) (*ExecutionPlan, error)
	MarkExecutionPlanReady(context.Context, MarkExecutionPlanReadyRequest) (*ExecutionPlan, error)
	MarkExecutionPlanFailed(context.Context, string, string, string, time.Time) (*ExecutionPlan, error)
	DeleteExpiredExecutionPlans(context.Context, time.Time, int32) (int64, error)
	CreateExecutionRun(context.Context, CreateExecutionRunRequest) (*ExecutionRun, error)
	ListExecutionRuns(context.Context, string, int32, int32) ([]ExecutionRun, int64, error)
	GetExecutionRun(context.Context, string, string) (*ExecutionRun, error)
	ListExecutionRunSteps(context.Context, string, string, int32, int32) ([]ExecutionRunStep, int64, error)
	AuthorizeExecutionRun(context.Context, AuthorizeExecutionRunRequest) (*ExecutionRun, error)
	StartExecutionRun(context.Context, StartExecutionRunRequest) (*ExecutionCoordinatorLease, error)
	PauseExecutionRun(context.Context, ExecutionRunCommandRequest) (*ExecutionRun, error)
	ResumeExecutionRun(context.Context, ResumeExecutionRunRequest) (*ExecutionCoordinatorLease, error)
	TerminateExecutionRun(context.Context, ExecutionRunCommandRequest) (*ExecutionRun, error)
	HeartbeatExecutionCoordinator(context.Context, HeartbeatExecutionCoordinatorRequest) (*ExecutionRun, error)
	BeginExecutionStep(context.Context, BeginExecutionStepRequest) (*ExecutionRunStep, error)
	BeginExecutionStepReconciliation(context.Context, ReconcileExecutionStepRequest) (*ExecutionRunStep, error)
	ClaimExecutionStep(context.Context, ExecutionStepClaimRequest) (*ExecutionRunStep, error)
	RecoverExecutionStep(context.Context, RecoverExecutionStepRequest) (*ExecutionRunStep, error)
	CompleteExecutionPreflight(context.Context, CompleteExecutionPreflightRequest) (*ExecutionRunStep, error)
	PrepareExecutionMutation(context.Context, PrepareExecutionMutationRequest) (*ExecutionMutationAttempt, error)
	RenewExecutionStepClaim(context.Context, RenewExecutionStepClaimRequest) (*ExecutionRunStep, error)
	DispatchExecutionMutation(context.Context, DispatchExecutionMutationRequest) (*ExecutionMutationAttempt, error)
	ResolveExecutionMutation(context.Context, ResolveExecutionMutationRequest) (*ExecutionMutationAttempt, error)
	RecordExecutionStepOpened(context.Context, RecordExecutionStepOpenedRequest) (*ExecutionRunStep, error)
	MarkExecutionStepSigning(context.Context, AdvanceExecutionStepRequest) (*ExecutionRunStep, error)
	RecordExecutionStepSigned(context.Context, RecordExecutionStepSignedRequest) (*ExecutionRunStep, error)
	RecordExecutionProviderObservation(context.Context, RecordExecutionProviderObservationRequest) (*ExecutionRunStep, error)
	PauseExecutionRunForFailure(context.Context, PauseExecutionRunForFailureRequest) (*ExecutionRun, error)
	CreateExecutionStepIsolation(context.Context, CreateExecutionStepIsolationRequest) (*ExecutionStepIsolation, error)
	ResolveExecutionStepIsolation(context.Context, ResolveExecutionStepIsolationRequest) (*ExecutionStepIsolation, error)
	ListRecoverableExecutionSteps(context.Context, time.Time, int32) ([]RecoverableExecutionStep, error)
	ListActiveExecutionRunWalletIDs(context.Context, string, []int64) ([]int64, error)
	GetPositionCashOutCreation(context.Context, GetPositionCashOutCreationRequest) (*PositionCashOut, bool, error)
	CreatePositionCashOut(context.Context, CreatePositionCashOutRequest) (*PositionCashOut, error)
	GetPositionCashOut(context.Context, string, string) (*PositionCashOut, error)
	ListActivePositionCashOuts(context.Context, string) ([]PositionCashOut, error)
	AuthorizePositionCashOut(context.Context, AuthorizePositionCashOutRequest) (*PositionCashOut, error)
	ClaimPositionCashOut(context.Context, PositionCashOutClaimRequest) (*PositionCashOut, error)
	RenewPositionCashOutClaim(context.Context, PositionCashOutClaimRequest) (*PositionCashOut, error)
	BeginPositionCashOutClosing(context.Context, BeginPositionCashOutClosingRequest) (*PositionCashOut, error)
	PreparePositionCashOutAttempt(context.Context, PreparePositionCashOutAttemptRequest) (*PositionCashOutAttempt, error)
	DispatchPositionCashOutAttempt(context.Context, DispatchPositionCashOutAttemptRequest) (*PositionCashOutAttempt, error)
	ResolvePositionCashOutAttempt(context.Context, ResolvePositionCashOutAttemptRequest) (*PositionCashOutAttempt, error)
	CompletePositionCashOutDispatch(context.Context, CompletePositionCashOutDispatchRequest) (*PositionCashOut, error)
	RecordPositionCashOutObservation(context.Context, RecordPositionCashOutObservationRequest) (*PositionCashOut, error)
	RequestPositionCashOutReconciliation(context.Context, PositionCashOutCommandRequest) (*PositionCashOut, error)
	ListRecoverablePositionCashOuts(context.Context, time.Time, int32) ([]PositionCashOut, error)
	ExpirePositionCashOuts(context.Context, time.Time, int32) (int64, error)
	CreatePositionCashOutBatch(context.Context, CreatePositionCashOutBatchRequest) (*PositionCashOutBatch, error)
	GetPositionCashOutBatch(context.Context, string, string) (*PositionCashOutBatch, error)
	GetActivePositionCashOutBatch(context.Context, string) (*PositionCashOutBatch, error)
	ListPositionCashOutBatchItems(context.Context, string, string, int32, int32) ([]PositionCashOutBatchItem, int64, error)
	CompletePositionCashOutBatchBuild(context.Context, CompletePositionCashOutBatchBuildRequest) (*PositionCashOutBatch, error)
	FailPositionCashOutBatchBuild(context.Context, FailPositionCashOutBatchBuildRequest) (*PositionCashOutBatch, error)
	AuthorizePositionCashOutBatch(context.Context, AuthorizePositionCashOutBatchRequest) (*PositionCashOutBatch, error)
	ControlPositionCashOutBatch(context.Context, PositionCashOutBatchCommandRequest) (*PositionCashOutBatch, error)
	ClaimPositionCashOutBatch(context.Context, PositionCashOutBatchClaimRequest) (*PositionCashOutBatch, error)
	RenewPositionCashOutBatchClaim(context.Context, PositionCashOutBatchClaimRequest) (*PositionCashOutBatch, error)
	DeferPositionCashOutBatchPreflight(context.Context, DeferPositionCashOutBatchPreflightRequest) (*PositionCashOutBatch, error)
	DeferPositionCashOutBatchCheck(context.Context, DeferPositionCashOutBatchCheckRequest) (*PositionCashOutBatch, error)
	ActivatePositionCashOutBatchItem(context.Context, ActivatePositionCashOutBatchItemRequest) (*PositionCashOutBatch, error)
	DispatchPositionCashOutBatchAttempt(context.Context, DispatchPositionCashOutBatchAttemptRequest) (*PositionCashOutAttempt, error)
	RecordPositionCashOutBatchItemState(context.Context, RecordPositionCashOutBatchItemStateRequest) (*PositionCashOutBatch, error)
	RecordPositionCashOutBatchBalance(context.Context, RecordPositionCashOutBatchBalanceRequest) (*PositionCashOutBatch, error)
	DeferPositionCashOutBatchBalance(context.Context, DeferPositionCashOutBatchBalanceRequest) (*PositionCashOutBatch, error)
	RecordPositionCashOutBatchBlocking(context.Context, RecordPositionCashOutBatchBlockingRequest) (*PositionCashOutBatch, error)
	CompletePositionCashOutBatchBoundary(context.Context, CompletePositionCashOutBatchBoundaryRequest) (*PositionCashOutBatch, error)
	ListRecoverablePositionCashOutBatches(context.Context, time.Time, int32) ([]PositionCashOutBatch, error)
	ExpirePositionCashOutBatches(context.Context, time.Time, int32) (int64, error)
	Close() error
}
