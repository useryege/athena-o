package store

import (
	"context"
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
	ConnectionAttemptKindConnect   ConnectionAttemptKind = "CONNECT"
	ConnectionAttemptKindReconnect ConnectionAttemptKind = "RECONNECT"
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
	WalletID         int64
	Address          string
	Kind             ConnectionAttemptKind
	Nonce            string
	ChallengeMessage string
	MessageDigest    []byte
	ExpiresAt        time.Time
	Now              time.Time
}

type ActivateCredentialRequest struct {
	AttemptID           string
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

// Store is the durable persistence boundary used by Worm Trading. It stores
// Wallet correlation and credential lifecycle state, plus owner-account UUIDs
// and trusted display snapshots for saved market combinations. It never stores
// custodial private keys, login credentials, or Worm transaction signatures.
type Store interface {
	Ping(context.Context) error
	PrepareConnectionAttempt(context.Context, PrepareConnectionAttemptRequest) (*ConnectionAttempt, error)
	GetConnectionAttempt(context.Context, string) (*ConnectionAttempt, error)
	BeginConnectionAttemptCompletion(context.Context, string, time.Time) (*ConnectionAttempt, error)
	FailConnectionAttempt(context.Context, string, string, time.Time) error
	MarkConnectionAttemptOutcomeUnknown(context.Context, string, string, time.Time) error
	ExpireConnectionAttempts(context.Context, time.Time, int32) error
	ActivateCredential(context.Context, ActivateCredentialRequest) (*WalletConnectionSnapshot, error)
	ListWalletConnectionSnapshots(context.Context, []WalletReference) ([]WalletConnectionSnapshot, error)
	GetWalletConnectionSnapshot(context.Context, int64, string) (*WalletConnectionSnapshot, error)
	BeginDisconnect(context.Context, int64, string, time.Time) (*StoredCredential, error)
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
	Close() error
}
