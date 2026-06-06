package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application/events"
	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/internal/application/ingest"
	appstore "github.com/useryege/athena/internal/application/store"
	"github.com/useryege/athena/util/ethws"
)

func runChainIngestorMode(ctx context.Context, opts Options) error {
	if opts.Store == nil {
		return fmt.Errorf("application store is required for chain-ingestor mode")
	}
	if opts.ChainID <= 0 {
		return fmt.Errorf("chain-ingestor mode requires --chain-id or ATHENA_APPLICATION_CHAIN_ID")
	}
	producer := events.NewDirectProducer(opts.Store)
	log.WithField("chain_id", opts.ChainID).Info("athena-application chain ingestor started")
	return runUntilShutdown(ctx, func(runCtx context.Context) error {
		return runLazyChainIngestor(runCtx, opts, producer)
	})
}

func runLazyChainIngestor(ctx context.Context, opts Options, producer ingest.Producer) error {
	pollInterval := opts.IngestPollInterval
	if pollInterval <= 0 {
		pollInterval = 3 * time.Second
	}
	var nodeClient *ethclient.Client
	var ingestorInstance *ingest.Ingestor
	defer func() {
		if nodeClient != nil {
			nodeClient.Close()
		}
	}()
	for {
		checkpoint, err := opts.Store.GetChainIngestCheckpoint(ctx, opts.ChainID)
		if err != nil {
			return err
		}
		if checkpoint == nil || checkpoint.Status != appstore.ChainIngestStatusRunning {
			if nodeClient != nil {
				nodeClient.Close()
				nodeClient = nil
				ingestorInstance = nil
				log.WithField("chain_id", opts.ChainID).Info("athena-application chain ingestor paused")
			}
			if !waitForIngestPoll(ctx, pollInterval) {
				return ctx.Err()
			}
			continue
		}
		if ingestorInstance == nil {
			if strings.TrimSpace(opts.NodeWSURL) == "" {
				return fmt.Errorf("node websocket URL is required for running chain-ingestor chain %d", opts.ChainID)
			}
			athenaContractAddress, err := parseRequiredAddress("ATHENA contract address", opts.AthenaContract, "--athena-contract", "ATHENA_APPLICATION_ATHENA_CONTRACT")
			if err != nil {
				return err
			}
			nodeClient, err = ethws.DialContext(ctx, opts.NodeWSURL, opts.NodeWSUseProxy)
			if err != nil {
				return fmt.Errorf("connect chain ingestor node: %w", err)
			}
			reader := ingest.NewEVMReader(nodeClient, opts.ChainID)
			chainValidator, err := evm.NewAthenaChainValidator(nodeClient, athenaContractAddress)
			if err != nil {
				nodeClient.Close()
				nodeClient = nil
				return err
			}
			ingestorInstance, err = ingest.NewIngestor(ingest.Options{
				ChainID:           opts.ChainID,
				ConfirmationDepth: opts.ConfirmationDepth,
				StartBlock:        opts.StartBlock,
				Reader:            reader,
				Producer:          producer,
				Store:             opts.Store,
				ChainValidator:    chainValidator,
			})
			if err != nil {
				nodeClient.Close()
				nodeClient = nil
				return err
			}
			log.WithField("chain_id", opts.ChainID).Info("athena-application chain ingestor activated")
		}
		if _, err := ingestorInstance.ProcessOnce(ctx); err != nil {
			return err
		}
		if !waitForIngestPoll(ctx, pollInterval) {
			return ctx.Err()
		}
	}
}

func waitForIngestPoll(ctx context.Context, interval time.Duration) bool {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
