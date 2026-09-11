package tradersync

import (
	"context"
	"github.com/useryege/athena/internal/tradersync/store"
	tm "github.com/useryege/athena/internal/tradersync/types"
)

// Intake accepts only the actual funding-wallet OrderFilled source. Metadata and
// receipt siblings never enter this path; ownership is frozen by its first INSERT.
type Intake struct {
	store *store.SQLStore
	token uint64
}

func NewIntake(s *store.SQLStore, token uint64) *Intake { return &Intake{store: s, token: token} }
func (i *Intake) Persist(ctx context.Context, epoch uint64, received tm.ReceivedLog) error {
	raw := received.Raw
	for id, version := range sourceVersions {
		if raw.Address == version.deployment.address && len(raw.Topics) > 0 && raw.Topics[0] == orderFilledEvents[id].ID {
			return i.store.PersistReceived(ctx, i.token, epoch, received)
		}
	}
	return nil
}
