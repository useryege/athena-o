package redisrepo

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestKeyspaceProjectKeys(t *testing.T) {
	keys := NewKeyspace("application")
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	creator := common.HexToAddress("0x1000000000000000000000000000000000000002")
	pair := common.HexToAddress("0x1000000000000000000000000000000000000003")

	assertEqual(t, keys.ProjectBase(contract), "application:project:"+contract.Hex()+":base")
	assertEqual(t, keys.ProjectChainState(contract), "application:project:"+contract.Hex()+":chain_state")
	assertEqual(t, keys.ProjectSimulation(contract), "application:project:"+contract.Hex()+":simulation")
	assertEqual(t, keys.ProjectReport(contract), "application:project:"+contract.Hex()+":report")
	assertEqual(t, keys.ProjectBytecodeFact(contract), "application:project:"+contract.Hex()+":bytecode_fact")
	assertEqual(t, keys.ProjectAveDetail(contract), "application:project:"+contract.Hex()+":ave_detail")
	assertEqual(t, keys.ProjectGenesisWallets(contract), "application:project:"+contract.Hex()+":genesis_wallets")
	assertEqual(t, keys.ProjectCreatorHistory(contract), "application:project:"+contract.Hex()+":creator_history")
	assertEqual(t, keys.ProjectComponentState(contract, "report"), "application:project:"+contract.Hex()+":component_state:report")
	assertEqual(t, keys.ProjectEventLogItem(contract, "abc"), "application:project:"+contract.Hex()+":event_log:abc")
	assertEqual(t, keys.ProjectIndexBase(), "application:index:project:base")
	assertEqual(t, keys.ProjectIndexCreator(creator), "application:index:project:creator:"+creator.Hex())
	assertEqual(t, keys.ProjectIndexPair(pair), "application:index:project:pair:"+pair.Hex())
	assertEqual(t, keys.ProjectIndexComponentNextRun("report"), "application:index:component:next_run:report")
	assertEqual(t, keys.ProjectComponentStream(), "application:stream:component")
}

func TestKeyspaceBufferKeys(t *testing.T) {
	keys := NewKeyspace("application")
	creator := common.HexToAddress("0x1000000000000000000000000000000000000002")
	pair := common.HexToAddress("0x1000000000000000000000000000000000000003")
	contract := common.HexToAddress("0x1000000000000000000000000000000000000004")

	assertEqual(t, keys.BufferItem("base", "member"), "application:buffer:base:member")
	assertEqual(t, keys.BufferDirty("base"), "application:dirty:base")
	assertEqual(t, keys.BufferIndexBase(), "application:index:buffer:base")
	assertEqual(t, keys.BufferIndexCreator(creator), "application:index:buffer:creator:"+creator.Hex())
	assertEqual(t, keys.BufferIndexPair(pair), "application:index:buffer:pair:"+pair.Hex())
	assertEqual(t, keys.BufferIndexComponentNextRun("ave_detail"), "application:index:buffer:component_next_run:ave_detail")
	assertEqual(t, keys.BufferIndexEventLog(contract), "application:index:buffer:event_log:"+contract.Hex())
}

func assertEqual(t *testing.T, got string, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("key = %q, want %q", got, want)
	}
}
