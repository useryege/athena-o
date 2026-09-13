package solanadiscovery

import "time"

const (
	TokenProgram     = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	Token2022Program = "TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb"
)

// Project records only facts visible in the successful Mint initialization.
type Project struct {
	Mint            string
	TokenProgram    string
	Signature       string
	FeePayer        string
	MintAuthority   string
	FreezeAuthority string
	Decimals        uint32
	Slot            uint64
	BlockTime       int64
	DiscoveredAt    time.Time
}

type DiscoveryStatus struct {
	Status              string
	StartSlot           uint64
	LastProcessedSlot   uint64
	LatestFinalizedSlot uint64
	LastSuccessAt       time.Time
	TotalProjects       int64
	LastError           string
}
