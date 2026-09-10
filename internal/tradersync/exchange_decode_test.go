package tradersync

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type sourceFixture struct {
	Name, Role, Version string
	Log                 types.Log
	Expected            struct{ Wallet, Side, PositionID, CollateralRaw, SharesRaw, FeeRaw string }
}

func sourceFixtures(t *testing.T) []sourceFixture {
	t.Helper()
	b, err := os.ReadFile("testdata/source_records.json")
	if err != nil {
		t.Fatal(err)
	}
	var items []sourceFixture
	if err = json.Unmarshal(b, &items); err != nil {
		t.Fatal(err)
	}
	return items
}

func TestDecodeFixedSourceMatrix(t *testing.T) {
	for _, f := range sourceFixtures(t) {
		t.Run(f.Name, func(t *testing.T) {
			got, err := DecodeOwnTrade(f.Log, f.Version)
			if err != nil {
				t.Fatal(err)
			}
			if got.Wallet != common.HexToAddress(f.Expected.Wallet) || got.Exchange != f.Log.Address || got.Side != f.Expected.Side || got.PositionID != f.Expected.PositionID || got.CollateralRaw != f.Expected.CollateralRaw || got.SharesRaw != f.Expected.SharesRaw || got.FeeRaw != f.Expected.FeeRaw {
				t.Fatalf("decoded %#v; want %+v", got, f.Expected)
			}
			if got.CollateralSymbol != "pUSD" || got.CollateralDecimals != 6 || got.SharesDecimals != 6 || got.SourceVersion != f.Version {
				t.Fatalf("wrong version units: %#v", got)
			}
			if got.PriceNumerator != f.Expected.CollateralRaw || got.PriceDenominator != f.Expected.SharesRaw {
				t.Fatalf("fee changed price ratio: %#v", got)
			}
		})
	}
}
func TestDecodeCounterpartyOnlyDoesNotAttributeObservedWallet(t *testing.T) {
	f := sourceFixtures(t)[0]
	observed := common.BytesToAddress(f.Log.Topics[3].Bytes()[12:])
	got, err := DecodeOwnTrade(f.Log, f.Version)
	if err != nil {
		t.Fatal(err)
	}
	if got.Wallet == observed || got.Wallet != common.HexToAddress(f.Expected.Wallet) {
		t.Fatalf("counterparty misattributed: %#v", got)
	}
}
func TestDecodeSameTransactionPreservesBothOwnLogs(t *testing.T) {
	f := sourceFixtures(t)[0]
	second := f.Log
	second.Index++
	second.Data = append([]byte(nil), f.Log.Data...)
	second.Data[63] ^= 1
	a, err := DecodeOwnTrade(f.Log, f.Version)
	if err != nil {
		t.Fatal(err)
	}
	b, err := DecodeOwnTrade(second, f.Version)
	if err != nil {
		t.Fatal(err)
	}
	if a.Wallet != b.Wallet || a.PositionID == b.PositionID {
		t.Fatal("distinct fills collapsed")
	}
}
func TestDecodeZeroSharesHasNoInventedPrice(t *testing.T) {
	f := sourceFixtures(t)[0]
	f.Log.Data = append([]byte(nil), f.Log.Data...)
	// This fixture is BUY: takerAmount word is shares.
	clear(f.Log.Data[96:128])
	got, err := DecodeOwnTrade(f.Log, f.Version)
	if err != nil {
		t.Fatal(err)
	}
	if got.SharesRaw != "0" || got.PriceNumerator != "" || got.PriceDenominator != "" {
		t.Fatalf("zero-share price was fabricated: %#v", got)
	}
	clear(f.Log.Data[64:96])
	got, err = DecodeOwnTrade(f.Log, f.Version)
	if err != nil || got.CollateralRaw != "0" {
		t.Fatal("zero value filtered", got, err)
	}
}
func TestDecodeRejectsWrongVersionAddressTopicAndMalformedABI(t *testing.T) {
	for _, mutate := range []struct {
		name string
		fn   func(*types.Log)
	}{
		{"unknown exchange", func(l *types.Log) { l.Address = common.HexToAddress("0x123") }},
		{"OrdersMatched", func(l *types.Log) {
			l.Topics[0] = crypto.Keccak256Hash([]byte("OrdersMatched(bytes32,address,uint8,uint256,uint256,uint256)"))
		}},
		// Fixed Binary/Combinatorial module ABI signatures; no module enters the trade registry.
		{"PositionsSplit", func(l *types.Log) {
			l.Topics[0] = crypto.Keccak256Hash([]byte("PositionsSplit(address,bytes31,address,address,uint256)"))
		}},
		{"PositionsMerged", func(l *types.Log) {
			l.Topics[0] = crypto.Keccak256Hash([]byte("PositionsMerged(address,bytes31,address,uint256)"))
		}},
		{"missing indexed field", func(l *types.Log) { l.Topics = l.Topics[:3] }},
		{"extra indexed field", func(l *types.Log) { l.Topics = append(l.Topics, common.Hash{}) }},
		{"short data", func(l *types.Log) { l.Data = l.Data[:223] }},
		{"extra data", func(l *types.Log) { l.Data = append(l.Data, 0) }},
		{"invalid side", func(l *types.Log) { l.Data[31] = 2 }},
		{"uint8 padding", func(l *types.Log) { l.Data[0] = 1 }},
		{"address padding", func(l *types.Log) { l.Topics[2][0] = 1 }},
		{"removed", func(l *types.Log) { l.Removed = true }},
	} {
		t.Run(mutate.name, func(t *testing.T) {
			f := sourceFixtures(t)[0]
			mutate.fn(&f.Log)
			if _, err := DecodeOwnTrade(f.Log, f.Version); err == nil {
				t.Fatal("accepted invalid source")
			}
		})
	}
	f := sourceFixtures(t)[0]
	if _, err := DecodeOwnTrade(f.Log, "polygon137-combo-641b40ec414a"); err == nil {
		t.Fatal("version valid for a different address")
	}
}
func TestUnknownExecutionVersionIsNotDecoded(t *testing.T) {
	if _, err := DecodeOwnTrade(types.Log{}, "unrecognized-implementation"); err == nil {
		t.Fatal("decoded unknown execution version")
	}
}
func TestDecodeUint256IsExact(t *testing.T) {
	f := sourceFixtures(t)[0]
	f.Log.Data = append([]byte(nil), f.Log.Data...)
	for i := 32; i < 64; i++ {
		f.Log.Data[i] = 255
	}
	got, err := DecodeOwnTrade(f.Log, f.Version)
	want := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)).String()
	if err != nil || got.PositionID != want {
		t.Fatal(got, err)
	}
}
