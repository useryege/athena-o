package chainingestor

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	tokenstore "github.com/useryege/athena/internal/token/store"
	"github.com/useryege/athena/util/ethws"
)

type chainRunnerOptions struct {
	store          *tokenstore.SQLStore
	chainID        int64
	chainName      string
	nodeWSURL      string
	nodeWSUseProxy bool
	pollInterval   time.Duration
}

type chainRunner struct {
	opts   chainRunnerOptions
	signer types.Signer

	clientMu sync.Mutex
	client   *ethclient.Client
}

func newChainRunner(opts chainRunnerOptions) *chainRunner {
	return &chainRunner{
		opts:   opts,
		signer: types.LatestSignerForChainID(big.NewInt(opts.chainID)),
	}
}

func (r *chainRunner) run(ctx context.Context) {
	defer r.close()
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		if err := r.processAvailableBlocks(ctx); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"chain_id":   r.opts.chainID,
				"chain_name": r.opts.chainName,
			}).Error("token chain ingestor scan failed")
			if !sleepContext(ctx, r.opts.pollInterval) {
				return
			}
			continue
		}
		if !sleepContext(ctx, r.opts.pollInterval) {
			return
		}
	}
}

func (r *chainRunner) processAvailableBlocks(ctx context.Context) error {
	checkpoint, err := r.opts.store.GetChainIngestCheckpoint(ctx, r.opts.chainID)
	if err != nil {
		return err
	}
	if checkpoint == nil {
		return fmt.Errorf("token chain ingestor checkpoint missing for chain %d", r.opts.chainID)
	}
	if !checkpoint.Enabled || checkpoint.Status != tokenstore.ChainIngestStatusRunning {
		return nil
	}
	client, err := r.ensureClient(ctx)
	if err != nil {
		return err
	}
	latest, err := client.BlockNumber(ctx)
	if err != nil {
		r.resetClient()
		return err
	}
	next := uint64(1)
	if checkpoint.CursorBlockNumber > 0 {
		next = checkpoint.CursorBlockNumber + 1
	}
	if next > latest {
		log.WithFields(log.Fields{
			"chain_id":            r.opts.chainID,
			"latest_block_number": latest,
			"cursor_block_number": checkpoint.CursorBlockNumber,
		}).Debug("token chain ingestor is caught up")
		return nil
	}
	for number := next; number <= latest; number++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		checkpoint, err = r.opts.store.GetChainIngestCheckpoint(ctx, r.opts.chainID)
		if err != nil {
			return err
		}
		if checkpoint == nil || !checkpoint.Enabled || checkpoint.Status != tokenstore.ChainIngestStatusRunning {
			return nil
		}
		block, err := client.BlockByNumber(ctx, new(big.Int).SetUint64(number))
		if err != nil {
			r.resetClient()
			return err
		}
		candidates, err := r.projectCandidatesFromBlock(ctx, client, block)
		if err != nil {
			r.resetClient()
			return err
		}
		for _, candidate := range candidates {
			if _, err := r.opts.store.UpsertProjectCandidate(ctx, candidate); err != nil {
				return err
			}
		}
		if _, err := r.opts.store.UpsertChainIngestCheckpoint(ctx, tokenstore.ChainIngestCheckpoint{
			ChainID:              r.opts.chainID,
			FinalizedBlockNumber: block.NumberU64(),
			CursorBlockNumber:    block.NumberU64(),
			Status:               tokenstore.ChainIngestStatusRunning,
		}); err != nil {
			return err
		}
		log.WithFields(log.Fields{
			"chain_id":         r.opts.chainID,
			"block_number":     block.NumberU64(),
			"candidate_count":  len(candidates),
			"latest_block_num": latest,
		}).Debug("token chain ingestor processed block")
	}
	return nil
}

func (r *chainRunner) projectCandidatesFromBlock(ctx context.Context, client *ethclient.Client, block *types.Block) ([]tokenstore.ProjectCandidate, error) {
	if block == nil {
		return nil, nil
	}
	candidates := make([]tokenstore.ProjectCandidate, 0)
	for txIndex, tx := range block.Transactions() {
		if tx == nil || tx.To() != nil {
			continue
		}
		creator, err := types.Sender(r.signer, tx)
		if err != nil {
			return nil, fmt.Errorf("derive contract creator for tx %s: %w", tx.Hash().Hex(), err)
		}
		receipt, err := client.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			return nil, err
		}
		if receipt == nil || receipt.Status != types.ReceiptStatusSuccessful || receipt.ContractAddress == (common.Address{}) {
			continue
		}
		candidates = append(candidates, tokenstore.ProjectCandidate{
			ChainID:     r.opts.chainID,
			Contract:    receipt.ContractAddress,
			Creator:     creator,
			TxHash:      tx.Hash(),
			TxIndex:     uint64(txIndex),
			BlockNumber: block.NumberU64(),
			BlockTime:   block.Time(),
			Status:      tokenstore.ProjectCandidateStatusPending,
		})
	}
	return candidates, nil
}

func (r *chainRunner) ensureClient(ctx context.Context) (*ethclient.Client, error) {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if r.client != nil {
		return r.client, nil
	}
	client, err := ethws.DialContext(ctx, r.opts.nodeWSURL, r.opts.nodeWSUseProxy)
	if err != nil {
		return nil, err
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		client.Close()
		return nil, err
	}
	if chainID == nil || chainID.Int64() != r.opts.chainID {
		client.Close()
		return nil, fmt.Errorf("token chain ingestor node returned chain_id %v for configured chain_id %d", chainID, r.opts.chainID)
	}
	r.client = client
	log.WithFields(log.Fields{
		"chain_id":   r.opts.chainID,
		"chain_name": r.opts.chainName,
	}).Info("token chain ingestor connected to node websocket")
	return client, nil
}

func (r *chainRunner) resetClient() {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if r.client != nil {
		r.client.Close()
		r.client = nil
	}
}

func (r *chainRunner) close() {
	r.resetClient()
}

func sleepContext(ctx context.Context, interval time.Duration) bool {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
