package projectvalidator

import (
	"context"
	"fmt"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/domain"
)

func (r *validatorRunner) processPendingCandidates(ctx context.Context) error {
	if len(r.opts.chainIDs) == 0 {
		log.Debug("token project validator has no enabled chains")
		return nil
	}
	totalCandidates := 0
	for _, chainID := range r.opts.chainIDs {
		if err := ctx.Err(); err != nil {
			return err
		}
		lockToken, err := uuid.NewRandom()
		if err != nil {
			return fmt.Errorf("generate project candidate validation lock token: %w", err)
		}
		candidates, err := r.opts.store.ClaimProjectCandidateValidations(ctx, chainID, lockToken, candidateLease, r.opts.candidateLimit)
		if err != nil {
			return err
		}
		if len(candidates) == 0 {
			continue
		}
		totalCandidates += len(candidates)
		stopRenewal := r.renewCandidateValidationClaims(ctx, lockToken)
		processErr := r.processChainCandidates(ctx, chainID, candidates)
		stopRenewal()
		releaseErr := r.opts.store.ReleaseProjectCandidateValidationClaims(ctx, lockToken)
		if processErr != nil {
			if releaseErr != nil {
				log.WithError(releaseErr).WithField("validation_lock_token", lockToken.String()).Error("token project validator failed to release claims")
			}
			return processErr
		}
		if releaseErr != nil {
			return releaseErr
		}
	}
	if totalCandidates == 0 {
		log.Debug("token project validator has no pending candidates")
		return nil
	}
	log.WithField("candidate_count", totalCandidates).Debug("token project validator processed candidate batch")
	return nil
}

func (r *validatorRunner) renewCandidateValidationClaims(ctx context.Context, lockToken uuid.UUID) func() {
	renewCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(candidateLeaseRenew)
		defer ticker.Stop()
		for {
			select {
			case <-renewCtx.Done():
				return
			case <-ticker.C:
				if err := r.opts.store.RenewProjectCandidateValidationClaims(renewCtx, lockToken, candidateLease); err != nil && renewCtx.Err() == nil {
					log.WithError(err).WithField("validation_lock_token", lockToken.String()).Error("token project validator failed to renew claims")
				}
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
}

func (r *validatorRunner) processChainCandidates(ctx context.Context, chainID int64, candidates []domain.ProjectCandidate) error {
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
	concurrency := r.opts.candidateConcurrency
	if concurrency <= 0 {
		concurrency = 1
	}
	semaphore := make(chan struct{}, concurrency)
	errors := make(chan error, len(candidates))
	var workers sync.WaitGroup
	for i, candidate := range candidates {
		validation := validations[i]
		semaphore <- struct{}{}
		workers.Go(func() {
			defer func() { <-semaphore }()
			if err := r.processValidatedCandidate(ctx, client, candidate, validation); err != nil {
				errors <- err
			}
		})
	}
	workers.Wait()
	close(errors)
	for err := range errors {
		r.resetChain(chainID)
		return err
	}
	return nil
}

func (r *validatorRunner) processValidatedCandidate(ctx context.Context, client *ethclient.Client, candidate domain.ProjectCandidate, validation tokenValidation) error {
	if !validation.IsValidERC20 {
		return r.rejectCandidate(ctx, candidate, "token project validator rejected non ERC20 candidate")
	}
	code, err := client.CodeAt(ctx, candidate.Contract, nil)
	if err != nil {
		return fmt.Errorf("fetch contract code chain_id=%d contract=%s: %w", candidate.ChainID, candidate.Contract.Hex(), err)
	}
	if len(code) == 0 {
		return r.rejectCandidate(ctx, candidate, "token project validator rejected candidate without code")
	}
	codeHash := crypto.Keccak256Hash(code)
	receipt, err := fetchProjectCreationReceipt(ctx, client, candidate)
	if err != nil {
		return fmt.Errorf("fetch creation receipt chain_id=%d tx_hash=%s: %w", candidate.ChainID, candidate.TxHash.Hex(), err)
	}
	initialRecipientCandidates := extractInitialRecipientWallets(receipt.Logs, candidate.Contract)
	initialRecipients, err := filterInitialRecipientWallets(ctx, client, initialRecipientCandidates, initialRecipientWalletLimit)
	if err != nil {
		return fmt.Errorf("filter initial recipient contracts chain_id=%d contract=%s: %w", candidate.ChainID, candidate.Contract.Hex(), err)
	}
	relatedWallets := buildProjectRelatedWallets(candidate, initialRecipients)
	projectInitialRecipients := buildProjectInitialRecipients(candidate, initialRecipients, validation.TotalSupply)
	token := domain.ProjectTokenMetadata{
		Name:        validation.Name,
		Symbol:      validation.Symbol,
		Decimals:    validation.Decimals,
		TotalSupply: validation.TotalSupply,
	}
	if _, err := r.opts.store.ValidateProjectCandidate(ctx, candidate, codeHash, token, validation.WethPair, validation.UsdtPair, relatedWallets, projectInitialRecipients); err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"candidate_id":                   candidate.ID,
		"chain_id":                       candidate.ChainID,
		"contract":                       candidate.Contract.Hex(),
		"code_hash":                      codeHash.Hex(),
		"weth_pair":                      validation.WethPair.Hex(),
		"usdt_pair":                      validation.UsdtPair.Hex(),
		"related_wallet_count":           len(relatedWallets),
		"initial_recipient_count":        len(initialRecipients),
		"initial_recipient_detail_count": len(projectInitialRecipients),
	}).Info("token project validator validated candidate")
	return nil
}

func (r *validatorRunner) rejectCandidate(ctx context.Context, candidate domain.ProjectCandidate, message string) error {
	if err := r.opts.store.RejectProjectCandidate(ctx, candidate); err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"candidate_id": candidate.ID,
		"chain_id":     candidate.ChainID,
		"contract":     candidate.Contract.Hex(),
	}).Info(message)
	return nil
}
