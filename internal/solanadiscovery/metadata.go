package solanadiscovery

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/gagliardetto/solana-go"
)

const metaplexProgram = "metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s"

// AccountInfo is an account at a finalized bank; DecodeError is isolated per address.
type AccountInfo struct {
	Owner       string
	Executable  bool
	Data        []byte
	DecodeError error
}

type MetadataResult struct {
	Name, Symbol, Status, Source, Account string
	ObservedSlot                          uint64
}

func MetadataAddress(mint string) (string, error) {
	key, err := solana.PublicKeyFromBase58(mint)
	if err != nil {
		return "", err
	}
	program := solana.MustPublicKeyFromBase58(metaplexProgram)
	pda, _, err := solana.FindProgramAddress([][]byte{[]byte("metadata"), program.Bytes(), key.Bytes()}, program)
	return pda.String(), err
}

func validateAccount(a *AccountInfo, owner string) error {
	if a.DecodeError != nil {
		return a.DecodeError
	}
	if a.Owner != owner || a.Executable {
		return errors.New("metadata account owner or executable flag mismatch")
	}
	return nil
}

// ReadMetadata reads issuer-provided on-chain text only, never a URI resource.
// The second account must have been requested at MetadataAddress(p.Mint).
func ReadMetadata(p Project, mintAccount, metadataAccount *AccountInfo) (MetadataResult, error) {
	unavailable := MetadataResult{Status: "unavailable"}
	mint, err := solana.PublicKeyFromBase58(p.Mint)
	if err != nil {
		return MetadataResult{}, errors.New("invalid metadata mint")
	}
	if mintAccount == nil {
		return unavailable, nil
	}
	if p.TokenProgram != TokenProgram && p.TokenProgram != Token2022Program {
		return MetadataResult{}, errors.New("unsupported token program")
	}
	if err := validateAccount(mintAccount, p.TokenProgram); err != nil {
		return MetadataResult{}, err
	}
	b := mintAccount.Data
	if len(b) < 82 || len(b) == 355 || b[45] != 1 || binary.LittleEndian.Uint32(b[:4]) > 1 || binary.LittleEndian.Uint32(b[46:50]) > 1 {
		return MetadataResult{}, errors.New("invalid initialized mint account")
	}
	pda, err := MetadataAddress(p.Mint)
	if err != nil {
		return MetadataResult{}, err
	}
	if p.TokenProgram == TokenProgram && len(b) != 82 {
		return MetadataResult{}, errors.New("invalid legacy mint size")
	}
	if p.TokenProgram == Token2022Program && len(b) > 82 {
		if len(b) < 166 || b[165] != 1 || !allZero(b[82:165]) {
			return MetadataResult{}, errors.New("invalid mint extension header")
		}
		var pointer, metadata []byte
		for pos := 166; pos+1 < len(b); {
			typ := binary.LittleEndian.Uint16(b[pos : pos+2])
			if typ == 0 {
				break
			}
			if len(b)-pos < 4 {
				return MetadataResult{}, errors.New("truncated mint TLV header")
			}
			n := int(binary.LittleEndian.Uint16(b[pos+2 : pos+4]))
			pos += 4
			if n > len(b)-pos {
				return MetadataResult{}, errors.New("truncated mint TLV value")
			}
			switch typ {
			case 18:
				if pointer != nil || n != 64 {
					return MetadataResult{}, errors.New("invalid metadata pointer")
				}
				pointer = b[pos : pos+n]
			case 19:
				if metadata != nil {
					return MetadataResult{}, errors.New("duplicate token metadata")
				}
				metadata = b[pos : pos+n]
			}
			pos += n
		}
		if pointer != nil {
			switch {
			case bytes.Equal(pointer[32:], mint.Bytes()):
				if metadata == nil {
					return unavailable, nil
				}
				if len(metadata) < 64 || !bytes.Equal(metadata[32:64], mint.Bytes()) {
					return MetadataResult{}, errors.New("token metadata mint mismatch")
				}
				c := metadataCursor{data: metadata[64:]}
				name, err := c.text(0)
				if err != nil {
					return MetadataResult{}, err
				}
				symbol, err := c.text(0)
				if err != nil {
					return MetadataResult{}, err
				}
				if _, err = c.text(0); err != nil {
					return MetadataResult{}, err
				}
				count, err := c.u32()
				if err != nil {
					return MetadataResult{}, err
				}
				if uint64(count) > uint64(len(c.data))/8 {
					return MetadataResult{}, errors.New("invalid additional metadata count")
				}
				for i := uint32(0); i < count; i++ {
					if _, err = c.text(0); err != nil {
						return MetadataResult{}, err
					}
					if _, err = c.text(0); err != nil {
						return MetadataResult{}, err
					}
				}
				return textMetadata(name, symbol, "token2022_on_mint", p.Mint)
			case bytes.Equal(pointer[32:], solana.MustPublicKeyFromBase58(pda).Bytes()):
				// The canonical Metaplex account is the explicit target.
			default:
				return unavailable, nil
			}
		} else if metadata != nil {
			return unavailable, nil
		}
	}
	if metadataAccount == nil {
		return unavailable, nil
	}
	if err := validateAccount(metadataAccount, metaplexProgram); err != nil {
		return MetadataResult{}, err
	}
	b = metadataAccount.Data
	if len(b) < 65 || b[0] != 4 || !bytes.Equal(b[33:65], mint.Bytes()) {
		return MetadataResult{}, errors.New("invalid Metaplex metadata identity")
	}
	c := metadataCursor{data: b[65:]}
	name, err := c.text(32)
	if err != nil {
		return MetadataResult{}, err
	}
	symbol, err := c.text(10)
	if err != nil {
		return MetadataResult{}, err
	}
	if _, err = c.text(200); err != nil {
		return MetadataResult{}, err
	}
	return textMetadata(strings.TrimRight(name, "\x00"), strings.TrimRight(symbol, "\x00"), "metaplex", pda)
}

func textMetadata(name, symbol, source, account string) (MetadataResult, error) {
	if strings.ContainsRune(name, 0) || strings.ContainsRune(symbol, 0) {
		return MetadataResult{}, errors.New("metadata text contains embedded NUL")
	}
	if strings.TrimSpace(name) == "" {
		name = ""
	}
	if strings.TrimSpace(symbol) == "" {
		symbol = ""
	}
	result := MetadataResult{Name: name, Symbol: symbol, Source: source, Account: account, Status: "ready"}
	if strings.TrimSpace(name) == "" && strings.TrimSpace(symbol) == "" {
		result.Status = "unavailable"
	}
	return result, nil
}
func allZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

type metadataCursor struct{ data []byte }

func (c *metadataCursor) u32() (uint32, error) {
	if len(c.data) < 4 {
		return 0, errors.New("truncated metadata length")
	}
	n := binary.LittleEndian.Uint32(c.data[:4])
	c.data = c.data[4:]
	return n, nil
}
func (c *metadataCursor) text(max uint32) (string, error) {
	n, err := c.u32()
	if err != nil {
		return "", err
	}
	if uint64(n) > uint64(len(c.data)) || (max > 0 && n > max) {
		return "", errors.New("invalid metadata string length")
	}
	s := c.data[:int(n)]
	c.data = c.data[int(n):]
	if !utf8.Valid(s) {
		return "", errors.New("invalid metadata UTF-8")
	}
	return string(s), nil
}
