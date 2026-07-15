package scanner

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/useryege/athena/internal/token/discovery"
)

func (r *chainRunner) projectCandidatesFromBlock(block *types.Block) ([]discovery.ProjectCandidate, error) {
	if block == nil {
		return nil, nil
	}
	candidates := make([]discovery.ProjectCandidate, 0)
	for txIndex, tx := range block.Transactions() {
		if tx == nil || tx.To() != nil {
			continue
		}
		txSender, err := types.Sender(r.signer, tx)
		if err != nil {
			return nil, fmt.Errorf("derive transaction sender for tx %s: %w", tx.Hash().Hex(), err)
		}
		contract := crypto.CreateAddress(txSender, tx.Nonce())
		if contract == (common.Address{}) {
			continue
		}
		candidates = append(candidates, discovery.ProjectCandidate{
			ChainID:     r.opts.chainID,
			Contract:    contract,
			TxSender:    txSender,
			TxHash:      tx.Hash(),
			TxIndex:     uint64(txIndex),
			BlockNumber: block.NumberU64(),
			BlockTime:   block.Time(),
			Status:      discovery.ProjectCandidateStatusPending,
		})
	}
	return candidates, nil
}
