package tradersync

import (
	"bytes"
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	CoreExchangeVersion    = "polygon137-core-v2-ccc0596074f4"
	NegRiskExchangeVersion = "polygon137-negrisk-v2-ccc0596074f4"
	ComboExchangeVersion   = "polygon137-combo-641b40ec414a"
)

type contractDeployment struct {
	address            common.Address
	codeHash           common.Hash
	implementation     common.Address
	implementationHash common.Hash
}
type sourceVersion struct {
	deployment                         contractDeployment
	collateralSymbol                   string
	collateralDecimals, sharesDecimals uint8
}

// IDs identify immutable Polygon 137 deployment/ABI evidence, not a latest alias.
var sourceVersions = map[string]sourceVersion{
	CoreExchangeVersion:    {deployment: contractDeployment{address: common.HexToAddress("0xE111180000d2663C0091e4f400237545B87B996B"), codeHash: common.HexToHash("0xa08da89bbac2063dfa6a705e70314d218d40fb2b2a6405442297c241fcd58401")}, collateralSymbol: "pUSD", collateralDecimals: 6, sharesDecimals: 6},
	NegRiskExchangeVersion: {deployment: contractDeployment{address: common.HexToAddress("0xe2222d279d744050d28e00520010520000310F59"), codeHash: common.HexToHash("0x04b857d48dcc38b3d484239569dc96a7a6c39bbb90ed2461227fc6e50ed5787d")}, collateralSymbol: "pUSD", collateralDecimals: 6, sharesDecimals: 6},
	ComboExchangeVersion:   {deployment: contractDeployment{address: common.HexToAddress("0xe3333700cA9d93003F00f0F71f8515005F6c00Aa"), codeHash: common.HexToHash("0xaaa52c8cc8a0e3fd27ce756cc6b4e70c51423e9b597b11f32d3e49f8b1fc890d"), implementation: common.HexToAddress("0x641b40ec414a076b9e79e703fc7bf4ebec248bb7"), implementationHash: common.HexToHash("0x42b8522e8b56d0587bc02e66a37e43614911580ded1c676a925f5373c7f2fcd6")}, collateralSymbol: "pUSD", collateralDecimals: 6, sharesDecimals: 6},
}

type VersionRPC interface {
	ChainID(context.Context) (*big.Int, error)
	HeaderByHash(context.Context, common.Hash) (*types.Header, error)
	StorageAtHash(context.Context, common.Address, common.Hash, common.Hash) ([]byte, error)
	CodeAtHash(context.Context, common.Address, common.Hash) ([]byte, error)
	// UpgradeLogs returns a non-nil empty slice only for a complete no-event result.
	UpgradeLogs(context.Context, common.Hash, common.Address) ([]types.Log, error)
}
type versionCacheKey struct {
	chain    uint64
	exchange common.Address
	block    common.Hash
}
type verifiedVersion struct {
	id     string
	height uint64
}
type VersionVerifier struct {
	node     VersionRPC
	mu       sync.Mutex
	verified map[versionCacheKey]verifiedVersion
	order    []versionCacheKey
}

func NewVersionVerifier(node VersionRPC) *VersionVerifier {
	return &VersionVerifier{node: node, verified: make(map[versionCacheKey]verifiedVersion)}
}

type versionEvidenceError struct {
	reason string
	cause  error
}

func (e *versionEvidenceError) Error() string { return "execution version unverified: " + e.reason }
func (e *versionEvidenceError) Unwrap() error { return e.cause }
func (e *versionEvidenceError) GRPCStatus() *status.Status {
	return status.New(codes.FailedPrecondition, e.Error())
}
func unverifiedVersion(reason string, cause error) error {
	return &versionEvidenceError{reason: reason, cause: cause}
}

// Verify only caches successful immutable block-code evidence. Canonical/finality
// evidence is independent and must still be refreshed by ConfirmReceived.
func (v *VersionVerifier) Verify(ctx context.Context, raw types.Log) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", unverifiedVersion("cancelled", err)
	}
	if raw.Removed || raw.BlockHash == (common.Hash{}) {
		return "", unverifiedVersion("invalid_log_location", nil)
	}
	chain, err := v.node.ChainID(ctx)
	if err != nil || chain == nil || chain.Cmp(big.NewInt(137)) != 0 {
		return "", unverifiedVersion("polygon_chain_unavailable", err)
	}
	key := versionCacheKey{137, raw.Address, raw.BlockHash}
	v.mu.Lock()
	cached := v.verified[key]
	v.mu.Unlock()
	if cached.id != "" {
		if cached.height != raw.BlockNumber {
			return "", unverifiedVersion("cached_header_height_mismatch", nil)
		}
		return cached.id, nil
	}
	var id string
	var registered sourceVersion
	for version, source := range sourceVersions {
		if source.deployment.address == raw.Address {
			id, registered = version, source
			break
		}
	}
	if id == "" {
		return "", unverifiedVersion("unknown_exchange", nil)
	}
	header, err := v.node.HeaderByHash(ctx, raw.BlockHash)
	if err != nil || header == nil || header.Number == nil || !header.Number.IsUint64() || header.Number.Uint64() != raw.BlockNumber || header.Hash() != raw.BlockHash || header.ParentHash == (common.Hash{}) {
		return "", unverifiedVersion("known_header_unavailable", err)
	}
	if err = verifyDeploymentAtHashes(ctx, v.node, registered.deployment, raw.BlockHash, header.ParentHash); err != nil {
		return "", err
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if _, exists := v.verified[key]; !exists {
		// Bounded FIFO: eviction repeats verification, never broadens acceptance.
		if len(v.order) == 256 {
			delete(v.verified, v.order[0])
			v.order = v.order[1:]
		}
		v.verified[key] = verifiedVersion{id: id, height: raw.BlockNumber}
		v.order = append(v.order, key)
	}
	return id, nil
}

var implementationSlot = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")

// Preconditions: the caller established chain 137 for this node instance and
// verified the known candidate header/hash and its actual ParentHash. This helper
// only checks deployment code/slots/upgrade evidence at those two hashes, with no
// trade dependency. Module metadata can use its own approved deployment registry.
func verifyDeploymentAtHashes(ctx context.Context, node VersionRPC, deployment contractDeployment, block, parent common.Hash) error {
	for _, hash := range []common.Hash{parent, block} {
		code, err := node.CodeAtHash(ctx, deployment.address, hash)
		if err != nil || len(code) == 0 || crypto.Keccak256Hash(code) != deployment.codeHash {
			return unverifiedVersion("deployment_code_unverified", err)
		}
		if deployment.implementation == (common.Address{}) {
			continue
		}
		slot, err := node.StorageAtHash(ctx, deployment.address, implementationSlot, hash)
		if err != nil || len(slot) != 32 || !bytes.Equal(slot[:12], make([]byte, 12)) || common.BytesToAddress(slot[12:]) != deployment.implementation {
			return unverifiedVersion("implementation_slot_unverified", err)
		}
		code, err = node.CodeAtHash(ctx, deployment.implementation, hash)
		if err != nil || len(code) == 0 || crypto.Keccak256Hash(code) != deployment.implementationHash {
			return unverifiedVersion("implementation_code_unverified", err)
		}
	}
	if deployment.implementation != (common.Address{}) {
		// Matching block-end slots cannot rule out an upgrade away and back. The
		// approved actual proxy and fixed implementation guarantee the first upgrade
		// emits Upgraded; any event (or failed query) makes execution ambiguous.
		upgrades, err := node.UpgradeLogs(ctx, block, deployment.address)
		if err != nil || upgrades == nil || len(upgrades) > 0 {
			return unverifiedVersion("same_block_upgrade_unverified", err)
		}
	}
	return nil
}
