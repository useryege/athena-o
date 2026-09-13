package solanadiscovery

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/gagliardetto/solana-go/base58"
	"github.com/stretchr/testify/require"
)

const legacyTokenProgram = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
const token2022Program = "TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb"
const usdcMint = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
const wrappedSolMint = "So11111111111111111111111111111111111111112"

// Removing top-level or CPI handling, accepting a failed transaction, or trusting
// an instruction's type without its program identity must break this test.
func TestParseBlockFindsSuccessfulTopLevelAndCPIInitializations(t *testing.T) {
	fixture := `{
      "blockTime": 1720000000,
      "transactions": [
        {"transaction":{"signatures":["sig-top"],"message":{"accountKeys":[{"pubkey":"11111111111111111111111111111111","signer":true,"writable":true}],"instructions":[
          {"program":"spl-token","programId":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","parsed":{"type":"initializeMint","info":{"mint":"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v","decimals":6,"mintAuthority":"11111111111111111111111111111111","freezeAuthority":null}}}
        ]}},"meta":{"err":null,"innerInstructions":[]}},
        {"transaction":{"signatures":["sig-cpi"],"message":{"accountKeys":[{"pubkey":"So11111111111111111111111111111111111111112","signer":true,"writable":true,"source":"transaction"},{"pubkey":"TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb","signer":false,"writable":false,"source":"lookupTable"}],"instructions":[{"programId":"11111111111111111111111111111111","parsed":{"type":"initializeMint2","info":{"mint":"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v","decimals":9,"mintAuthority":"11111111111111111111111111111111"}}}]}},"meta":{"err":null,"innerInstructions":[{"index":0,"instructions":[
          {"program":"spl-token-2022","programId":"TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb","parsed":{"type":"initializeMint2","info":{"mint":"So11111111111111111111111111111111111111112","decimals":9,"mintAuthority":"11111111111111111111111111111111","freezeAuthority":"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"}}}
        ]}]}},
        {"transaction":{"signatures":["sig-failed"],"message":{"accountKeys":[{"pubkey":"11111111111111111111111111111111","signer":true}],"instructions":[{"programId":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","parsed":{"type":"initializeMint","info":{"mint":"So11111111111111111111111111111111111111112","decimals":0,"mintAuthority":"11111111111111111111111111111111"}}}]}},"meta":{"err":{"InstructionError":[0,"Custom"]}}},
        {"transaction":{"signatures":["sig-repeat"],"message":{"accountKeys":[{"pubkey":"11111111111111111111111111111111","signer":true}],"instructions":[{"programId":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","parsed":{"type":"initializeMint2","info":{"mint":"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v","decimals":6,"mintAuthority":"11111111111111111111111111111111"}}}]}},"meta":{"err":null}}
      ]
    }`
	got, err := ParseBlock(42, []byte(fixture))
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, usdcMint, got[0].Mint)
	require.Equal(t, legacyTokenProgram, got[0].TokenProgram)
	require.Equal(t, "sig-top", got[0].Signature)
	require.Equal(t, "11111111111111111111111111111111", got[0].FeePayer)
	require.Equal(t, uint32(6), got[0].Decimals)
	require.Equal(t, "", got[0].FreezeAuthority)
	require.Equal(t, uint64(42), got[0].Slot)
	require.Equal(t, int64(1720000000), got[0].BlockTime)
	require.Equal(t, wrappedSolMint, got[1].Mint)
	require.Equal(t, token2022Program, got[1].TokenProgram)
	require.Equal(t, "sig-cpi", got[1].Signature)
	require.Equal(t, wrappedSolMint, got[1].FeePayer)
	require.Equal(t, usdcMint, got[1].FreezeAuthority)
}

// A recognized but incomplete initialization must stop the range instead of
// silently losing a candidate and advancing the checkpoint.
func TestParseBlockRejectsBrokenInitialization(t *testing.T) {
	fixture := `{"transactions":[{"transaction":{"signatures":["sig"],"message":{"accountKeys":[{"pubkey":"11111111111111111111111111111111","signer":true}],"instructions":[{"programId":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","parsed":{"type":"initializeMint","info":{"mint":"bad","decimals":6}}}]}},"meta":{"err":null}}]}`
	_, err := ParseBlock(42, []byte(fixture))
	require.Error(t, err)
}

func TestParseBlockRejectsRecognizedInitializationWithWrongFieldType(t *testing.T) {
	fixture := `{"transactions":[{"transaction":{"signatures":["sig"],"message":{"accountKeys":[{"pubkey":"11111111111111111111111111111111","signer":true}],"instructions":[{"programId":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","parsed":{"type":"initializeMint2","info":{"mint":"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v","decimals":"six","mintAuthority":"11111111111111111111111111111111"}}}]}},"meta":{"err":null}}]}`
	_, err := ParseBlock(42, []byte(fixture))
	require.Error(t, err)
}

func TestParseBlockRequiresExecutionStatus(t *testing.T) {
	fixture := `{"transactions":[{"transaction":{"signatures":["sig"],"message":{"accountKeys":[{"pubkey":"11111111111111111111111111111111","signer":true}],"instructions":[{"programId":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","parsed":{"type":"initializeMint","info":{"mint":"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v","decimals":6,"mintAuthority":"11111111111111111111111111111111"}}}]}},"meta":{}}]}`
	_, err := ParseBlock(42, []byte(fixture))
	require.Error(t, err)
}

func TestParseBlockRejectsMissingTransactionsField(t *testing.T) {
	_, err := ParseBlock(42, []byte(`{"blockTime":1720000000}`))
	require.Error(t, err)
}

func TestParseBlockDecodesRawTokenInitializationsAndSkipsUnknownOpcode(t *testing.T) {
	data := append([]byte{0, 6}, make([]byte, 33)...)
	raw := base58.Encode(data)
	fixture := fmt.Sprintf(`{"transactions":[{"transaction":{"signatures":["sig-raw"],"message":{"accountKeys":[{"pubkey":"11111111111111111111111111111111","signer":true}],"instructions":[{"programId":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","accounts":["%s"],"data":"%s"},{"programId":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","accounts":["%s"],"data":"%s"}]}},"meta":{"err":null}}]}`, usdcMint, base58.Encode([]byte{99}), usdcMint, raw)
	got, err := ParseBlock(42, []byte(fixture))
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, usdcMint, got[0].Mint)
	require.Equal(t, "11111111111111111111111111111111", got[0].MintAuthority)
	require.Equal(t, uint32(6), got[0].Decimals)
}

func TestParseBlockRejectsTokenInstructionWithNeitherParsedNorRawData(t *testing.T) {
	fixture := `{"transactions":[{"transaction":{"signatures":["sig"],"message":{"accountKeys":[{"pubkey":"11111111111111111111111111111111","signer":true}],"instructions":[{"programId":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","accounts":["EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"]}]}},"meta":{"err":null}}]}`
	_, err := ParseBlock(42, []byte(fixture))
	require.Error(t, err)
}

// SPL instruction options occupy one byte, unlike account-state COptions.
func TestParseBlockDecodesRawInitializeMint2WithFreezeAuthority(t *testing.T) {
	authority, err := base58.Decode(wrappedSolMint)
	require.NoError(t, err)
	freeze, err := base58.Decode(usdcMint)
	require.NoError(t, err)
	data := append([]byte{20, 9}, authority...)
	data = append(data, 1)
	data = append(data, freeze...)
	fixture := fmt.Sprintf(`{"transactions":[{"transaction":{"signatures":["sig"],"message":{"accountKeys":[{"pubkey":"11111111111111111111111111111111"}],"instructions":[{"programId":"11111111111111111111111111111111"}]}},"meta":{"err":null,"innerInstructions":[{"index":0,"instructions":[{"programId":"TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb","accounts":["%s"],"data":"%s"}]}]}}]}`, usdcMint, base58.Encode(data))
	got, err := ParseBlock(42, []byte(fixture))
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, usdcMint, got[0].Mint)
	require.Equal(t, wrappedSolMint, got[0].MintAuthority)
	require.Equal(t, usdcMint, got[0].FreezeAuthority)
	require.Equal(t, uint32(9), got[0].Decimals)
}

func TestParseBlockRejectsMalformedRawInitialization(t *testing.T) {
	for _, data := range [][]byte{{0}, {20, 6}, append(append([]byte{20, 6}, make([]byte, 32)...), 2)} {
		fixture := fmt.Sprintf(`{"transactions":[{"transaction":{"signatures":["sig"],"message":{"accountKeys":[{"pubkey":"11111111111111111111111111111111"}],"instructions":[{"programId":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","accounts":["%s"],"data":"%s"}]}},"meta":{"err":null}}]}`, usdcMint, base58.Encode(data))
		_, err := ParseBlock(42, []byte(fixture))
		require.Error(t, err)
	}
}

func TestParseBlockRejectsIncompleteSuccessfulTransaction(t *testing.T) {
	for _, instructions := range []string{"", `,"instructions":null`} {
		fixture := fmt.Sprintf(`{"transactions":[{"transaction":{"signatures":["sig"],"message":{"accountKeys":[{"pubkey":"11111111111111111111111111111111"}]%s}},"meta":{"err":null}}]}`, instructions)
		_, err := ParseBlock(42, []byte(fixture))
		require.Error(t, err)
	}
}

func TestParseBlockRejectsOutOfRangeDecimals(t *testing.T) {
	_, err := ParseBlock(42, []byte(strings.Replace(scannerBlock, `"decimals":6`, `"decimals":256`, 1)))
	require.Error(t, err)
}

func TestParseBlockAcceptsRawInitializationTrailingBytes(t *testing.T) {
	for _, option := range []byte{0, 1} {
		data := append([]byte{20, 6}, make([]byte, 32)...)
		data = append(data, option)
		if option == 1 {
			data = append(data, make([]byte, 32)...)
		}
		data = append(data, 77, 88)
		fixture := fmt.Sprintf(`{"transactions":[{"transaction":{"signatures":["sig"],"message":{"accountKeys":[{"pubkey":"11111111111111111111111111111111"}],"instructions":[{"programId":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","accounts":["%s"],"data":"%s"}]}},"meta":{"err":null}}]}`, usdcMint, base58.Encode(data))
		got, err := ParseBlock(42, []byte(fixture))
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, usdcMint, got[0].Mint)
	}
}

// A CPI runs before the following top-level instruction; duplicate evidence must
// retain the first initialization in execution order, not top-level-first order.
func TestParseBlockPreservesExecutionOrderForDuplicateMint(t *testing.T) {
	var block map[string]any
	require.NoError(t, json.Unmarshal([]byte(scannerBlock), &block))
	tx := block["transactions"].([]any)[0].(map[string]any)
	message := tx["transaction"].(map[string]any)["message"].(map[string]any)
	later := message["instructions"].([]any)[0]
	message["instructions"] = []any{map[string]any{"programId": "11111111111111111111111111111111"}, later}
	inner := map[string]any{"programId": TokenProgram, "parsed": map[string]any{"type": "initializeMint2", "info": map[string]any{"mint": usdcMint, "mintAuthority": wrappedSolMint, "decimals": 9}}}
	tx["meta"].(map[string]any)["innerInstructions"] = []any{map[string]any{"index": 0, "instructions": []any{inner}}}
	data, err := json.Marshal(block)
	require.NoError(t, err)
	got, err := ParseBlock(42, data)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, wrappedSolMint, got[0].MintAuthority)
	require.Equal(t, uint32(9), got[0].Decimals)
}

func TestParseBlockRejectsInnerInstructionsOutsideMessage(t *testing.T) {
	fixture := strings.Replace(scannerBlock, `"meta":{"err":null}`, `"meta":{"err":null,"innerInstructions":[{"index":5,"instructions":[]}]}`, 1)
	_, err := ParseBlock(42, []byte(fixture))
	require.Error(t, err)
}
