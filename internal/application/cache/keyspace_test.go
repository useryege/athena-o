package cache

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestKeyspaceProjectKeys(t *testing.T) {
	keys := NewKeyspace("application")
	chainID := int64(56)
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	creator := common.HexToAddress("0x1000000000000000000000000000000000000002")
	pair := common.HexToAddress("0x1000000000000000000000000000000000000003")

	assertEqual(t, keys.Project(chainID, contract), "application:project:56:"+contract.Hex())
	assertEqual(t, keys.ProjectChainState(chainID, contract), "application:project:56:"+contract.Hex()+":chain_state")
	assertEqual(t, keys.ProjectSimulation(chainID, contract), "application:project:56:"+contract.Hex()+":simulation")
	assertEqual(t, keys.ProjectGenesisWallets(chainID, contract), "application:project:56:"+contract.Hex()+":genesis_wallets")
	assertEqual(t, keys.ProjectCreatorHistory(chainID, contract), "application:project:56:"+contract.Hex()+":creator_history")
	assertEqual(t, keys.ProjectComponentState(chainID, contract, "bytecode_fact"), "application:project:56:"+contract.Hex()+":component_state:bytecode_fact")
	assertEqual(t, keys.ProjectIndex(chainID), "application:index:project:56")
	assertEqual(t, keys.ProjectIndexCreator(chainID, creator), "application:index:project:56:creator:"+creator.Hex())
	assertEqual(t, keys.ProjectIndexPair(chainID, pair), "application:index:project:56:pair:"+pair.Hex())
	assertEqual(t, keys.ProjectIndexComponentNextRun(chainID, "bytecode_fact"), "application:index:component:56:next_run:bytecode_fact")
}

func assertEqual(t *testing.T, got string, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("key = %q, want %q", got, want)
	}
}
