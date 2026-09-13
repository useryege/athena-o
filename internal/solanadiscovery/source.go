package solanadiscovery

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gagliardetto/solana-go/base58"
)

type SourceResult struct{ Source, Program, Status string }

func issuanceSource(mint, tokenProgram string, top bool, ancestors []parsedInstruction, parent string) SourceResult {
	if top {
		return SourceResult{Source: "direct_token", Program: tokenProgram, Status: "identified"}
	}
	result := SourceResult{Source: "unknown", Program: parent, Status: "unrecognized"}
	var matched string
	for _, a := range ancestors {
		platform := launchPlatform(a, mint)
		if platform == "" {
			continue
		}
		if matched != "" && matched != platform {
			return SourceResult{Source: "unknown", Program: parent, Status: "unrecognized"}
		}
		matched = platform
		result = SourceResult{Source: platform, Program: a.ProgramID, Status: "identified"}
	}
	return result
}
func launchPlatform(a parsedInstruction, mint string) string {
	var selectors [][]byte
	var platform string
	index := 0
	switch a.ProgramID {
	case "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P":
		platform = "pump_fun"
		selectors = [][]byte{{24, 30, 200, 40, 5, 28, 7, 119}, {214, 144, 76, 236, 95, 139, 49, 180}}
	case "LanMV9sAd7wArD4vJFi2qDdfnVhFxYSUg6eADduJ3uj":
		platform = "raydium_launchlab"
		index = 6
		selectors = [][]byte{{175, 175, 109, 31, 13, 152, 155, 237}, {67, 153, 175, 39, 218, 16, 38, 32}, {37, 190, 126, 222, 44, 154, 171, 17}}
	default:
		return ""
	}
	if len(a.Accounts) <= index || a.Accounts[index] != mint {
		return ""
	}
	data, err := base58.Decode(a.Data)
	if err != nil || len(data) < 8 {
		return ""
	}
	for _, selector := range selectors {
		if bytes.Equal(data[:8], selector) {
			return platform
		}
	}
	return ""
}

// sourceFrames retains only a contiguous, evidenced CPI stack. The top-level
// instruction always remains a provable ancestor even when inner heights are absent.
type sourceFrames struct{ frames []parsedInstruction }

func (s *sourceFrames) before(raw json.RawMessage) ([]parsedInstruction, string, parsedInstruction, bool) {
	var instruction parsedInstruction
	if json.Unmarshal(raw, &instruction) != nil {
		return s.frames[:1], "", instruction, false
	}
	var height uint32
	if json.Unmarshal(instruction.StackHeight, &height) != nil || height < 2 {
		s.frames = s.frames[:1]
		return s.frames, "", instruction, false
	}
	if height <= uint32(len(s.frames))+1 {
		s.frames = s.frames[:int(height)-1]
		parent := s.frames[len(s.frames)-1].ProgramID
		if !validPublicKey(parent) {
			parent = ""
		}
		return s.frames, parent, instruction, true
	}
	s.frames = s.frames[:1]
	return s.frames, "", instruction, false
}

// ParseTransaction shares the successful initialization parser with live blocks.
func ParseTransaction(data []byte) ([]Project, error) {
	var tx struct {
		Slot      *uint64 `json:"slot"`
		BlockTime *int64  `json:"blockTime"`
	}
	if err := json.Unmarshal(data, &tx); err != nil {
		return nil, err
	}
	if tx.Slot == nil {
		return nil, errors.New("transaction has no slot")
	}
	block, err := json.Marshal(struct {
		BlockTime    *int64            `json:"blockTime"`
		Transactions []json.RawMessage `json:"transactions"`
	}{tx.BlockTime, []json.RawMessage{data}})
	if err != nil {
		return nil, err
	}
	return ParseBlock(*tx.Slot, block)
}
