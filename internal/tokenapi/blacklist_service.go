package tokenapi

import (
	"context"
	"fmt"
	"strings"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	athenacommon "github.com/useryege/athena/common"
	tokenstore "github.com/useryege/athena/internal/token/store"
	"github.com/useryege/athena/internal/tokenapi/apiclient"
	"github.com/useryege/athena/util/ethws"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) GetBytecodeBlacklist(ctx context.Context, req *apiclient.GetBytecodeBlacklistRequest) (*apiclient.GetBytecodeBlacklistResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	codeHash, err := parseHashField("code_hash", req.GetCodeHash())
	if err != nil {
		return nil, err
	}
	item, err := store.GetBytecodeBlacklistEntry(ctx, codeHash)
	if err != nil {
		return nil, wrapStoreError("get bytecode blacklist", err)
	}
	if item == nil {
		return &apiclient.GetBytecodeBlacklistResponse{}, nil
	}
	return &apiclient.GetBytecodeBlacklistResponse{
		Found:             true,
		BytecodeBlacklist: mapBytecodeBlacklist(*item),
	}, nil
}

func (s *Service) ListBytecodeBlacklists(ctx context.Context, _ *apiclient.ListBytecodeBlacklistsRequest) (*apiclient.ListBytecodeBlacklistsResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	items, err := store.ListBytecodeBlacklistEntries(ctx)
	if err != nil {
		return nil, wrapStoreError("list bytecode blacklists", err)
	}
	return &apiclient.ListBytecodeBlacklistsResponse{
		BytecodeBlacklists: mapBytecodeBlacklists(items),
	}, nil
}

func (s *Service) CreateBytecodeBlacklist(ctx context.Context, req *apiclient.CreateBytecodeBlacklistRequest) (*apiclient.CreateBytecodeBlacklistResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	sourceChainID := req.GetSourceChainId()
	sourceContract, err := parseAddressField("source_contract", req.GetSourceContract())
	if err != nil {
		return nil, err
	}
	codeHash, err := s.fetchContractCodeHash(ctx, sourceChainID, sourceContract)
	if err != nil {
		return nil, err
	}
	if err := store.AddBytecodeBlacklistEntry(ctx, tokenstore.BytecodeBlacklistEntry{
		CodeHash:       codeHash,
		Note:           req.GetNote(),
		SourceChainID:  sourceChainID,
		SourceContract: sourceContract,
	}); err != nil {
		return nil, wrapStoreError("create bytecode blacklist", err)
	}
	return &apiclient.CreateBytecodeBlacklistResponse{}, nil
}

func (s *Service) fetchContractCodeHash(ctx context.Context, chainID int64, contract ethcommon.Address) (ethcommon.Hash, error) {
	nodeWSURL, ok := s.nodeConfig(chainID)
	if !ok {
		return ethcommon.Hash{}, status.Errorf(codes.InvalidArgument, "source_chain_id must be %d or %d", athenacommon.ChainIDEthereumMainnet, athenacommon.ChainIDBSCMainnet)
	}
	nodeWSURL = strings.TrimSpace(nodeWSURL)
	if nodeWSURL == "" {
		return ethcommon.Hash{}, status.Errorf(codes.FailedPrecondition, "node websocket URL is not configured for %s", athenacommon.ChainName(chainID))
	}
	client, err := ethws.DialContext(ctx, nodeWSURL, s.nodeWSUseProxy())
	if err != nil {
		return ethcommon.Hash{}, status.Errorf(codes.Unavailable, "connect to %s node websocket: %v", athenacommon.ChainName(chainID), err)
	}
	defer client.Close()

	nodeChainID, err := client.ChainID(ctx)
	if err != nil {
		return ethcommon.Hash{}, status.Errorf(codes.Unavailable, "get %s node chain id: %v", athenacommon.ChainName(chainID), err)
	}
	if nodeChainID == nil || nodeChainID.Int64() != chainID {
		return ethcommon.Hash{}, status.Error(codes.FailedPrecondition, fmt.Sprintf("node returned chain_id %v for configured chain_id %d", nodeChainID, chainID))
	}
	code, err := client.CodeAt(ctx, contract, nil)
	if err != nil {
		return ethcommon.Hash{}, status.Errorf(codes.Unavailable, "fetch contract bytecode: %v", err)
	}
	if len(code) == 0 {
		return ethcommon.Hash{}, status.Errorf(codes.FailedPrecondition, "source_contract has no deployed bytecode on %s", athenacommon.ChainName(chainID))
	}
	return crypto.Keccak256Hash(code), nil
}

func (s *Service) UpdateBytecodeBlacklist(ctx context.Context, req *apiclient.UpdateBytecodeBlacklistRequest) (*apiclient.UpdateBytecodeBlacklistResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	codeHash, err := parseHashField("code_hash", req.GetCodeHash())
	if err != nil {
		return nil, err
	}
	updated, err := store.UpdateBytecodeBlacklistNote(ctx, codeHash, req.GetNote())
	if err != nil {
		return nil, wrapStoreError("update bytecode blacklist", err)
	}
	return &apiclient.UpdateBytecodeBlacklistResponse{UpdatedCount: updated}, nil
}

func (s *Service) DeleteBytecodeBlacklist(ctx context.Context, req *apiclient.DeleteBytecodeBlacklistRequest) (*apiclient.DeleteBytecodeBlacklistResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	codeHash, err := parseHashField("code_hash", req.GetCodeHash())
	if err != nil {
		return nil, err
	}
	deleted, err := store.DeleteBytecodeBlacklistEntry(ctx, codeHash)
	if err != nil {
		return nil, wrapStoreError("delete bytecode blacklist", err)
	}
	return &apiclient.DeleteBytecodeBlacklistResponse{DeletedCount: deleted}, nil
}

func (s *Service) GetWalletBlacklist(ctx context.Context, req *apiclient.GetWalletBlacklistRequest) (*apiclient.GetWalletBlacklistResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	wallet, err := parseAddressField("wallet", req.GetWallet())
	if err != nil {
		return nil, err
	}
	item, err := store.GetWalletBlacklistEntry(ctx, wallet)
	if err != nil {
		return nil, wrapStoreError("get wallet blacklist", err)
	}
	if item == nil {
		return &apiclient.GetWalletBlacklistResponse{}, nil
	}
	return &apiclient.GetWalletBlacklistResponse{
		Found:           true,
		WalletBlacklist: mapWalletBlacklist(*item),
	}, nil
}

func (s *Service) ListWalletBlacklists(ctx context.Context, _ *apiclient.ListWalletBlacklistsRequest) (*apiclient.ListWalletBlacklistsResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	items, err := store.ListWalletBlacklistEntries(ctx)
	if err != nil {
		return nil, wrapStoreError("list wallet blacklists", err)
	}
	return &apiclient.ListWalletBlacklistsResponse{
		WalletBlacklists: mapWalletBlacklists(items),
	}, nil
}

func (s *Service) CreateWalletBlacklist(ctx context.Context, req *apiclient.CreateWalletBlacklistRequest) (*apiclient.CreateWalletBlacklistResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	wallet, err := parseAddressField("wallet", req.GetWallet())
	if err != nil {
		return nil, err
	}
	if err := store.AddWalletBlacklistEntry(ctx, tokenstore.WalletBlacklistEntry{
		Wallet: wallet,
		Note:   req.GetNote(),
	}); err != nil {
		return nil, wrapStoreError("create wallet blacklist", err)
	}
	return &apiclient.CreateWalletBlacklistResponse{}, nil
}

func (s *Service) UpdateWalletBlacklist(ctx context.Context, req *apiclient.UpdateWalletBlacklistRequest) (*apiclient.UpdateWalletBlacklistResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	wallet, err := parseAddressField("wallet", req.GetWallet())
	if err != nil {
		return nil, err
	}
	updated, err := store.UpdateWalletBlacklistNote(ctx, wallet, req.GetNote())
	if err != nil {
		return nil, wrapStoreError("update wallet blacklist", err)
	}
	return &apiclient.UpdateWalletBlacklistResponse{UpdatedCount: updated}, nil
}

func (s *Service) DeleteWalletBlacklist(ctx context.Context, req *apiclient.DeleteWalletBlacklistRequest) (*apiclient.DeleteWalletBlacklistResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	wallet, err := parseAddressField("wallet", req.GetWallet())
	if err != nil {
		return nil, err
	}
	deleted, err := store.DeleteWalletBlacklistEntry(ctx, wallet)
	if err != nil {
		return nil, wrapStoreError("delete wallet blacklist", err)
	}
	return &apiclient.DeleteWalletBlacklistResponse{DeletedCount: deleted}, nil
}
