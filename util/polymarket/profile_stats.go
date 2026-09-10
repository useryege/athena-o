package polymarket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// Decimal preserves the source JSON number, including scale. Missing and null stay nil.
type Decimal struct{ Value *string }

func (d *Decimal) UnmarshalJSON(raw []byte) error { v, e := decimalToken(raw); d.Value = v; return e }

func (d Decimal) MarshalJSON() ([]byte, error) {
	if d.Value == nil {
		return []byte("null"), nil
	}
	if _, e := decimalToken([]byte(*d.Value)); e != nil {
		return nil, e
	}
	return []byte(*d.Value), nil
}

func decimalToken(raw []byte) (*string, error) {
	text := strings.TrimSpace(string(raw))
	if text == "null" {
		return nil, nil
	}
	if len(text) == 0 || !json.Valid(raw) || (text[0] != '-' && (text[0] < '0' || text[0] > '9')) {
		return nil, fmt.Errorf("invalid numeric token")
	}
	if _, ok := new(big.Rat).SetString(text); !ok {
		return nil, fmt.Errorf("invalid decimal")
	}
	return &text, nil
}

type MarketsTraded struct {
	User   string  `json:"user"`
	Traded Decimal `json:"traded"`
}
type ProfileStats struct {
	JoinDate   *string `json:"joinDate"`
	LargestWin Decimal `json:"largestWin"`
}

func decodeObject(raw []byte, out any) error {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '{' {
		return fmt.Errorf("expected object")
	}
	return json.Unmarshal(raw, out)
}

// UserStats reads only joinDate and largestWin; malformed auxiliary fields are independent.
func (a *ProfileAdapter) UserStats(ctx context.Context, wallet string) (ProfileStats, string, error) {
	source := DefaultDataBaseURL + "/v1/user-stats?" + url.Values{"proxyAddress": {wallet}}.Encode()
	raw, e := a.get(ctx, source)
	if e != nil {
		return ProfileStats{}, source, e
	}
	var fields map[string]json.RawMessage
	if e = decodeObject(raw, &fields); e != nil {
		return ProfileStats{}, source, e
	}
	var out ProfileStats
	if r, ok := fields["joinDate"]; ok {
		_ = json.Unmarshal(r, &out.JoinDate)
	}
	if r, ok := fields["largestWin"]; ok {
		_ = json.Unmarshal(r, &out.LargestWin)
	}
	return out, source, nil
}

func (a *ProfileAdapter) Predictions(ctx context.Context, wallet string) (Decimal, string, error) {
	source := DefaultDataBaseURL + "/traded?" + url.Values{"user": {wallet}}.Encode()
	raw, e := a.get(ctx, source)
	if e != nil {
		return Decimal{}, source, e
	}
	var out MarketsTraded
	if e = decodeObject(raw, &out); e != nil {
		return Decimal{}, source, e
	}
	if !common.IsHexAddress(out.User) || !strings.EqualFold(out.User, wallet) {
		return Decimal{}, source, fmt.Errorf("traded wallet mismatch")
	}
	return out.Traded, source, nil
}

func (a *ProfileAdapter) PositionValue(ctx context.Context, wallet string) (Decimal, string, error) {
	source := DefaultDataBaseURL + "/value?" + url.Values{"user": {wallet}}.Encode()
	raw, e := a.get(ctx, source)
	if e != nil {
		return Decimal{}, source, e
	}
	var out []DataUserValue
	if e = json.Unmarshal(raw, &out); e != nil {
		return Decimal{}, source, e
	}
	if len(out) != 1 || !common.IsHexAddress(out[0].User) || !strings.EqualFold(out[0].User, wallet) {
		return Decimal{}, source, fmt.Errorf("position value wallet mismatch or ambiguous response")
	}
	return out[0].Value, source, nil
}

// optionalProfileField isolates malformed auxiliary fields from the wallet identity.
func optionalProfileField[T any](raw json.RawMessage) *T {
	if len(raw) == 0 {
		return nil
	}
	var value *T
	if json.Unmarshal(raw, &value) != nil {
		return nil
	}
	return value
}
