package genesiswallet

import (
	"context"
	"math/big"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

func TestComponentCollectRerunsAfterSuccess(t *testing.T) {
	ctx := context.Background()
	contract := common.BigToAddress(big.NewInt(11))
	creator := common.BigToAddress(big.NewInt(12))
	txHash := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	store := &refreshComponentStoreFake{
		baseByContract: map[common.Address]appstore.Project{
			contract: {
				ChainID:     56,
				Contract:    contract,
				Creator:     creator,
				BlockNumber: 77,
				TxHash:      txHash,
				TxIndex:     1,
			},
		},
		chainStateByContract: map[common.Address]appstore.ProjectChainState{
			contract: {
				ChainID:         56,
				ProjectContract: contract,
				ChainState: athenacontract.AthenaProject{
					Token: athenacontract.AthenaToken{
						TotalSupply: big.NewInt(1000),
					},
				},
			},
		},
		componentStates: map[string]appstore.ProjectComponentState{
			componentKey(contract, appstore.ProjectComponentGenesisWallet): {
				ChainID:         56,
				ProjectContract: contract,
				Component:       appstore.ProjectComponentGenesisWallet,
				Status:          appstore.ProjectComponentStatusSuccess,
				LastSuccessAt:   time.Now().UTC(),
			},
		},
	}
	node := &genesisWalletNodeFake{
		receiptByHash: map[common.Hash]*types.Receipt{
			txHash: {Logs: nil},
		},
	}
	component := NewComponent(Options{ChainID: 56, Store: store, NodeClient: node})
	if component == nil {
		t.Fatal("genesis wallet component is nil")
	}

	if err := component.Collect(ctx, 56, contract); err != nil {
		t.Fatalf("collect: %v", err)
	}

	if got := store.replaceGenesisWalletCalls; got != 1 {
		t.Fatalf("replace genesis wallet calls = %d, want 1", got)
	}
}

type refreshComponentStoreFake struct {
	appstore.Store

	baseByContract       map[common.Address]appstore.Project
	chainStateByContract map[common.Address]appstore.ProjectChainState
	componentStates      map[string]appstore.ProjectComponentState

	replaceGenesisWalletCalls int
}

func (s *refreshComponentStoreFake) GetProjectByContract(_ context.Context, _ int64, contract common.Address) (*appstore.Project, error) {
	item, ok := s.baseByContract[contract]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (s *refreshComponentStoreFake) GetProjectChainState(_ context.Context, _ int64, contract common.Address) (*appstore.ProjectChainState, error) {
	item, ok := s.chainStateByContract[contract]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (s *refreshComponentStoreFake) GetProjectComponentState(_ context.Context, _ int64, contract common.Address, component string) (*appstore.ProjectComponentState, error) {
	item, ok := s.componentStates[componentKey(contract, component)]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (s *refreshComponentStoreFake) UpsertProjectComponentState(_ context.Context, item appstore.ProjectComponentState) error {
	if s.componentStates == nil {
		s.componentStates = map[string]appstore.ProjectComponentState{}
	}
	s.componentStates[componentKey(item.ProjectContract, item.Component)] = item
	return nil
}

func (s *refreshComponentStoreFake) ReplaceProjectGenesisWallets(_ context.Context, _ int64, _ common.Address, _ []appstore.ProjectGenesisWallet) error {
	s.replaceGenesisWalletCalls++
	return nil
}

type genesisWalletNodeFake struct {
	receiptByHash map[common.Hash]*types.Receipt
}

func (n *genesisWalletNodeFake) TransactionReceipt(_ context.Context, txHash common.Hash) (*types.Receipt, error) {
	if receipt, ok := n.receiptByHash[txHash]; ok {
		return receipt, nil
	}
	return nil, nil
}

func (n *genesisWalletNodeFake) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	return nil, nil
}

func componentKey(contract common.Address, component string) string {
	return contract.Hex() + "|" + component
}
