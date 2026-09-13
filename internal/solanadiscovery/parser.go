package solanadiscovery

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/base58"
)

type parsedBlock struct {
	BlockTime    *int64 `json:"blockTime"`
	Transactions []struct {
		Transaction struct {
			Signatures []string `json:"signatures"`
			Message    struct {
				AccountKeys []struct {
					Pubkey string `json:"pubkey"`
				} `json:"accountKeys"`
				Instructions []json.RawMessage `json:"instructions"`
			} `json:"message"`
		} `json:"transaction"`
		Meta *struct {
			Err               json.RawMessage `json:"err"`
			InnerInstructions []struct {
				Index        uint32            `json:"index"`
				Instructions []json.RawMessage `json:"instructions"`
			} `json:"innerInstructions"`
		} `json:"meta"`
	} `json:"transactions"`
}

type parsedInstruction struct {
	StackHeight json.RawMessage `json:"stackHeight"`
	ProgramID   string          `json:"programId"`
	Parsed      json.RawMessage `json:"parsed"`
	Accounts    []string        `json:"accounts"`
	Data        string          `json:"data"`
}

type mintInstruction struct {
	Type string `json:"type"`
	Info struct {
		Mint            string  `json:"mint"`
		Decimals        *uint32 `json:"decimals"`
		MintAuthority   string  `json:"mintAuthority"`
		FreezeAuthority *string `json:"freezeAuthority"`
	} `json:"info"`
}

// ParseBlock parses a jsonParsed/full getBlock result. A malformed recognized
// initialization is fatal so the scanner can retry without losing the slot.
func ParseBlock(slot uint64, data []byte) ([]Project, error) {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil, fmt.Errorf("slot %d: missing block", slot)
	}
	var block parsedBlock
	if err := json.Unmarshal(data, &block); err != nil {
		return nil, fmt.Errorf("slot %d: decode block: %w", slot, err)
	}
	if block.Transactions == nil {
		return nil, fmt.Errorf("slot %d: missing transactions", slot)
	}
	projects := make([]Project, 0)
	seen := make(map[string]struct{})
	for txIndex, tx := range block.Transactions {
		if tx.Meta == nil {
			return nil, fmt.Errorf("slot %d transaction %d: missing metadata", slot, txIndex)
		}
		if len(bytes.TrimSpace(tx.Meta.Err)) == 0 {
			return nil, fmt.Errorf("slot %d transaction %d: missing execution status", slot, txIndex)
		}
		if !isJSONNull(tx.Meta.Err) {
			continue
		}
		if len(tx.Transaction.Signatures) == 0 || len(tx.Transaction.Message.AccountKeys) == 0 || tx.Transaction.Message.Instructions == nil {
			return nil, fmt.Errorf("slot %d transaction %d: missing signature, fee payer or instructions", slot, txIndex)
		}
		signature := tx.Transaction.Signatures[0]
		feePayer := tx.Transaction.Message.AccountKeys[0].Pubkey
		if signature == "" || !validPublicKey(feePayer) {
			return nil, fmt.Errorf("slot %d transaction %d: invalid signature or fee payer", slot, txIndex)
		}
		var ancestors []parsedInstruction
		var parent string
		topLevel := true
		appendCandidate := func(mint, program, authority, freeze string, decimals uint32) {
			source := issuanceSource(mint, program, topLevel, ancestors, parent)
			if _, exists := seen[mint]; exists {
				return
			}
			seen[mint] = struct{}{}
			blockTime := int64(0)
			if block.BlockTime != nil {
				blockTime = *block.BlockTime
			}
			projects = append(projects, Project{
				MetadataStatus: "pending", IssuanceSource: source.Source, IssuanceProgram: source.Program, SourceStatus: source.Status,
				Mint: mint, TokenProgram: program, Signature: signature, FeePayer: feePayer,
				MintAuthority: authority, FreezeAuthority: freeze,
				Decimals: decimals, Slot: slot, BlockTime: blockTime,
			})
		}
		consume := func(raw json.RawMessage) error {
			var instruction parsedInstruction
			if err := json.Unmarshal(raw, &instruction); err != nil {
				return fmt.Errorf("decode instruction: %w", err)
			}
			if instruction.ProgramID != TokenProgram && instruction.ProgramID != Token2022Program {
				return nil
			}
			if len(instruction.Parsed) == 0 || isJSONNull(instruction.Parsed) {
				mint, authority, freeze, decimals, recognized, err := decodeRawMint(instruction)
				if err != nil {
					return err
				}
				if recognized {
					appendCandidate(mint, instruction.ProgramID, authority, freeze, decimals)
				}
				return nil
			}
			var header struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(instruction.Parsed, &header); err != nil {
				return nil // A non-jsonParsed instruction has no recognized type.
			}
			if header.Type != "initializeMint" && header.Type != "initializeMint2" {
				return nil
			}
			var parsed mintInstruction
			if err := json.Unmarshal(instruction.Parsed, &parsed); err != nil {
				return fmt.Errorf("invalid %s fields: %w", header.Type, err)
			}
			if !validPublicKey(parsed.Info.Mint) || !validPublicKey(parsed.Info.MintAuthority) || parsed.Info.Decimals == nil || *parsed.Info.Decimals > 255 {
				return fmt.Errorf("invalid %s fields", parsed.Type)
			}
			freeze := ""
			if parsed.Info.FreezeAuthority != nil {
				freeze = *parsed.Info.FreezeAuthority
				if !validPublicKey(freeze) {
					return fmt.Errorf("invalid %s freeze authority", parsed.Type)
				}
			}
			appendCandidate(parsed.Info.Mint, instruction.ProgramID, parsed.Info.MintAuthority, freeze, *parsed.Info.Decimals)
			return nil
		}
		innerByIndex := make(map[uint32][]json.RawMessage, len(tx.Meta.InnerInstructions))
		for _, group := range tx.Meta.InnerInstructions {
			if uint64(group.Index) >= uint64(len(tx.Transaction.Message.Instructions)) {
				return nil, fmt.Errorf("slot %d transaction %d: invalid CPI index %d", slot, txIndex, group.Index)
			}
			innerByIndex[group.Index] = append(innerByIndex[group.Index], group.Instructions...)
		}
		for index, raw := range tx.Transaction.Message.Instructions {
			topLevel = true
			ancestors = nil
			parent = ""
			if err := consume(raw); err != nil {
				return nil, fmt.Errorf("slot %d transaction %d: %w", slot, txIndex, err)
			}
			var outer parsedInstruction
			_ = json.Unmarshal(raw, &outer)
			frames := sourceFrames{frames: []parsedInstruction{outer}}
			topLevel = false
			for _, inner := range innerByIndex[uint32(index)] {
				var current parsedInstruction
				var push bool
				ancestors, parent, current, push = frames.before(inner)
				if err := consume(inner); err != nil {
					return nil, fmt.Errorf("slot %d transaction %d CPI %d: %w", slot, txIndex, index, err)
				}
				if push {
					frames.frames = append(frames.frames, current)
				}
			}
		}
	}
	return projects, nil
}

func decodeRawMint(instruction parsedInstruction) (mint, authority, freeze string, decimals uint32, recognized bool, err error) {
	if instruction.Data == "" {
		return "", "", "", 0, false, fmt.Errorf("missing raw token instruction data")
	}
	data, err := base58.Decode(instruction.Data)
	if err != nil {
		return "", "", "", 0, false, fmt.Errorf("decode raw token instruction: %w", err)
	}
	if len(data) == 0 {
		return "", "", "", 0, false, fmt.Errorf("empty raw token instruction")
	}
	if data[0] != 0 && data[0] != 20 {
		return "", "", "", 0, false, nil
	}
	if len(data) < 35 || len(instruction.Accounts) == 0 || !validPublicKey(instruction.Accounts[0]) {
		return "", "", "", 0, true, fmt.Errorf("invalid raw Mint initialization")
	}
	mint = instruction.Accounts[0]
	authority = solana.PublicKeyFromBytes(data[2:34]).String()
	decimals = uint32(data[1])
	switch data[34] {
	case 0:
		// SPL ignores trailing bytes after the optional authority.
	case 1:
		if len(data) < 67 {
			return "", "", "", 0, true, fmt.Errorf("invalid raw Mint freeze authority length")
		}
		freeze = solana.PublicKeyFromBytes(data[35:67]).String()
	default:
		return "", "", "", 0, true, fmt.Errorf("invalid raw Mint freeze authority option")
	}
	return mint, authority, freeze, decimals, true, nil
}

func isJSONNull(value json.RawMessage) bool {
	trimmed := bytes.TrimSpace(value)
	return bytes.Equal(trimmed, []byte("null"))
}

func validPublicKey(value string) bool {
	_, err := solana.PublicKeyFromBase58(value)
	return err == nil
}
