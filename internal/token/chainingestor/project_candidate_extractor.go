package chainingestor

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	tokenstore "github.com/useryege/athena/internal/token/store"
)

func (r *chainRunner) projectCandidatesFromBlock(block *types.Block) ([]tokenstore.ProjectCandidate, error) {
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
		contract := crypto.CreateAddress(creator, tx.Nonce())
		if contract == (common.Address{}) {
			continue
		}
		candidates = append(candidates, tokenstore.ProjectCandidate{
			ChainID:     r.opts.chainID,
			Contract:    contract,
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
