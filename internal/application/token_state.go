package application

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type TokenMetadataSnapshot struct {
	Name        string
	Symbol      string
	Decimals    uint8
	TotalSupply string
	Address     common.Address

	SourceCode    string
	SourceCodeABI string

	StateHash string
}

func NewTokenMetadataSnapshot(metadata *TokenMetadata) TokenMetadataSnapshot {
	totalSupply := "0"
	if metadata.TotalSupply != nil {
		totalSupply = metadata.TotalSupply.String()
	}

	s := TokenMetadataSnapshot{
		Name:          metadata.Name,
		Symbol:        metadata.Symbol,
		Decimals:      metadata.Decimals,
		TotalSupply:   totalSupply,
		Address:       metadata.Address,
		SourceCode:    metadata.SourceCode,
		SourceCodeABI: metadata.SourceCodeABI,
	}

	s.StateHash = s.Hash()
	return s
}

func (s TokenMetadataSnapshot) ToTokenMetadata() *TokenMetadata {
	totalSupply := new(big.Int)

	if s.TotalSupply != "" {
		totalSupply.SetString(s.TotalSupply, 10)
	}

	return &TokenMetadata{
		Name:          s.Name,
		Symbol:        s.Symbol,
		Decimals:      s.Decimals,
		TotalSupply:   totalSupply,
		Address:       s.Address,
		SourceCode:    s.SourceCode,
		SourceCodeABI: s.SourceCodeABI,
	}
}

func (s TokenMetadataSnapshot) Hash() string {
	h := sha256.New()

	h.Write([]byte(s.Name))
	h.Write([]byte{0})

	h.Write([]byte(s.Symbol))
	h.Write([]byte{0})

	h.Write([]byte{s.Decimals})
	h.Write([]byte{0})

	h.Write([]byte(s.TotalSupply))
	h.Write([]byte{0})

	h.Write(s.Address.Bytes())
	h.Write([]byte{0})

	h.Write([]byte(s.SourceCode))
	h.Write([]byte{0})

	h.Write([]byte(s.SourceCodeABI))

	return hex.EncodeToString(h.Sum(nil))
}

type TokenMetadataDiff struct {
	ChangedFields []string
}

func CompareTokenMetadata(oldState, newState TokenMetadataSnapshot) TokenMetadataDiff {
	var fields []string

	if oldState.Name != newState.Name {
		fields = append(fields, "Name")
	}

	if oldState.Symbol != newState.Symbol {
		fields = append(fields, "Symbol")
	}

	if oldState.Decimals != newState.Decimals {
		fields = append(fields, "Decimals")
	}

	if oldState.TotalSupply != newState.TotalSupply {
		fields = append(fields, "TotalSupply")
	}

	if oldState.Address != newState.Address {
		fields = append(fields, "Address")
	}

	if oldState.SourceCode != newState.SourceCode {
		fields = append(fields, "SourceCode")
	}

	if oldState.SourceCodeABI != newState.SourceCodeABI {
		fields = append(fields, "SourceCodeABI")
	}

	return TokenMetadataDiff{
		ChangedFields: fields,
	}
}

func (d TokenMetadataDiff) Changed() bool {
	return len(d.ChangedFields) > 0
}
