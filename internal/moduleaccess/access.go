// Package moduleaccess implements API-only admission. Background services do not
// depend on this policy and admitted requests are never revoked retroactively.
package moduleaccess

import (
	"context"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Key string

const (
	TraderSync        Key = "trader_sync"
	Solana            Key = "solana"
	MarketRadar       Key = "market_radar"
	ManagedOO         Key = "managed_oo"
	ProfitSharing     Key = "profit_sharing"
	Worm              Key = "worm"
	Domain                = "athena.module_access"
	ClosedReason          = "MODULE_ACCESS_CLOSED"
	UnavailableReason     = "MODULE_ACCESS_UNAVAILABLE"
)

func Keys() []Key { return []Key{TraderSync, Solana, MarketRadar, ManagedOO, ProfitSharing, Worm} }
func Valid(key Key) bool {
	for _, k := range Keys() {
		if k == key {
			return true
		}
	}
	return false
}

type Setting struct {
	Key                Key
	Open               bool
	UpdatedByAccountID string
	UpdatedByUsername  string
	UpdatedAt          time.Time
}
type Store interface {
	GetModuleAccessSetting(context.Context, Key) (Setting, error)
	ListModuleAccessSettings(context.Context) ([]Setting, error)
	UpdateModuleAccessSetting(context.Context, Key, bool, string) (Setting, error)
}

func Error(reason string, key Key) error {
	info := &errdetails.ErrorInfo{Domain: Domain, Reason: reason}
	if key != "" {
		info.Metadata = map[string]string{"module_key": string(key)}
	}
	s, _ := status.New(codes.Unavailable, reason).WithDetails(info)
	return s.Err()
}
func Check(ctx context.Context, store Store, key Key) error {
	if !Valid(key) {
		return status.Error(codes.Internal, "module access classification is missing")
	}
	if store == nil {
		return Error(UnavailableReason, key)
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	setting, err := store.GetModuleAccessSetting(ctx, key)
	if err != nil {
		return Error(UnavailableReason, key)
	}
	if !setting.Open {
		return Error(ClosedReason, key)
	}
	return nil
}
