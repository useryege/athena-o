package projectqualifier

import (
	"context"
	"fmt"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	athenacommon "github.com/useryege/athena/common"
	tokenstore "github.com/useryege/athena/internal/token/store"
	"github.com/useryege/athena/util/ethws"
)

type qualifierRunnerOptions struct {
	store           *tokenstore.SQLStore
	chainIDs        []int64
	nodeWSURLs      map[int64]string
	athenaContracts map[int64]ethcommon.Address
	nodeWSUseProxy  bool
	pollInterval    time.Duration
	candidateLimit  int32
}

type qualifierRunner struct {
	opts qualifierRunnerOptions

	clientMu   sync.Mutex
	clients    map[int64]*ethclient.Client
	validators map[int64]*athenaValidator
}

func newQualifierRunner(opts qualifierRunnerOptions) *qualifierRunner {
	return &qualifierRunner{
		opts:       opts,
		clients:    make(map[int64]*ethclient.Client),
		validators: make(map[int64]*athenaValidator),
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
	if len(r.opts.chainIDs) == 0 {
		log.Debug("token project qualifier has no enabled chains")
		return nil
	}
	totalCandidates := 0
	for _, chainID := range r.opts.chainIDs {
		if err := ctx.Err(); err != nil {
			return err
		}
		page, err := r.opts.store.ListProjectCandidates(ctx, chainID, tokenstore.ProjectCandidateStatusPending, 1, r.opts.candidateLimit)
		if err != nil {
			return err
		}
		if len(page.Items) == 0 {
			continue
		}
		totalCandidates += len(page.Items)
		if err := r.processChainCandidates(ctx, chainID, page.Items); err != nil {
			return err
		}
	}
	if totalCandidates == 0 {
		log.Debug("token project qualifier has no pending candidates")
		return nil
	}
	log.WithField("candidate_count", totalCandidates).Debug("token project qualifier processed candidate batch")
	return nil
}

func (r *qualifierRunner) processChainCandidates(ctx context.Context, chainID int64, candidates []tokenstore.ProjectCandidate) error {
	client, err := r.ensureClient(ctx, chainID)
	if err != nil {
		return err
	}
	validator, err := r.ensureValidator(ctx, chainID)
	if err != nil {
		return err
	}
	contracts := make([]ethcommon.Address, 0, len(candidates))
	for _, candidate := range candidates {
		contracts = append(contracts, candidate.Contract)
	}
	validations, err := validator.validateERC20(ctx, contracts)
	if err != nil {
		r.resetChain(chainID)
		return fmt.Errorf("validate ERC20 chain_id=%d candidate_count=%d: %w", chainID, len(candidates), err)
	}
	if len(validations) != len(candidates) {
		r.resetChain(chainID)
		return fmt.Errorf("validate ERC20 chain_id=%d returned %d results for %d candidates", chainID, len(validations), len(candidates))
	}
	for i, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := r.processValidatedCandidate(ctx, client, candidate, validations[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *qualifierRunner) processValidatedCandidate(ctx context.Context, client *ethclient.Client, candidate tokenstore.ProjectCandidate, validation tokenValidation) error {
	if !validation.IsValidERC20 {
		return r.rejectCandidate(ctx, candidate, "token project qualifier rejected non ERC20 candidate")
	}
	code, err := client.CodeAt(ctx, candidate.Contract, nil)
	if err != nil {
		r.resetChain(candidate.ChainID)
		return fmt.Errorf("fetch contract code chain_id=%d contract=%s: %w", candidate.ChainID, candidate.Contract.Hex(), err)
	}
	if len(code) == 0 {
		return r.rejectCandidate(ctx, candidate, "token project qualifier rejected candidate without code")
	}
	codeHash := crypto.Keccak256Hash(code)
	receipt, err := fetchProjectCreationReceipt(ctx, client, candidate)
	if err != nil {
		r.resetChain(candidate.ChainID)
		return fmt.Errorf("fetch creation receipt chain_id=%d tx_hash=%s: %w", candidate.ChainID, candidate.TxHash.Hex(), err)
	}
	initialRecipients := extractInitialRecipientWallets(receipt.Logs, candidate.Contract, initialRecipientWalletLimit)
	relatedWallets := buildProjectRelatedWallets(candidate, initialRecipients)
	if _, err := r.opts.store.QualifyProjectCandidate(ctx, candidate, codeHash, validation.WethPair, validation.UsdtPair, relatedWallets); err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"candidate_id":            candidate.ID,
		"chain_id":                candidate.ChainID,
		"contract":                candidate.Contract.Hex(),
		"code_hash":               codeHash.Hex(),
		"weth_pair":               validation.WethPair.Hex(),
		"usdt_pair":               validation.UsdtPair.Hex(),
		"related_wallet_count":    len(relatedWallets),
		"initial_recipient_count": len(initialRecipients),
	}).Info("token project qualifier qualified candidate")
	return nil
}

func (r *qualifierRunner) rejectCandidate(ctx context.Context, candidate tokenstore.ProjectCandidate, message string) error {
	if _, err := r.opts.store.MarkProjectCandidateStatus(ctx, candidate.ID, tokenstore.ProjectCandidateStatusRejected); err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"candidate_id": candidate.ID,
		"chain_id":     candidate.ChainID,
		"contract":     candidate.Contract.Hex(),
	}).Info(message)
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
		"chain_name": athenacommon.ChainName(chainID),
	}).Info("token project qualifier connected to node websocket")
	return client, nil
}

func (r *qualifierRunner) ensureValidator(ctx context.Context, chainID int64) (*athenaValidator, error) {
	r.clientMu.Lock()
	if validator := r.validators[chainID]; validator != nil {
		r.clientMu.Unlock()
		return validator, nil
	}
	r.clientMu.Unlock()

	client, err := r.ensureClient(ctx, chainID)
	if err != nil {
		return nil, err
	}
	athenaContract := r.opts.athenaContracts[chainID]
	if athenaContract == (ethcommon.Address{}) {
		return nil, errAthenaContractRequired(chainID)
	}
	validator, err := newAthenaValidator(athenaContract, client)
	if err != nil {
		return nil, err
	}
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if current := r.validators[chainID]; current != nil {
		return current, nil
	}
	r.validators[chainID] = validator
	log.WithFields(log.Fields{
		"chain_id":        chainID,
		"chain_name":      athenacommon.ChainName(chainID),
		"athena_contract": athenaContract.Hex(),
	}).Info("token project qualifier initialized ATHENA validator")
	return validator, nil
}

func (r *qualifierRunner) resetChain(chainID int64) {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if client := r.clients[chainID]; client != nil {
		client.Close()
		delete(r.clients, chainID)
	}
	delete(r.validators, chainID)
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
	for chainID := range r.validators {
		delete(r.validators, chainID)
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
