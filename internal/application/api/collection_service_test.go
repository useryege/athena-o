package api

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	appstore "github.com/useryege/athena/internal/application/store"
)

type chainIngestServiceStoreFake struct {
	appstore.Store

	checkpoints map[int64]appstore.ChainIngestCheckpoint
	upserts     []appstore.ChainIngestCheckpoint
}

func (f *chainIngestServiceStoreFake) GetChainIngestCheckpoint(_ context.Context, chainID int64) (*appstore.ChainIngestCheckpoint, error) {
	if item, ok := f.checkpoints[chainID]; ok {
		return &item, nil
	}
	switch chainID {
	case 1:
		return &appstore.ChainIngestCheckpoint{ChainID: 1, ChainName: "Ethereum Mainnet", Enabled: true, Status: appstore.ChainIngestStatusStopped}, nil
	case 56:
		return &appstore.ChainIngestCheckpoint{ChainID: 56, ChainName: "BSC Mainnet", Enabled: true, Status: appstore.ChainIngestStatusStopped}, nil
	default:
		return nil, nil
	}
}

func (f *chainIngestServiceStoreFake) UpsertChainIngestCheckpoint(_ context.Context, item appstore.ChainIngestCheckpoint) (*appstore.ChainIngestCheckpoint, error) {
	if f.checkpoints == nil {
		f.checkpoints = map[int64]appstore.ChainIngestCheckpoint{}
	}
	f.checkpoints[item.ChainID] = item
	f.upserts = append(f.upserts, item)
	return &item, nil
}

func TestGetChainIngestStatusDefaultsExistingChainToStopped(t *testing.T) {
	service := &Service{store: &chainIngestServiceStoreFake{}}

	resp, err := service.GetChainIngestStatus(context.Background(), &applicationpkg.GetChainIngestStatusRequest{ChainId: 1})
	if err != nil {
		t.Fatalf("GetChainIngestStatus: %v", err)
	}
	if resp.ChainID != 1 || resp.Status != appstore.ChainIngestStatusStopped || resp.Name != "Ethereum Mainnet" || !resp.Enabled {
		t.Fatalf("status = %#v, want existing ETH chain stopped", resp)
	}
}

func TestStartStopChainIngestPreservesCheckpointProgress(t *testing.T) {
	cursorHash := common.HexToHash("0x100")
	finalizedHash := common.HexToHash("0x200")
	store := &chainIngestServiceStoreFake{checkpoints: map[int64]appstore.ChainIngestCheckpoint{
		56: {
			ChainID:              56,
			ChainName:            "BSC Mainnet",
			Enabled:              true,
			Status:               appstore.ChainIngestStatusStopped,
			CursorBlockNumber:    123,
			CursorBlockHash:      cursorHash,
			FinalizedBlockNumber: 120,
			FinalizedBlockHash:   finalizedHash,
		},
	}}
	service := &Service{store: store}

	started, err := service.StartChainIngest(context.Background(), &applicationpkg.StartChainIngestRequest{ChainId: 56})
	if err != nil {
		t.Fatalf("StartChainIngest: %v", err)
	}
	if started.Status != appstore.ChainIngestStatusRunning || started.CursorBlockNumber != 123 || started.FinalizedBlockNumber != 120 {
		t.Fatalf("started status = %#v, want running with preserved progress", started)
	}

	stopped, err := service.StopChainIngest(context.Background(), &applicationpkg.StopChainIngestRequest{ChainId: 56})
	if err != nil {
		t.Fatalf("StopChainIngest: %v", err)
	}
	if stopped.Status != appstore.ChainIngestStatusStopped || stopped.CursorBlockHash != cursorHash.Hex() || stopped.FinalizedBlockHash != finalizedHash.Hex() {
		t.Fatalf("stopped status = %#v, want stopped with preserved hashes", stopped)
	}
	if len(store.upserts) != 2 || store.upserts[0].Status != appstore.ChainIngestStatusRunning || store.upserts[1].Status != appstore.ChainIngestStatusStopped {
		t.Fatalf("upserts = %#v, want running then stopped", store.upserts)
	}
}
