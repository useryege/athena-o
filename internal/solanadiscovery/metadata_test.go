package solanadiscovery

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"github.com/gagliardetto/solana-go"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func metadataString(s string) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(len(s)))
	return append(b, []byte(s)...)
}
func tokenMetadataFixture(mint string) *AccountInfo {
	b := make([]byte, 166)
	b[45] = 1
	b[165] = 1
	key := solana.MustPublicKeyFromBase58(mint)
	b = append(b, 18, 0, 64, 0)
	b = append(b, make([]byte, 32)...)
	b = append(b, key.Bytes()...)
	payload := append(make([]byte, 32), key.Bytes()...)
	for _, s := range []string{"名称", "SYM", ""} {
		payload = append(payload, metadataString(s)...)
	}
	payload = append(payload, 0, 0, 0, 0)
	h := []byte{19, 0, 0, 0}
	binary.LittleEndian.PutUint16(h[2:], uint16(len(payload)))
	b = append(b, h...)
	b = append(b, payload...)
	return &AccountInfo{Owner: Token2022Program, Data: b}
}
func usdcMetadataFixture(t *testing.T) (*AccountInfo, *AccountInfo) {
	t.Helper()
	b, err := os.ReadFile("testdata/usdc-metadata.json")
	require.NoError(t, err)
	var doc struct {
		Response struct {
			Result struct {
				Value []struct {
					Owner string   `json:"owner"`
					Data  []string `json:"data"`
				} `json:"value"`
			} `json:"result"`
		} `json:"response"`
	}
	require.NoError(t, json.Unmarshal(b, &doc))
	out := make([]*AccountInfo, 2)
	for i, a := range doc.Response.Result.Value {
		data, err := base64.StdEncoding.DecodeString(a.Data[0])
		require.NoError(t, err)
		out[i] = &AccountInfo{Owner: a.Owner, Data: data}
	}
	return out[0], out[1]
}

// Removing owner/mint/length/UTF8 checks or reading the wrong TLV container must fail.
func TestMetadataOnMintValidation(t *testing.T) {
	for _, tt := range []struct {
		name    string
		mutate  func(*AccountInfo)
		wantErr bool
		status  string
	}{
		{"valid", func(*AccountInfo) {}, false, "ready"},
		{"owner", func(a *AccountInfo) { a.Owner = TokenProgram }, true, ""},
		{"mint mismatch", func(a *AccountInfo) { a.Data[234+4+32] ^= 1 }, true, ""},
		{"length", func(a *AccountInfo) { a.Data[236] = 255; a.Data[237] = 255 }, true, ""},
		{"embedded NUL", func(a *AccountInfo) { a.Data[316] = 0 }, true, ""},
		{"UTF8", func(a *AccountInfo) { a.Data[234+4+64+4] = 255 }, true, ""},
		{"external pointer", func(a *AccountInfo) { copy(a.Data[202:234], solana.MustPublicKeyFromBase58(wrappedSolMint).Bytes()) }, false, "unavailable"},
		{"padding", func(a *AccountInfo) { a.Data[100] = 1 }, true, ""},
		{"account type", func(a *AccountInfo) { a.Data[165] = 2 }, true, ""},
		{"uninitialized", func(a *AccountInfo) { a.Data[45] = 0 }, true, ""},
		{"truncated tail", func(a *AccountInfo) { a.Data = a.Data[:len(a.Data)-2] }, true, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a := tokenMetadataFixture(usdcMint)
			tt.mutate(a)
			got, err := ReadMetadata(Project{Mint: usdcMint, TokenProgram: Token2022Program}, a, nil)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.status, got.Status)
			if got.Status == "ready" {
				require.Equal(t, "名称", got.Name)
				require.Equal(t, "SYM", got.Symbol)
				require.Equal(t, "token2022_on_mint", got.Source)
				require.Equal(t, usdcMint, got.Account)
			}
		})
	}
}
func TestMetadataMetaplexRealPaddingAndIdentity(t *testing.T) {
	pda, err := MetadataAddress(usdcMint)
	require.NoError(t, err)
	require.Equal(t, "5x38Kp4hvdomTCnCrAny4UtMUt5rQBdB6px2K1Ui45Wq", pda)
	for _, tt := range []struct {
		name   string
		mutate func(*AccountInfo)
		bad    bool
	}{
		{"valid", func(*AccountInfo) {}, false},
		{"owner", func(a *AccountInfo) { a.Owner = TokenProgram }, true},
		{"mint", func(a *AccountInfo) { a.Data[33] ^= 1 }, true},
		{"key", func(a *AccountInfo) { a.Data[0] = 1 }, true},
		{"length", func(a *AccountInfo) { a.Data[65] = 255 }, true},
		{"embedded NUL", func(a *AccountInfo) { a.Data[70] = 0 }, true},
		{"utf8", func(a *AccountInfo) { a.Data[69] = 255 }, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mint, a := usdcMetadataFixture(t)
			tt.mutate(a)
			got, err := ReadMetadata(Project{Mint: usdcMint, TokenProgram: TokenProgram}, mint, a)
			if tt.bad {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "USD Coin", got.Name)
			require.Equal(t, "USDC", got.Symbol)
			require.Equal(t, "ready", got.Status)
			require.Equal(t, "metaplex", got.Source)
			require.Equal(t, pda, got.Account)
		})
	}
}
func TestMetadataNullAccountUnavailable(t *testing.T) {
	got, err := ReadMetadata(Project{Mint: usdcMint, TokenProgram: TokenProgram}, nil, nil)
	require.NoError(t, err)
	require.Equal(t, "unavailable", got.Status)
}

func TestMetadataWhitespaceIsAbsentAndCanonicalPointerResolves(t *testing.T) {
	mint, meta := usdcMetadataFixture(t)
	for i := 69; i < 101; i++ {
		meta.Data[i] = ' '
	}
	got, err := ReadMetadata(Project{Mint: usdcMint, TokenProgram: TokenProgram}, mint, meta)
	require.NoError(t, err)
	require.Empty(t, got.Name)
	require.Equal(t, "USDC", got.Symbol)
	require.Equal(t, "ready", got.Status)
	token := tokenMetadataFixture(usdcMint)
	token.Data = token.Data[:234]
	copy(token.Data[202:234], solana.MustPublicKeyFromBase58("5x38Kp4hvdomTCnCrAny4UtMUt5rQBdB6px2K1Ui45Wq").Bytes())
	got, err = ReadMetadata(Project{Mint: usdcMint, TokenProgram: Token2022Program}, token, meta)
	require.NoError(t, err)
	require.Equal(t, "metaplex", got.Source)
	require.Equal(t, "USDC", got.Symbol)
}
