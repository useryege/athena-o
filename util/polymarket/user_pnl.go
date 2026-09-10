package polymarket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

const DefaultUserPNLBaseURL = "https://user-pnl-api.polymarket.com"

type UserPNLPoint struct {
	T int64
	P Decimal
}

func (a *ProfileAdapter) UserPNL(ctx context.Context, wallet, interval, fidelity string) ([]UserPNLPoint, string, error) {
	source := DefaultUserPNLBaseURL + "/user-pnl?" + url.Values{"user_address": {wallet}, "interval": {interval}, "fidelity": {fidelity}}.Encode()
	raw, e := a.get(ctx, source)
	if e != nil {
		return nil, source, e
	}
	points, e := parseUserPNL(raw)
	return points, source, e
}

func parseUserPNL(raw []byte) ([]UserPNLPoint, error) {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '[' {
		return nil, fmt.Errorf("expected P/L array")
	}
	var values []struct {
		T *int64  `json:"t"`
		P Decimal `json:"p"`
	}
	if e := json.Unmarshal(raw, &values); e != nil {
		return nil, e
	}
	if len(values) < 2 {
		return nil, fmt.Errorf("insufficient P/L points")
	}
	out := make([]UserPNLPoint, len(values))
	for i, v := range values {
		if v.T == nil || v.P.Value == nil {
			return nil, fmt.Errorf("missing P/L t/p")
		}
		if i > 0 && *v.T < out[i-1].T {
			return nil, fmt.Errorf("P/L time reversed")
		}
		out[i] = UserPNLPoint{T: *v.T, P: v.P}
	}
	return out, nil
}
