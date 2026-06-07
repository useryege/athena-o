package projectqualifier

import (
	"context"
	"fmt"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	tokenstore "github.com/useryege/athena/internal/token/store"
)

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
