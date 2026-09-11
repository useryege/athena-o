package tradersync

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type versionRead struct {
	method  string
	address common.Address
	hash    common.Hash
	slot    common.Hash
}
type versionFake struct {
	chain            *big.Int
	header           *types.Header
	reads            []versionRead
	code             map[common.Address][]byte
	parentCode       map[common.Address][]byte
	slot, parentSlot []byte
	upgrades         []types.Log
	fail             string
}

func (f *versionFake) ChainID(context.Context) (*big.Int, error) {
	f.reads = append(f.reads, versionRead{method: "chain"})
	if f.fail == "chain" {
		return nil, errors.New("403")
	}
	return f.chain, nil
}
func (f *versionFake) HeaderByHash(_ context.Context, h common.Hash) (*types.Header, error) {
	f.reads = append(f.reads, versionRead{method: "header", hash: h})
	if f.fail == "header" {
		return nil, errors.New("403")
	}
	return f.header, nil
}
func (f *versionFake) CodeAtHash(_ context.Context, a common.Address, h common.Hash) ([]byte, error) {
	f.reads = append(f.reads, versionRead{method: "code", address: a, hash: h})
	if f.fail == "code" {
		return nil, errors.New("403")
	}
	if h == f.header.ParentHash {
		return f.parentCode[a], nil
	}
	return f.code[a], nil
}
func (f *versionFake) StorageAtHash(_ context.Context, a common.Address, s, h common.Hash) ([]byte, error) {
	f.reads = append(f.reads, versionRead{method: "slot", address: a, hash: h, slot: s})
	if f.fail == "slot" {
		return nil, errors.New("403")
	}
	if h == f.header.ParentHash {
		return f.parentSlot, nil
	}
	return f.slot, nil
}
func (f *versionFake) UpgradeLogs(_ context.Context, h common.Hash, a common.Address) ([]types.Log, error) {
	f.reads = append(f.reads, versionRead{method: "upgrades", address: a, hash: h})
	if f.fail == "upgrades" {
		return nil, errors.New("403")
	}
	return f.upgrades, nil
}
func runtimeFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/runtime/" + name + ".hex")
	if err != nil {
		t.Fatal(err)
	}
	b, err = hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(string(b)), "0x"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func versionFixture(t *testing.T, version string) (types.Log, *versionFake) {
	t.Helper()
	h := &types.Header{Number: big.NewInt(100), ParentHash: common.HexToHash("0x123456"), Time: 1700000000, Difficulty: big.NewInt(0)}
	f := &versionFake{upgrades: []types.Log{}, chain: big.NewInt(137), header: h, code: map[common.Address][]byte{}, parentCode: map[common.Address][]byte{}}
	var addr common.Address
	switch version {
	case CoreExchangeVersion:
		addr = common.HexToAddress("0xE111180000d2663C0091e4f400237545B87B996B")
		f.code[addr] = runtimeFixture(t, "core-exchange")
	case NegRiskExchangeVersion:
		addr = common.HexToAddress("0xe2222d279d744050d28e00520010520000310F59")
		f.code[addr] = runtimeFixture(t, "neg-risk-exchange")
	case ComboExchangeVersion:
		addr = common.HexToAddress("0xe3333700cA9d93003F00f0F71f8515005F6c00Aa")
		impl := common.HexToAddress("0x641b40ec414a076b9e79e703fc7bf4ebec248bb7")
		f.code[addr] = runtimeFixture(t, "combo-proxy")
		f.code[impl] = runtimeFixture(t, "combo-implementation")
		f.slot = common.LeftPadBytes(impl.Bytes(), 32)
		f.parentSlot = append([]byte(nil), f.slot...)
	}
	for a, c := range f.code {
		f.parentCode[a] = append([]byte(nil), c...)
	}
	return types.Log{Address: addr, BlockHash: h.Hash(), BlockNumber: 100}, f
}
func TestVersionChecksChainParentAndExactDeploymentBeforeCaching(t *testing.T) {
	for _, version := range []string{CoreExchangeVersion, NegRiskExchangeVersion, ComboExchangeVersion} {
		t.Run(version, func(t *testing.T) {
			l, n := versionFixture(t, version)
			v := NewVersionVerifier(n)
			got, err := v.Verify(context.Background(), l)
			if err != nil || got != version {
				t.Fatal(got, err)
			}
			if len(n.reads) < 4 || n.reads[0].method != "chain" || n.reads[1].method != "header" {
				t.Fatal(n.reads)
			}
			parent, block, upgrades := 0, 0, 0
			for _, r := range n.reads {
				if r.method == "code" || r.method == "slot" {
					if r.hash == l.BlockHash {
						block++
					} else if r.hash == n.header.ParentHash {
						parent++
					} else {
						t.Fatal("used unknown or latest hash", r)
					}
				}
				if r.method == "upgrades" {
					upgrades++
					if r.hash != l.BlockHash || r.address != l.Address {
						t.Fatal("wrong upgrade filter", r)
					}
				}
				if r.method == "slot" && r.slot != common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc") {
					t.Fatal(r)
				}
			}
			if parent == 0 || block == 0 || (version == ComboExchangeVersion && upgrades != 1) {
				t.Fatal(n.reads)
			}
			previous := len(n.reads)
			if _, err = v.Verify(context.Background(), l); err != nil {
				t.Fatal(err)
			}
			if len(n.reads) != previous+1 || n.reads[len(n.reads)-1].method != "chain" {
				t.Fatal("successful immutable block evidence not cached", n.reads)
			}
			// Connection identity is never borrowed from the cache for a wrong-chain node.
			n.chain = big.NewInt(1)
			if _, err = v.Verify(context.Background(), l); status.Code(err) != codes.FailedPrecondition {
				t.Fatal("wrong chain used cached proof", err)
			}
		})
	}
}
func TestVersionUnknownOrIncompleteEvidenceRemainsUnverifiedAndIsNotCached(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(*types.Log, *versionFake)
	}{
		{"wrong chain", func(l *types.Log, n *versionFake) { n.chain = big.NewInt(1) }},
		{"null chain", func(l *types.Log, n *versionFake) { n.chain = nil }},
		{"header failure", func(l *types.Log, n *versionFake) { n.fail = "header" }},
		{"code denied", func(l *types.Log, n *versionFake) { n.fail = "code" }},
		{"slot denied", func(l *types.Log, n *versionFake) { n.fail = "slot" }},
		{"null upgrade response", func(l *types.Log, n *versionFake) { n.upgrades = nil }},
		{"upgrade query denied", func(l *types.Log, n *versionFake) { n.fail = "upgrades" }},
		{"nil header", func(l *types.Log, n *versionFake) { n.header = nil }},
		{"different header", func(l *types.Log, n *versionFake) {
			n.header = types.CopyHeader(n.header)
			n.header.Extra = []byte("other")
		}},
		{"unknown proxy", func(l *types.Log, n *versionFake) { n.code[l.Address] = []byte{0x00} }},
		{"parent proxy different", func(l *types.Log, n *versionFake) { n.parentCode[l.Address] = nil }},
		{"unknown implementation", func(l *types.Log, n *versionFake) {
			n.slot = common.LeftPadBytes(common.HexToAddress("0x456").Bytes(), 32)
		}},
		{"parent implementation different", func(l *types.Log, n *versionFake) {
			n.parentSlot = common.LeftPadBytes(common.HexToAddress("0x456").Bytes(), 32)
		}},
		{"short slot", func(l *types.Log, n *versionFake) { n.slot = n.slot[1:] }},
		{"slot high bytes", func(l *types.Log, n *versionFake) { n.slot[0] = 1 }},
		{"implementation code changed", func(l *types.Log, n *versionFake) {
			n.code[common.HexToAddress("0x641b40ec414a076b9e79e703fc7bf4ebec248bb7")] = []byte{1}
		}},
		{"same-block upgrade away and back", func(l *types.Log, n *versionFake) {
			n.upgrades = []types.Log{{Address: l.Address, BlockHash: l.BlockHash}}
		}},
		{"removed", func(l *types.Log, n *versionFake) { l.Removed = true }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			l, n := versionFixture(t, ComboExchangeVersion)
			tt.change(&l, n)
			v := NewVersionVerifier(n)
			got, err := v.Verify(context.Background(), l)
			if got != "" || status.Code(err) != codes.FailedPrecondition {
				t.Fatal(got, err)
			}
			// Restore the same node object to test a retry, not a new verifier/cache.
			goodLog, good := versionFixture(t, ComboExchangeVersion)
			*n = *good
			got, err = v.Verify(context.Background(), goodLog)
			if err != nil || got != ComboExchangeVersion {
				t.Fatal("failure cached", got, err)
			}
		})
	}
}
func TestVersionCacheSeparatesExchangeAndBlockHash(t *testing.T) {
	l, n := versionFixture(t, CoreExchangeVersion)
	v := NewVersionVerifier(n)
	if _, err := v.Verify(context.Background(), l); err != nil {
		t.Fatal(err)
	}
	other, _ := versionFixture(t, NegRiskExchangeVersion)
	if _, err := v.Verify(context.Background(), other); err == nil {
		t.Fatal("exchange cache reused")
	}
	n.header = types.CopyHeader(n.header)
	n.header.Extra = []byte("new hash")
	l.BlockHash = n.header.Hash()
	n.fail = "code"
	if _, err := v.Verify(context.Background(), l); err == nil {
		t.Fatal("block cache reused")
	}
}

func TestVersionCachedProofStillChecksLogHeight(t *testing.T) {
	l, n := versionFixture(t, ComboExchangeVersion)
	v := NewVersionVerifier(n)
	if _, err := v.Verify(context.Background(), l); err != nil {
		t.Fatal(err)
	}
	l.BlockNumber++
	if _, err := v.Verify(context.Background(), l); status.Code(err) != codes.FailedPrecondition {
		t.Fatal("cached proof accepted wrong height", err)
	}
}
