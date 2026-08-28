package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	wormtradingsqlc "github.com/useryege/athena/internal/wormtrading/store/sqlc"
)

const (
	maxWalletAddressLength = 64
	maxNonceLength         = 512
	maxChallengeLength     = 1024
	maxWarningCodeLength   = 100
)

func canonicalNow(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

func timestampParam(value time.Time) pgtype.Timestamptz {
	if value.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

func timestampValue(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time.UTC()
}

func uuidParam(value string) (pgtype.UUID, string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || parsed == uuid.Nil {
		return pgtype.UUID{}, "", fmt.Errorf("attempt ID must be a non-zero UUID")
	}
	canonical := parsed.String()
	return pgtype.UUID{Bytes: [16]byte(parsed), Valid: true}, canonical, nil
}

func uuidValue(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return uuid.UUID(value.Bytes).String()
}

func validateWalletReference(walletID int64, address string) error {
	if walletID <= 0 {
		return fmt.Errorf("wallet ID must be positive")
	}
	if address == "" || address != strings.TrimSpace(address) || len(address) > maxWalletAddressLength {
		return fmt.Errorf("wallet address is invalid")
	}
	return nil
}

func validateWarningCode(value string) error {
	if value != strings.TrimSpace(value) || len(value) > maxWarningCodeLength {
		return fmt.Errorf("warning code is invalid")
	}
	return nil
}

func credentialCleanupWarning(
	ctx context.Context,
	queries *wormtradingsqlc.Queries,
	walletID int64,
	fallback string,
) (string, error) {
	counts, err := queries.GetCredentialCleanupCounts(ctx, walletID)
	if err != nil {
		return "", fmt.Errorf("count Worm credentials awaiting cleanup: %w", err)
	}
	if counts.RevocationRequiredCount > 0 {
		return WarningCredentialRevocationRequired, nil
	}
	if counts.CleanupCount > 0 {
		return WarningCredentialRevocationPending, nil
	}
	return fallback, nil
}

func validConnectionState(value ConnectionState) bool {
	switch value {
	case ConnectionStateNotConnected,
		ConnectionStateConnecting,
		ConnectionStateConnected,
		ConnectionStateReconnectRequired,
		ConnectionStateDisconnecting,
		ConnectionStateRevocationRequired:
		return true
	default:
		return false
	}
}

func mapConnection(row wormtradingsqlc.WormWalletConnection) WalletConnectionSnapshot {
	return WalletConnectionSnapshot{
		WalletID:          row.WalletID,
		RequestedAddress:  row.Address,
		StoredAddress:     row.Address,
		State:             ConnectionState(row.State),
		WarningCode:       row.WarningCode,
		ConnectedAt:       timestampValue(row.ConnectedAt),
		ConnectionCreated: timestampValue(row.CreatedAt),
		ConnectionUpdated: timestampValue(row.UpdatedAt),
	}
}

func mapCredential(row wormtradingsqlc.WormWalletCredential) StoredCredential {
	return StoredCredential{
		ID:                  row.ID,
		WalletID:            row.WalletID,
		Version:             row.Version,
		State:               CredentialState(row.State),
		APIKeyCiphertext:    append([]byte(nil), row.ApiKeyCiphertext...),
		APISecretCiphertext: append([]byte(nil), row.ApiSecretCiphertext...),
		CreatedAt:           timestampValue(row.CreatedAt),
		UpdatedAt:           timestampValue(row.UpdatedAt),
	}
}

func mapConnectionAttempt(row wormtradingsqlc.WormWalletConnectionAttempt) ConnectionAttempt {
	return ConnectionAttempt{
		ID:                      uuidValue(row.ID),
		WalletID:                row.WalletID,
		Address:                 row.Address,
		Kind:                    ConnectionAttemptKind(row.Kind),
		PreviousConnectionState: ConnectionState(row.PreviousConnectionState),
		PreviousConnectedAt:     timestampValue(row.PreviousConnectedAt),
		Nonce:                   row.Nonce,
		ChallengeMessage:        row.ChallengeMessage,
		MessageDigest:           append([]byte(nil), row.MessageDigest...),
		State:                   ConnectionAttemptState(row.State),
		FailureCode:             row.FailureCode,
		ExpiresAt:               timestampValue(row.ExpiresAt),
		CompletedAt:             timestampValue(row.CompletedAt),
		CreatedAt:               timestampValue(row.CreatedAt),
		UpdatedAt:               timestampValue(row.UpdatedAt),
	}
}
