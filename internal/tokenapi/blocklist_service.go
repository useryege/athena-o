package tokenapi

import (
	"context"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
	athenacommon "github.com/useryege/athena/common"
	tokenstore "github.com/useryege/athena/internal/token/store"
	"github.com/useryege/athena/internal/tokenapi/apiclient"
	"github.com/useryege/athena/util/ethws"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) GetContractCodeBlocklistEntry(ctx context.Context, req *apiclient.GetContractCodeBlocklistEntryRequest) (*apiclient.GetContractCodeBlocklistEntryResponse, error) {
	store, e := requiredStore(s.tokenStore())
	if e != nil {
		return nil, e
	}
	hash, e := parseHashField("code_hash", req.GetCodeHash())
	if e != nil {
		return nil, e
	}
	item, e := store.GetContractCodeBlocklistEntry(ctx, hash)
	if e != nil {
		return nil, wrapStoreError("get contract code blocklist entry", e)
	}
	if item == nil {
		return &apiclient.GetContractCodeBlocklistEntryResponse{}, nil
	}
	return &apiclient.GetContractCodeBlocklistEntryResponse{Found: true, Entry: mapContractCodeBlocklistEntry(*item)}, nil
}
func (s *Service) ListContractCodeBlocklistEntries(ctx context.Context, _ *apiclient.ListContractCodeBlocklistEntriesRequest) (*apiclient.ListContractCodeBlocklistEntriesResponse, error) {
	store, e := requiredStore(s.tokenStore())
	if e != nil {
		return nil, e
	}
	items, e := store.ListContractCodeBlocklistEntries(ctx)
	if e != nil {
		return nil, wrapStoreError("list contract code blocklist entries", e)
	}
	return &apiclient.ListContractCodeBlocklistEntriesResponse{Entries: mapContractCodeBlocklistEntries(items)}, nil
}
func (s *Service) CreateContractCodeBlocklistEntry(ctx context.Context, req *apiclient.CreateContractCodeBlocklistEntryRequest) (*apiclient.CreateContractCodeBlocklistEntryResponse, error) {
	store, e := requiredStore(s.tokenStore())
	if e != nil {
		return nil, e
	}
	contract, e := parseAddressField("source_contract", req.GetSourceContract())
	if e != nil {
		return nil, e
	}
	hash, e := s.fetchContractCodeHash(ctx, req.GetSourceChainId(), contract)
	if e != nil {
		return nil, e
	}
	if e = store.CreateContractCodeBlocklistEntry(ctx, tokenstore.ContractCodeBlocklistEntry{CodeHash: hash, Note: req.GetNote(), SourceChainID: req.GetSourceChainId(), SourceContract: contract}); e != nil {
		return nil, wrapStoreError("create contract code blocklist entry", e)
	}
	return &apiclient.CreateContractCodeBlocklistEntryResponse{}, nil
}
func (s *Service) UpdateContractCodeBlocklistEntry(ctx context.Context, req *apiclient.UpdateContractCodeBlocklistEntryRequest) (*apiclient.UpdateContractCodeBlocklistEntryResponse, error) {
	store, e := requiredStore(s.tokenStore())
	if e != nil {
		return nil, e
	}
	hash, e := parseHashField("code_hash", req.GetCodeHash())
	if e != nil {
		return nil, e
	}
	n, e := store.UpdateContractCodeBlocklistEntryNote(ctx, hash, req.GetNote())
	if e != nil {
		return nil, wrapStoreError("update contract code blocklist entry", e)
	}
	return &apiclient.UpdateContractCodeBlocklistEntryResponse{UpdatedCount: n}, nil
}
func (s *Service) DeleteContractCodeBlocklistEntry(ctx context.Context, req *apiclient.DeleteContractCodeBlocklistEntryRequest) (*apiclient.DeleteContractCodeBlocklistEntryResponse, error) {
	store, e := requiredStore(s.tokenStore())
	if e != nil {
		return nil, e
	}
	hash, e := parseHashField("code_hash", req.GetCodeHash())
	if e != nil {
		return nil, e
	}
	n, e := store.DeleteContractCodeBlocklistEntry(ctx, hash)
	if e != nil {
		return nil, wrapStoreError("delete contract code blocklist entry", e)
	}
	return &apiclient.DeleteContractCodeBlocklistEntryResponse{DeletedCount: n}, nil
}

func (s *Service) GetWalletBlocklistEntry(ctx context.Context, req *apiclient.GetWalletBlocklistEntryRequest) (*apiclient.GetWalletBlocklistEntryResponse, error) {
	store, e := requiredStore(s.tokenStore())
	if e != nil {
		return nil, e
	}
	wallet, e := parseAddressField("wallet", req.GetWallet())
	if e != nil {
		return nil, e
	}
	item, e := store.GetWalletBlocklistEntry(ctx, wallet)
	if e != nil {
		return nil, wrapStoreError("get wallet blocklist entry", e)
	}
	if item == nil {
		return &apiclient.GetWalletBlocklistEntryResponse{}, nil
	}
	return &apiclient.GetWalletBlocklistEntryResponse{Found: true, Entry: mapWalletBlocklistEntry(*item)}, nil
}
func (s *Service) ListWalletBlocklistEntries(ctx context.Context, _ *apiclient.ListWalletBlocklistEntriesRequest) (*apiclient.ListWalletBlocklistEntriesResponse, error) {
	store, e := requiredStore(s.tokenStore())
	if e != nil {
		return nil, e
	}
	items, e := store.ListWalletBlocklistEntries(ctx)
	if e != nil {
		return nil, wrapStoreError("list wallet blocklist entries", e)
	}
	return &apiclient.ListWalletBlocklistEntriesResponse{Entries: mapWalletBlocklistEntries(items)}, nil
}
func (s *Service) CreateWalletBlocklistEntry(ctx context.Context, req *apiclient.CreateWalletBlocklistEntryRequest) (*apiclient.CreateWalletBlocklistEntryResponse, error) {
	store, e := requiredStore(s.tokenStore())
	if e != nil {
		return nil, e
	}
	wallet, e := parseAddressField("wallet", req.GetWallet())
	if e != nil {
		return nil, e
	}
	if e = store.CreateWalletBlocklistEntry(ctx, tokenstore.WalletBlocklistEntry{Wallet: wallet, Note: req.GetNote()}); e != nil {
		return nil, wrapStoreError("create wallet blocklist entry", e)
	}
	return &apiclient.CreateWalletBlocklistEntryResponse{}, nil
}
func (s *Service) UpdateWalletBlocklistEntry(ctx context.Context, req *apiclient.UpdateWalletBlocklistEntryRequest) (*apiclient.UpdateWalletBlocklistEntryResponse, error) {
	store, e := requiredStore(s.tokenStore())
	if e != nil {
		return nil, e
	}
	wallet, e := parseAddressField("wallet", req.GetWallet())
	if e != nil {
		return nil, e
	}
	n, e := store.UpdateWalletBlocklistEntryNote(ctx, wallet, req.GetNote())
	if e != nil {
		return nil, wrapStoreError("update wallet blocklist entry", e)
	}
	return &apiclient.UpdateWalletBlocklistEntryResponse{UpdatedCount: n}, nil
}
func (s *Service) DeleteWalletBlocklistEntry(ctx context.Context, req *apiclient.DeleteWalletBlocklistEntryRequest) (*apiclient.DeleteWalletBlocklistEntryResponse, error) {
	store, e := requiredStore(s.tokenStore())
	if e != nil {
		return nil, e
	}
	wallet, e := parseAddressField("wallet", req.GetWallet())
	if e != nil {
		return nil, e
	}
	n, e := store.DeleteWalletBlocklistEntry(ctx, wallet)
	if e != nil {
		return nil, wrapStoreError("delete wallet blocklist entry", e)
	}
	return &apiclient.DeleteWalletBlocklistEntryResponse{DeletedCount: n}, nil
}

func (s *Service) fetchContractCodeHash(ctx context.Context, chainID int64, contract ethcommon.Address) (ethcommon.Hash, error) {
	urls, ok := s.nodeConfig(chainID)
	if !ok {
		return ethcommon.Hash{}, status.Errorf(codes.InvalidArgument, "source_chain_id must be %d or %d", athenacommon.ChainIDEthereumMainnet, athenacommon.ChainIDBSCMainnet)
	}
	urls = ethws.NormalizeEndpoints(urls)
	if len(urls) == 0 {
		return ethcommon.Hash{}, status.Errorf(codes.FailedPrecondition, "node websocket URLs are not configured for %s", athenacommon.ChainName(chainID))
	}
	client, endpoint, e := ethws.DialFastestContext(ctx, urls, chainID, s.nodeWSUseProxy())
	if e != nil {
		return ethcommon.Hash{}, status.Errorf(codes.Unavailable, "connect to node: %v", e)
	}
	defer client.Close()
	log.WithFields(log.Fields{"chain_id": chainID, "node_ws_url": ethws.RedactEndpoint(endpoint)}).Info("token API connected to node websocket")
	code, e := client.CodeAt(ctx, contract, nil)
	if e != nil {
		return ethcommon.Hash{}, status.Errorf(codes.Unavailable, "fetch contract bytecode: %v", e)
	}
	if len(code) == 0 {
		return ethcommon.Hash{}, status.Error(codes.FailedPrecondition, "source_contract has no deployed bytecode")
	}
	return crypto.Keccak256Hash(code), nil
}
