package solanadiscovery

import (
	"encoding/json"
	"github.com/gagliardetto/solana-go/base58"
	"github.com/stretchr/testify/require"
	"os"
	"reflect"
	"testing"
)

// Missing public evidence fields must fail before API consumers can silently omit them.
func TestSourceProjectContract(t *testing.T) {
	typ := reflect.TypeOf(Project{})
	for _, name := range []string{"Name", "Symbol", "MetadataStatus", "MetadataSource", "MetadataAccount", "MetadataObservedSlot", "MetadataUpdatedAt", "IssuanceSource", "IssuanceProgram", "SourceStatus"} {
		_, ok := typ.FieldByName(name)
		require.True(t, ok, "Project is missing %s", name)
	}
}

// A transaction-wide tag would misidentify the nested unknown fixture's immediate parent.
func TestIssuanceRealTransactions(t *testing.T) {
	for _, tt := range []struct{ file, mint, source, program, status string }{
		{"source-0.json", "DTgysU4LwNfBR2RMXttgBaVF8ghF3vkePJV5Fgg4pump", "pump_fun", "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P", "identified"},
		{"source-1.json", "5RoQkobzTxjzAHzwu3T811kar1seHcofrXFrd3bPDkYi", "unknown", "cpamdpZCGKUy5JxQXB4dcpGPiikHawvSWAd6mEn1sGG", "unrecognized"},
	} {
		t.Run(tt.file, func(t *testing.T) {
			data, err := os.ReadFile("testdata/" + tt.file)
			require.NoError(t, err)
			var tx map[string]any
			require.NoError(t, json.Unmarshal(data, &tx))
			block, err := json.Marshal(map[string]any{"blockTime": tx["blockTime"], "transactions": []any{tx}})
			require.NoError(t, err)
			got, err := ParseBlock(uint64(tx["slot"].(float64)), block)
			require.NoError(t, err)
			var found bool
			for _, p := range got {
				if p.Mint == tt.mint {
					found = true
					v := reflect.ValueOf(p)
					for name, want := range map[string]string{"IssuanceSource": tt.source, "IssuanceProgram": tt.program, "SourceStatus": tt.status} {
						f := v.FieldByName(name)
						require.True(t, f.IsValid(), "missing %s", name)
						require.Equal(t, want, f.String())
					}
				}
			}
			require.True(t, found)
		})
	}
}

func sourceInstruction(program string, selector []byte, accounts []string, height any) map[string]any {
	return map[string]any{"programId": program, "data": base58.Encode(selector), "accounts": accounts, "stackHeight": height}
}
func sourceInit(mint string, height any) map[string]any {
	return map[string]any{"programId": Token2022Program, "stackHeight": height, "parsed": map[string]any{"type": "initializeMint2", "info": map[string]any{"mint": mint, "decimals": 6, "mintAuthority": wrappedSolMint}}}
}
func sourceBlock(t *testing.T, outer map[string]any, inner []any) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{"transactions": []any{map[string]any{"transaction": map[string]any{"signatures": []string{"sig"}, "message": map[string]any{"accountKeys": []any{map[string]any{"pubkey": wrappedSolMint}}, "instructions": []any{outer}}}, "meta": map[string]any{"err": nil, "innerInstructions": []any{map[string]any{"index": 0, "instructions": inner}}}}}})
	require.NoError(t, err)
	return b
}

// Sibling calls and quote mints must never inherit another mint's launch label.
func TestIssuanceCallTreeAndMintAccount(t *testing.T) {
	const pump = "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P"
	const launch = "LanMV9sAd7wArD4vJFi2qDdfnVhFxYSUg6eADduJ3uj"
	create := []byte{214, 144, 76, 236, 95, 139, 49, 180}
	wrapper := sourceInstruction(wrappedSolMint, []byte{99}, nil, 1)
	for _, tt := range []struct {
		name          string
		outer         map[string]any
		inner         []any
		want, program string
	}{
		{"nested pump", wrapper, []any{sourceInstruction(pump, create, []string{usdcMint}, 2), sourceInit(usdcMint, 3)}, "pump_fun", pump},
		{"sibling", wrapper, []any{sourceInstruction(pump, create, []string{usdcMint}, 2), sourceInit(usdcMint, 2)}, "unknown", wrappedSolMint},
		{"jump", wrapper, []any{sourceInstruction(pump, create, []string{usdcMint}, 2), sourceInit(usdcMint, 4)}, "unknown", ""},
		{"missing height", wrapper, []any{sourceInstruction(pump, create, []string{usdcMint}, nil), sourceInit(usdcMint, nil)}, "unknown", ""},
		{"top known missing height", sourceInstruction(pump, create, []string{usdcMint}, 1), []any{sourceInit(usdcMint, nil)}, "pump_fun", pump},
		{"wrong mint", sourceInstruction(pump, create, []string{wrappedSolMint}, 1), []any{sourceInit(usdcMint, 2)}, "unknown", pump},
		{"direct", sourceInit(usdcMint, 1), nil, "direct_token", Token2022Program},
		{"quote", sourceInstruction(launch, []byte{175, 175, 109, 31, 13, 152, 155, 237}, []string{"a", "b", "c", "d", "e", "f", wrappedSolMint, usdcMint}, 1), []any{sourceInit(usdcMint, 2)}, "unknown", launch},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p, err := ParseBlock(42, sourceBlock(t, tt.outer, tt.inner))
			require.NoError(t, err)
			require.Len(t, p, 1)
			require.Equal(t, tt.want, p[0].IssuanceSource)
			require.Equal(t, tt.program, p[0].IssuanceProgram)
		})
	}
	for _, selector := range [][]byte{{175, 175, 109, 31, 13, 152, 155, 237}, {67, 153, 175, 39, 218, 16, 38, 32}, {37, 190, 126, 222, 44, 154, 171, 17}} {
		p, err := ParseBlock(42, sourceBlock(t, sourceInstruction(launch, selector, []string{"a", "b", "c", "d", "e", "f", usdcMint, wrappedSolMint}, 1), []any{sourceInit(usdcMint, 2)}))
		require.NoError(t, err)
		require.Equal(t, "raydium_launchlab", p[0].IssuanceSource)
	}
}
