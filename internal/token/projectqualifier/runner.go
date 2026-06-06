package projectqualifier

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	tokenstore "github.com/useryege/athena/internal/token/store"
	"github.com/useryege/athena/util/ethws"
)

type qualifierRunnerOptions struct {
	store          *tokenstore.SQLStore
	nodeWSURLs     map[int64]string
	nodeWSUseProxy bool
	pollInterval   time.Duration
	candidateLimit int32
}

type qualifierRunner struct {
	opts qualifierRunnerOptions

	clientMu sync.Mutex
	clients  map[int64]*ethclient.Client
}

func newQualifierRunner(opts qualifierRunnerOptions) *qualifierRunner {
	return &qualifierRunner{
		opts:    opts,
		clients: make(map[int64]*ethclient.Client),
	}
}

func (r *qualifierRunner) run(ctx context.Context) {
	defer r.close()
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		if err := r.processPendingCandidates(ctx); err != nil {
			log.WithError(err).Error("token project qualifier failed")
		}
		if !sleepContext(ctx, r.opts.pollInterval) {
			return
		}
	}
}

func (r *qualifierRunner) processPendingCandidates(ctx context.Context) error {
	candidates, err := r.opts.store.ListProjectCandidatesByStatus(ctx, tokenstore.ProjectCandidateStatusPending, r.opts.candidateLimit)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		log.Debug("token project qualifier has no pending candidates")
		return nil
	}
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := r.processCandidate(ctx, candidate); err != nil {
			return err
		}
	}
	log.WithField("candidate_count", len(candidates)).Debug("token project qualifier processed candidate batch")
	return nil
}

func (r *qualifierRunner) processCandidate(ctx context.Context, candidate tokenstore.ProjectCandidate) error {
	client, err := r.ensureClient(ctx, candidate.ChainID)
	if err != nil {
		return err
	}
	code, err := client.CodeAt(ctx, candidate.Contract, nil)
	if err != nil {
		r.resetClient(candidate.ChainID)
		return fmt.Errorf("fetch contract code chain_id=%d contract=%s: %w", candidate.ChainID, candidate.Contract.Hex(), err)
	}
	if len(code) == 0 {
		if _, err := r.opts.store.MarkProjectCandidateStatus(ctx, candidate.ID, tokenstore.ProjectCandidateStatusRejected); err != nil {
			return err
		}
		log.WithFields(log.Fields{
			"candidate_id": candidate.ID,
			"chain_id":     candidate.ChainID,
			"contract":     candidate.Contract.Hex(),
		}).Info("token project qualifier rejected candidate without code")
		return nil
	}
	codeHash := crypto.Keccak256Hash(code)
	if _, err := r.opts.store.QualifyProjectCandidate(ctx, candidate, codeHash); err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"candidate_id": candidate.ID,
		"chain_id":     candidate.ChainID,
		"contract":     candidate.Contract.Hex(),
		"code_hash":    codeHash.Hex(),
	}).Info("token project qualifier qualified candidate")
	return nil
}

func (r *qualifierRunner) ensureClient(ctx context.Context, chainID int64) (*ethclient.Client, error) {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if client := r.clients[chainID]; client != nil {
		return client, nil
	}
	nodeWSURL := r.opts.nodeWSURLs[chainID]
	if nodeWSURL == "" {
		return nil, errNodeWSURLRequired(chainID)
	}
	client, err := ethws.DialContext(ctx, nodeWSURL, r.opts.nodeWSUseProxy)
	if err != nil {
		return nil, err
	}
	nodeChainID, err := client.ChainID(ctx)
	if err != nil {
		client.Close()
		return nil, err
	}
	if nodeChainID == nil || nodeChainID.Int64() != chainID {
		client.Close()
		return nil, fmt.Errorf("token project qualifier node returned chain_id %v for configured chain_id %d", nodeChainID, chainID)
	}
	r.clients[chainID] = client
	log.WithFields(log.Fields{
		"chain_id":   chainID,
		"chain_name": chainName(chainID),
	}).Info("token project qualifier connected to node websocket")
	return client, nil
}

func (r *qualifierRunner) resetClient(chainID int64) {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if client := r.clients[chainID]; client != nil {
		client.Close()
		delete(r.clients, chainID)
	}
}

func (r *qualifierRunner) close() {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	for chainID, client := range r.clients {
		if client != nil {
			client.Close()
		}
		delete(r.clients, chainID)
	}
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
