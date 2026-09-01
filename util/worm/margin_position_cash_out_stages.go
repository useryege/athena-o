package worm

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"

	solana "github.com/gagliardetto/solana-go"
)

const marginPositionCashOutMaximumDecimalBytes = 128

var marginPositionCashOutPositiveDecimalPattern = regexp.MustCompile(
	`^(?:0|[1-9][0-9]*)(?:\.[0-9]+)?$`,
)

// MarginPositionCashOutClient is the narrow HMAC protocol surface required by
// the resumable margin-position cash-out stages. Callers own durable mutation
// checkpoints and must never dispatch the same prepared Close after an
// ambiguous outcome.
type MarginPositionCashOutClient interface {
	GetMarginPosition(ctx context.Context, pubkey string) (*MarginPosition, error)
	CloseMarginPosition(
		ctx context.Context,
		pubkey string,
		options CloseMarginPositionOptions,
	) (*CloseMarginPositionResult, error)
}

// MarginPositionCashOutTarget is the safe, immutable identity and current
// lifecycle projection of one HMAC margin position. PositionPubkey is the HMAC
// position pubkey, not the numeric Worm Web position_id.
type MarginPositionCashOutTarget struct {
	PositionPubkey        string
	PositionRequestPubkey *string
	MarketConditionID     string
	IsYes                 bool
	CreatedAt             int64
	TotalShares           string
	IsClosed              bool
	IsLiquidated          bool
}

// MarginPositionCashOutCommand is an immutable, prepared whole-position
// market Close. The provider request and target are private; durable callers
// persist RequestSHA256 before dispatching it.
type MarginPositionCashOutCommand struct {
	target        MarginPositionCashOutTarget
	request       marginPositionCashOutProviderRequest
	requestSHA256 [sha256.Size]byte
}

// RequestSHA256 returns the stable digest of the exact HMAC Close target and
// query shape. A nil price is intentional and means a whole-position market
// cash out.
func (c *MarginPositionCashOutCommand) RequestSHA256() [sha256.Size]byte {
	if c == nil {
		return [sha256.Size]byte{}
	}
	return c.requestSHA256
}

// Target returns a defensive copy of the frozen position identity.
func (c *MarginPositionCashOutCommand) Target() MarginPositionCashOutTarget {
	if c == nil {
		return MarginPositionCashOutTarget{}
	}
	return cloneMarginPositionCashOutTarget(c.target)
}

// MarginPositionCashOutCloseType is Worm's bounded Close response mode.
type MarginPositionCashOutCloseType string

const (
	MarginPositionCashOutCloseTypeZero  MarginPositionCashOutCloseType = "zero"
	MarginPositionCashOutCloseTypeOrder MarginPositionCashOutCloseType = "order"
)

// MarginPositionCashOutDispatchObservation contains only the safe response
// echo returned by one HMAC Close dispatch.
type MarginPositionCashOutDispatchObservation struct {
	PositionPubkey string
	CloseType      MarginPositionCashOutCloseType
	IsClosed       bool
}

type marginPositionCashOutProviderRequest struct {
	PositionPubkey string   `json:"position_pubkey"`
	Price          *float64 `json:"price"`
}

// InspectMarginPositionCashOutTarget strictly converts one provider position
// into the identity used by preflight, preparation, and later observation.
// Lifecycle flags are retained as evidence rather than interpreted here.
func InspectMarginPositionCashOutTarget(
	position *MarginPosition,
) (MarginPositionCashOutTarget, error) {
	if position == nil {
		return MarginPositionCashOutTarget{}, errors.New("Worm margin position is required")
	}
	target := MarginPositionCashOutTarget{
		PositionPubkey:    position.Pubkey,
		MarketConditionID: position.Market.ConditionID,
		IsYes:             position.IsYes,
		TotalShares:       position.TotalShares,
		IsClosed:          position.IsClosed,
		IsLiquidated:      position.IsLiquidated,
	}
	if position.Created == nil {
		return target, errors.New("Worm margin position has no creation time")
	}
	target.CreatedAt = *position.Created
	if position.PositionRequestPubkey != nil {
		positionRequestPubkey := *position.PositionRequestPubkey
		target.PositionRequestPubkey = &positionRequestPubkey
	}
	if err := validateMarginPositionCashOutTarget(target); err != nil {
		return target, err
	}
	return target, nil
}

// PrepareMarginPositionCashOut validates and freezes one open HMAC position as
// a whole-position market Close. It performs no provider request.
func PrepareMarginPositionCashOut(
	target MarginPositionCashOutTarget,
) (*MarginPositionCashOutCommand, error) {
	if err := validateMarginPositionCashOutTarget(target); err != nil {
		return nil, err
	}
	if target.IsLiquidated {
		return nil, errors.New("Worm margin position is liquidated and cannot be cashed out")
	}
	if target.IsClosed {
		return nil, errors.New("Worm margin position is already closed")
	}

	request := marginPositionCashOutProviderRequest{
		PositionPubkey: target.PositionPubkey,
		Price:          nil,
	}
	digest, err := digestMarginPositionCashOutRequest(request)
	if err != nil {
		return nil, fmt.Errorf("digest Worm margin-position Cash Out: %w", err)
	}
	return &MarginPositionCashOutCommand{
		target:        cloneMarginPositionCashOutTarget(target),
		request:       request,
		requestSHA256: digest,
	}, nil
}

// DispatchMarginPositionCashOut sends exactly one prepared HMAC Close with no
// price. It never retries or performs a follow-up read. A caller must persist a
// dispatched checkpoint before invoking this function.
func DispatchMarginPositionCashOut(
	ctx context.Context,
	client MarginPositionCashOutClient,
	command *MarginPositionCashOutCommand,
) (*MarginPositionCashOutDispatchObservation, error) {
	if client == nil {
		return nil, errors.New("Worm margin-position Cash Out client is required")
	}
	if command == nil {
		return nil, errors.New("Worm margin-position Cash Out command is required")
	}
	if err := validateMarginPositionCashOutTarget(command.target); err != nil {
		return nil, fmt.Errorf("validate Worm margin-position Cash Out command: %w", err)
	}
	if command.target.IsLiquidated || command.target.IsClosed {
		return nil, errors.New("Worm margin-position Cash Out command is not closable")
	}
	if command.request.PositionPubkey != command.target.PositionPubkey || command.request.Price != nil {
		return nil, errors.New("Worm margin-position Cash Out command changed after preparation")
	}
	expectedDigest, err := digestMarginPositionCashOutRequest(command.request)
	if err != nil || subtle.ConstantTimeCompare(expectedDigest[:], command.requestSHA256[:]) != 1 {
		return nil, errors.New("Worm margin-position Cash Out command changed after preparation")
	}

	providerResult, dispatchErr := client.CloseMarginPosition(
		ctx,
		command.request.PositionPubkey,
		CloseMarginPositionOptions{Price: nil},
	)
	observation, observationErr := normalizeMarginPositionCashOutResult(
		providerResult,
		command.target.PositionPubkey,
	)
	return observation, errors.Join(dispatchErr, observationErr)
}

// ObserveMarginPositionCashOut performs exactly one safe HMAC GET for the
// frozen pubkey and rejects any identity drift. Polling, retry timing,
// connection transitions, and durable recovery remain caller policy.
func ObserveMarginPositionCashOut(
	ctx context.Context,
	client MarginPositionCashOutClient,
	target MarginPositionCashOutTarget,
) (MarginPositionCashOutTarget, error) {
	if client == nil {
		return MarginPositionCashOutTarget{}, errors.New("Worm margin-position Cash Out client is required")
	}
	if err := validateMarginPositionCashOutTarget(target); err != nil {
		return MarginPositionCashOutTarget{}, err
	}
	position, err := client.GetMarginPosition(ctx, target.PositionPubkey)
	if err != nil {
		return MarginPositionCashOutTarget{}, err
	}
	observed, err := InspectMarginPositionCashOutTarget(position)
	if err != nil {
		return observed, err
	}
	if err := validateMarginPositionCashOutIdentity(target, observed); err != nil {
		return observed, err
	}
	return observed, nil
}

func validateMarginPositionCashOutTarget(target MarginPositionCashOutTarget) error {
	if err := validateCanonicalMarginPositionCashOutPublicKey(target.PositionPubkey, "position pubkey"); err != nil {
		return err
	}
	if err := validateCanonicalMarginPositionCashOutPublicKey(target.MarketConditionID, "market condition id"); err != nil {
		return err
	}
	if target.PositionRequestPubkey != nil {
		if err := validateCanonicalMarginPositionCashOutPublicKey(
			*target.PositionRequestPubkey,
			"position request pubkey",
		); err != nil {
			return err
		}
	}
	if target.CreatedAt <= 0 {
		return errors.New("Worm margin position creation time must be positive")
	}
	if err := validateMarginPositionCashOutPositiveDecimal(target.TotalShares, "total shares"); err != nil {
		return err
	}
	return nil
}

func validateCanonicalMarginPositionCashOutPublicKey(value string, field string) error {
	if value == "" {
		return fmt.Errorf("Worm margin position %s is required", field)
	}
	publicKey, err := solana.PublicKeyFromBase58(value)
	if err != nil || publicKey.IsZero() || publicKey.String() != value {
		return fmt.Errorf("Worm margin position %s must be a canonical Solana public key", field)
	}
	return nil
}

func validateMarginPositionCashOutIdentity(
	expected MarginPositionCashOutTarget,
	observed MarginPositionCashOutTarget,
) error {
	if observed.PositionPubkey != expected.PositionPubkey {
		return errors.New("Worm margin position pubkey changed during Cash Out observation")
	}
	if observed.MarketConditionID != expected.MarketConditionID {
		return errors.New("Worm margin position market changed during Cash Out observation")
	}
	if observed.IsYes != expected.IsYes {
		return errors.New("Worm margin position side changed during Cash Out observation")
	}
	if observed.CreatedAt != expected.CreatedAt {
		return errors.New("Worm margin position creation time changed during Cash Out observation")
	}
	if !equalMarginPositionCashOutOptionalString(
		observed.PositionRequestPubkey,
		expected.PositionRequestPubkey,
	) {
		return errors.New("Worm margin position request pubkey changed during Cash Out observation")
	}
	return nil
}

func validateMarginPositionCashOutPositiveDecimal(value string, field string) error {
	if len(value) == 0 || len(value) > marginPositionCashOutMaximumDecimalBytes ||
		!marginPositionCashOutPositiveDecimalPattern.MatchString(value) {
		return fmt.Errorf("Worm margin position %s must be a canonical positive decimal", field)
	}
	decimal, ok := new(big.Rat).SetString(value)
	if !ok || decimal.Sign() <= 0 {
		return fmt.Errorf("Worm margin position %s must be a canonical positive decimal", field)
	}
	return nil
}

func normalizeMarginPositionCashOutResult(
	result *CloseMarginPositionResult,
	expectedPositionPubkey string,
) (*MarginPositionCashOutDispatchObservation, error) {
	if result == nil {
		return nil, errors.New("Worm margin-position Cash Out response is missing")
	}
	observation := &MarginPositionCashOutDispatchObservation{
		IsClosed: result.IsClosed,
	}
	if err := validateCanonicalMarginPositionCashOutPublicKey(result.PositionPubkey, "response position pubkey"); err != nil {
		return observation, err
	}
	observation.PositionPubkey = result.PositionPubkey
	if observation.PositionPubkey != expectedPositionPubkey {
		return observation, errors.New("Worm margin-position Cash Out response position pubkey mismatch")
	}
	switch result.CloseType {
	case string(MarginPositionCashOutCloseTypeZero):
		observation.CloseType = MarginPositionCashOutCloseTypeZero
	case string(MarginPositionCashOutCloseTypeOrder):
		observation.CloseType = MarginPositionCashOutCloseTypeOrder
	default:
		return observation, errors.New("Worm margin-position Cash Out response has an invalid close type")
	}
	return observation, nil
}

func cloneMarginPositionCashOutTarget(
	target MarginPositionCashOutTarget,
) MarginPositionCashOutTarget {
	cloned := target
	if target.PositionRequestPubkey != nil {
		positionRequestPubkey := *target.PositionRequestPubkey
		cloned.PositionRequestPubkey = &positionRequestPubkey
	}
	return cloned
}

func equalMarginPositionCashOutOptionalString(left *string, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func digestMarginPositionCashOutRequest(
	request marginPositionCashOutProviderRequest,
) ([sha256.Size]byte, error) {
	encoded, err := json.Marshal(request)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(encoded), nil
}
