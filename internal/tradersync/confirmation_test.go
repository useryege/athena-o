package tradersync

import (
	"context"
	"errors"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type canonicalFake struct {
	calls                   []string
	receiptHash, knownHash  common.Hash
	height                  *big.Int
	final, canonical, known *types.Header
	receipt                 *types.Receipt
	fail                    string
}

func (f *canonicalFake) record(name string) error {
	f.calls = append(f.calls, name)
	if f.fail == name {
		return errors.New("403 provider denied")
	}
	return nil
}
func (f *canonicalFake) FinalizedHeader(context.Context) (*types.Header, error) {
	return f.final, f.record("finalized")
}
func (f *canonicalFake) TransactionReceipt(_ context.Context, hash common.Hash) (*types.Receipt, error) {
	f.receiptHash = hash
	return f.receipt, f.record("receipt")
}
func (f *canonicalFake) HeaderByHash(_ context.Context, hash common.Hash) (*types.Header, error) {
	f.knownHash = hash
	return f.known, f.record("known")
}
func (f *canonicalFake) HeaderByNumber(_ context.Context, n *big.Int) (*types.Header, error) {
	f.height = new(big.Int).Set(n)
	return f.canonical, f.record("canonical")
}
func canonicalFixture(t *testing.T) (types.Log, *canonicalFake) {
	t.Helper()
	l := sourceFixtures(t)[0].Log
	h := &types.Header{Number: big.NewInt(100), Time: 1700000000, GasLimit: 30000000, Difficulty: big.NewInt(0)}
	l.BlockNumber = 100
	l.BlockHash = h.Hash()
	l.TxIndex = 2
	other := l
	other.Index++
	other.Address = common.HexToAddress("0x123")
	r := &types.Receipt{Status: types.ReceiptStatusSuccessful, TxHash: l.TxHash, BlockHash: l.BlockHash, BlockNumber: big.NewInt(100), TransactionIndex: l.TxIndex, Logs: []*types.Log{&other, &l}}
	return l, &canonicalFake{final: &types.Header{Number: big.NewInt(110)}, canonical: h, known: h, receipt: r}
}
func TestConfirmReceivedOrdersFreshFinalityThenExactKnownEvidence(t *testing.T) {
	l, n := canonicalFixture(t)
	before := time.Now()
	got, err := ConfirmReceived(context.Background(), n, l)
	if err != nil || got.Status != "confirmed" || got.BlockHash != l.BlockHash || !got.SettledAt.Equal(time.Unix(1700000000, 0).UTC()) || got.CheckedAt.Before(before) {
		t.Fatal(got, err)
	}
	if n.receiptHash != l.TxHash || n.knownHash != l.BlockHash || n.height.Uint64() != l.BlockNumber {
		t.Fatal("wrong candidate evidence requested", n.receiptHash, n.knownHash, n.height)
	}
	if !reflect.DeepEqual(n.calls, []string{"finalized", "receipt", "canonical", "known"}) {
		t.Fatal(n.calls)
	}
}
func TestConfirmUnavailableNeverTurnsIntoInvalidOrConfirmed(t *testing.T) {
	for _, step := range []string{"finalized", "receipt", "canonical", "known"} {
		t.Run(step, func(t *testing.T) {
			l, n := canonicalFixture(t)
			n.fail = step
			got, err := ConfirmReceived(context.Background(), n, l)
			if got.Status != "unverified" || got.Reason == "" || err == nil || !got.SettledAt.IsZero() {
				t.Fatal(got, err)
			}
		})
	}
	for _, step := range []string{"finalized", "receipt", "canonical", "known"} {
		t.Run("null-"+step, func(t *testing.T) {
			l, n := canonicalFixture(t)
			switch step {
			case "finalized":
				n.final = nil
			case "receipt":
				n.receipt = nil
			case "canonical":
				n.canonical = nil
			case "known":
				n.known = nil
			}
			got, _ := ConfirmReceived(context.Background(), n, l)
			if got.Status != "unverified" {
				t.Fatal(got)
			}
		})
	}
	l, n := canonicalFixture(t)
	n.final.Number = big.NewInt(99)
	got, err := ConfirmReceived(context.Background(), n, l)
	if err != nil || got.Status != "unverified" || len(n.calls) != 1 {
		t.Fatal(got, n.calls, err)
	}
}
func TestConfirmRejectsRemovedChangedCanonicalOrReceiptEvidence(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(*types.Log, *canonicalFake)
	}{
		{"removed", func(l *types.Log, n *canonicalFake) { l.Removed = true }},
		{"deep reorg", func(l *types.Log, n *canonicalFake) {
			h := types.CopyHeader(n.canonical)
			h.Extra = []byte("new canonical block")
			n.canonical = h
		}},
		{"receipt block", func(l *types.Log, n *canonicalFake) { n.receipt.BlockHash = common.HexToHash("0x1") }},
		{"receipt tx", func(l *types.Log, n *canonicalFake) { n.receipt.TxHash = common.HexToHash("0x2") }},
		{"receipt height", func(l *types.Log, n *canonicalFake) { n.receipt.BlockNumber = big.NewInt(101) }},
		{"receipt position", func(l *types.Log, n *canonicalFake) { n.receipt.TransactionIndex++ }},
		{"failed receipt", func(l *types.Log, n *canonicalFake) { n.receipt.Status = 0 }},
		{"missing log", func(l *types.Log, n *canonicalFake) { n.receipt.Logs = n.receipt.Logs[:1] }},
		{"changed bytes", func(l *types.Log, n *canonicalFake) {
			n.receipt.Logs[1].Data = append([]byte(nil), l.Data...)
			n.receipt.Logs[1].Data[0] ^= 1
		}},
		{"changed topic", func(l *types.Log, n *canonicalFake) {
			n.receipt.Logs[1].Topics = append([]common.Hash(nil), l.Topics...)
			n.receipt.Logs[1].Topics[2] = common.HexToHash("0x7")
		}},
		{"changed log address", func(l *types.Log, n *canonicalFake) { n.receipt.Logs[1].Address = common.HexToAddress("0x1") }},
		{"removed receipt log", func(l *types.Log, n *canonicalFake) { n.receipt.Logs[1].Removed = true }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			l, n := canonicalFixture(t)
			tt.change(&l, n)
			got, err := ConfirmReceived(context.Background(), n, l)
			if err != nil || got.Status != "invalid" || got.Reason == "" || !got.SettledAt.IsZero() {
				t.Fatal(got, err)
			}
			for _, call := range n.calls {
				if call == "known" {
					t.Fatal("timestamp read before evidence accepted")
				}
			}
		})
	}
}
func TestConfirmRechecksPreviouslyConfirmedHashAfterDeepReorg(t *testing.T) {
	l, n := canonicalFixture(t)
	got, _ := ConfirmReceived(context.Background(), n, l)
	if got.Status != "confirmed" {
		t.Fatal(got)
	}
	n.canonical = types.CopyHeader(n.canonical)
	n.canonical.Extra = []byte("replacement")
	got, _ = ConfirmReceived(context.Background(), n, l)
	if got.Status != "invalid" {
		t.Fatal("positive confirmation was cached across reorg", got)
	}
}
