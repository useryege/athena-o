package tradersync

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/tradersync/store"
	tsmodel "github.com/useryege/athena/internal/tradersync/types"
	"github.com/useryege/athena/util/polymarket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrantCheck func(context.Context, pgx.Tx, string) error
type ResolveContextTx func(context.Context, pgx.Tx, string, common.Address) (tsmodel.ResolutionContext, error)
type TargetResolver struct {
	pool           *pgxpool.Pool
	profiles       *polymarket.ProfileAdapter
	store          *store.SQLStore
	grant          GrantCheck
	resolveContext ResolveContextTx
}

func NewTargetResolver(pool *pgxpool.Pool, profiles *polymarket.ProfileAdapter, grant GrantCheck, resolveContext ResolveContextTx) (*TargetResolver, error) {
	if pool == nil || profiles == nil || grant == nil || resolveContext == nil {
		return nil, fmt.Errorf("pool, profile adapter, grant check and owner context resolver are required")
	}
	return &TargetResolver{pool: pool, profiles: profiles, store: store.NewSQLStore(pool), grant: grant, resolveContext: resolveContext}, nil
}

func identityFromProfile(p polymarket.ProfileIdentity) tsmodel.Identity {
	v := tsmodel.Identity{Wallet: p.Wallet, ResolutionInput: p.ResolutionInput, ProfileURL: p.CanonicalURL}
	if p.Public.Name != nil {
		v.DisplayName = *p.Public.Name
	} else if p.Public.Pseudonym != nil {
		v.DisplayName = *p.Public.Pseudonym
	}
	if p.Public.ProfileImage != nil {
		v.AvatarURL = *p.Public.ProfileImage
	}
	v.Digest = sha256.Sum256([]byte(polymarket.ProfileAdapterVersion + "\n" + v.ResolutionInput + "\n" + strings.ToLower(v.Wallet.Hex()) + "\n" + v.ProfileURL))
	return v
}

func (r *TargetResolver) Revalidate(ctx context.Context, identity tsmodel.Identity) error {
	if strings.TrimSpace(identity.ResolutionInput) == "" {
		return ErrIdentityChanged
	}
	p, e := r.profiles.ResolveIdentity(ctx, identity.ResolutionInput)
	if e != nil {
		return status.Errorf(codes.FailedPrecondition, "target resolution source cannot be revalidated; resolve again: %v", e)
	}
	if current := identityFromProfile(p); current.Digest != identity.Digest {
		return ErrIdentityChanged
	}
	return nil
}

func (r *TargetResolver) Resolve(ctx context.Context, ownerID, input string) (tsmodel.ResolvedTarget, error) {
	var result tsmodel.ResolvedTarget
	owner, e := uuid.Parse(ownerID)
	if e != nil || owner == uuid.Nil || len(ownerID) != 36 || !strings.EqualFold(owner.String(), ownerID) {
		return result, status.Error(codes.InvalidArgument, "owner must be a nonzero canonical UUID")
	}
	ownerID = owner.String()
	card, e := r.loadProfile(ctx, input)
	if e != nil {
		return result, e
	}
	token := make([]byte, 32)
	if _, e = rand.Read(token); e != nil {
		return result, e
	}
	digest := sha256.Sum256(token)
	e = txgate.WithAccountTx(ctx, r.pool, ownerID, func(tx pgx.Tx) error {
		if e := r.grant(ctx, tx, ownerID); e != nil {
			return e
		}
		ownerContext, e := r.resolveContext(ctx, tx, ownerID, card.Identity.Wallet)
		if e != nil {
			return e
		}
		expires, e := r.store.ConfirmationExpiryTx(ctx, tx)
		if e != nil {
			return e
		}
		if e = r.store.SaveConfirmationTx(ctx, tx, ownerID, card.Identity, digest[:], expires); e != nil {
			return e
		}
		if e = r.store.SaveConfirmationCardTx(ctx, tx, ownerID, digest[:], card); e != nil {
			return e
		}
		result = tsmodel.ResolvedTarget{Card: card, Context: ownerContext, Token: base64.RawURLEncoding.EncodeToString(token), ExpiresAt: expires}
		return nil
	})
	if e != nil {
		return tsmodel.ResolvedTarget{}, e
	}
	return result, nil
}
