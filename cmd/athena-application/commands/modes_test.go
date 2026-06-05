package commands

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	appevents "github.com/useryege/athena/internal/application/events"
	appstore "github.com/useryege/athena/internal/application/store"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

type chainIngestQuerierFake struct {
	appsqlc.Querier

	row appsqlc.GetChainIngestCheckpointRow
}

func (f *chainIngestQuerierFake) GetChainIngestCheckpoint(context.Context, int64) (appsqlc.GetChainIngestCheckpointRow, error) {
	return f.row, nil
}

type chainIngestProducerNoop struct{}

func (chainIngestProducerNoop) Publish(context.Context, string, string, appevents.Envelope) error {
	return nil
}

func TestRunLazyChainIngestorStoppedDoesNotRequireNodeURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	store := appstore.NewSQLStoreWithQuerier(&chainIngestQuerierFake{row: appsqlc.GetChainIngestCheckpointRow{
		ChainID:   56,
		ChainName: "BSC Mainnet",
		Enabled:   true,
		Status:    appstore.ChainIngestStatusStopped,
	}})

	err := runLazyChainIngestor(ctx, runtimeOptions{
		ChainID:            56,
		Store:              store,
		IngestPollInterval: time.Millisecond,
	}, chainIngestProducerNoop{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("runLazyChainIngestor error = %v, want context deadline after stopped polling", err)
	}
}

func TestRunLazyChainIngestorRunningRequiresNodeURL(t *testing.T) {
	store := appstore.NewSQLStoreWithQuerier(&chainIngestQuerierFake{row: appsqlc.GetChainIngestCheckpointRow{
		ChainID:   1,
		ChainName: "Ethereum Mainnet",
		Enabled:   true,
		Status:    appstore.ChainIngestStatusRunning,
	}})

	err := runLazyChainIngestor(context.Background(), runtimeOptions{
		ChainID:            1,
		Store:              store,
		IngestPollInterval: time.Millisecond,
	}, chainIngestProducerNoop{})
	if err == nil || !strings.Contains(err.Error(), "node websocket URL is required") {
		t.Fatalf("runLazyChainIngestor error = %v, want missing node URL after running checkpoint", err)
	}
}

func TestRunLazyChainIngestorRunningRequiresAthenaContract(t *testing.T) {
	store := appstore.NewSQLStoreWithQuerier(&chainIngestQuerierFake{row: appsqlc.GetChainIngestCheckpointRow{
		ChainID:   1,
		ChainName: "Ethereum Mainnet",
		Enabled:   true,
		Status:    appstore.ChainIngestStatusRunning,
	}})

	err := runLazyChainIngestor(context.Background(), runtimeOptions{
		ChainID:            1,
		NodeWSURL:          "ws://example.invalid:8546",
		Store:              store,
		IngestPollInterval: time.Millisecond,
	}, chainIngestProducerNoop{})
	if err == nil || !strings.Contains(err.Error(), "ATHENA contract address is required") {
		t.Fatalf("runLazyChainIngestor error = %v, want missing ATHENA contract", err)
	}
}
