package solidity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/useryege/athena/internal/solidity/apiclient"
	"github.com/useryege/athena/internal/solidity/sourcequality"
	soliditystore "github.com/useryege/athena/internal/solidity/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/ethereumapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	sourceOriginThirdPartyAPI = "third_party_api"
)

type Service struct {
	apiclient.UnimplementedSolidityServiceServer
	store                 *soliditystore.SQLStore
	nodeClient            *ethclient.Client
	chainID               int64
	apiFetcher            ethereumapi.EthereumAPI
	sourceQualityAnalyzer sourcequality.Analyzer
	codeAtFunc            func(ctx context.Context, contract common.Address) ([]byte, error)
	startStopMu           sync.Mutex
	started               bool
}

type ServiceOpts struct {
	Store                 *soliditystore.SQLStore
	NodeClient            *ethclient.Client
	ChainID               int64
	APIFetcher            ethereumapi.EthereumAPI
	SourceQualityAnalyzer sourcequality.Analyzer
	CodeAtFunc            func(ctx context.Context, contract common.Address) ([]byte, error)
}

func NewService(opts ServiceOpts) *Service {
	return &Service{
		store:                 opts.Store,
		nodeClient:            opts.NodeClient,
		chainID:               opts.ChainID,
		apiFetcher:            opts.APIFetcher,
		sourceQualityAnalyzer: opts.SourceQualityAnalyzer,
		codeAtFunc:            opts.CodeAtFunc,
	}
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.started {
		return nil
	}
	if s.store == nil {
		return status.Error(codes.FailedPrecondition, "solidity store is required")
	}
	s.started = true
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	s.started = false
	return nil
}

func (s *Service) GetSolidityStatus(context.Context, *apiclient.GetSolidityStatusRequest) (*v1alpha1.SolidityStatus, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &v1alpha1.SolidityStatus{
		Started: started,
		Status:  statusText,
	}, nil
}

func (s *Service) GetContractSourceInfo(ctx context.Context, req *apiclient.GetContractSourceInfoRequest) (*v1alpha1.ContractSourceInfo, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "solidity store is required")
	}
	chainID := s.resolveRequestChainID(req.GetChainId())
	if err := s.validateChainID(chainID); err != nil {
		return nil, err
	}
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	contract := common.HexToAddress(req.GetContract())
	codeHash, err := s.resolveContractBytecode(ctx, chainID, contract)
	if err != nil {
		return nil, err
	}
	if err := s.enrichBytecodeSource(ctx, contract, codeHash); err != nil {
		return nil, err
	}
	return s.contractSourceInfo(ctx, chainID, contract, codeHash)
}

func (s *Service) ListBytecodeBlacklistEntries(ctx context.Context, _ *apiclient.ListBytecodeBlacklistEntriesRequest) (*apiclient.ListBytecodeBlacklistEntriesResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "solidity store is required")
	}
	records, err := s.store.ListBytecodeBlacklistEntries(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*v1alpha1.BytecodeBlacklistEntry, 0, len(records))
	for _, record := range records {
		items = append(items, bytecodeBlacklistEntryToAPI(record))
	}
	return &apiclient.ListBytecodeBlacklistEntriesResponse{Items: items}, nil
}

func (s *Service) AddBytecodeBlacklistEntry(ctx context.Context, req *apiclient.AddBytecodeBlacklistEntryRequest) (*apiclient.AddBytecodeBlacklistEntryResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "solidity store is required")
	}
	codeHash, sourceChainID, sourceContract, err := s.blacklistRequestCodeHash(ctx, req)
	if err != nil {
		return nil, err
	}
	record := soliditystore.BytecodeBlacklistEntry{
		CodeHash:       codeHash,
		Note:           req.GetNote(),
		SourceChainID:  sourceChainID,
		SourceContract: sourceContract,
	}
	if err := s.store.AddBytecodeBlacklistEntry(ctx, record); err != nil {
		if errors.Is(err, soliditystore.ErrBytecodeBlacklistAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "bytecode blacklist entry %s already exists", codeHash.Hex())
		}
		return nil, err
	}
	created, err := s.store.GetBytecodeBlacklistEntry(ctx, codeHash)
	if err != nil {
		return nil, err
	}
	if created == nil {
		return &apiclient.AddBytecodeBlacklistEntryResponse{Item: bytecodeBlacklistEntryToAPI(record)}, nil
	}
	return &apiclient.AddBytecodeBlacklistEntryResponse{Item: bytecodeBlacklistEntryToAPI(*created)}, nil
}

func (s *Service) UpdateBytecodeBlacklistNote(ctx context.Context, req *apiclient.UpdateBytecodeBlacklistNoteRequest) (*apiclient.UpdateBytecodeBlacklistNoteResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "solidity store is required")
	}
	codeHash, err := parseHash(req.GetCodeHash())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid code hash %q", req.GetCodeHash())
	}
	if err := s.store.UpdateBytecodeBlacklistNote(ctx, codeHash, req.GetNote()); err != nil {
		if errors.Is(err, soliditystore.ErrBytecodeBlacklistNotFound) {
			return nil, status.Errorf(codes.NotFound, "bytecode blacklist entry %s not found", codeHash.Hex())
		}
		return nil, err
	}
	updated, err := s.store.GetBytecodeBlacklistEntry(ctx, codeHash)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, status.Errorf(codes.NotFound, "bytecode blacklist entry %s not found", codeHash.Hex())
	}
	return &apiclient.UpdateBytecodeBlacklistNoteResponse{Item: bytecodeBlacklistEntryToAPI(*updated)}, nil
}

func (s *Service) DeleteBytecodeBlacklist(ctx context.Context, req *apiclient.DeleteBytecodeBlacklistRequest) (*apiclient.DeleteBytecodeBlacklistResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "solidity store is required")
	}
	codeHash, err := parseHash(req.GetCodeHash())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid code hash %q", req.GetCodeHash())
	}
	if err := s.store.DeleteBytecodeBlacklist(ctx, codeHash); err != nil {
		if errors.Is(err, soliditystore.ErrBytecodeBlacklistNotFound) {
			return nil, status.Errorf(codes.NotFound, "bytecode blacklist entry %s not found", codeHash.Hex())
		}
		return nil, err
	}
	return &apiclient.DeleteBytecodeBlacklistResponse{}, nil
}

func (s *Service) validateChainID(chainID int64) error {
	if chainID <= 0 {
		return status.Error(codes.InvalidArgument, "chain_id must be positive")
	}
	if s.chainID > 0 && s.chainID != chainID {
		return status.Errorf(codes.InvalidArgument, "chain_id %d does not match service chain_id %d", chainID, s.chainID)
	}
	return nil
}

func (s *Service) resolveRequestChainID(chainID int64) int64 {
	if chainID <= 0 && s.chainID > 0 {
		return s.chainID
	}
	return chainID
}

func (s *Service) resolveContractBytecode(ctx context.Context, chainID int64, contract common.Address) (common.Hash, error) {
	code, err := s.fetchContractBytecode(ctx, contract)
	if err != nil {
		return common.Hash{}, status.Errorf(codes.Unavailable, "fetch contract bytecode for %s: %v", contract.Hex(), err)
	}
	if len(code) == 0 {
		return common.Hash{}, status.Errorf(codes.FailedPrecondition, "contract %s has empty runtime bytecode", contract.Hex())
	}
	codeHash := crypto.Keccak256Hash(code)
	if err := s.store.UpsertBytecode(ctx, codeHash, code); err != nil {
		return common.Hash{}, err
	}
	if err := s.store.UpsertContractBytecodeDeployment(ctx, soliditystore.ContractBytecodeDeployment{
		ChainID:  chainID,
		Contract: contract,
		CodeHash: codeHash,
	}); err != nil {
		return common.Hash{}, err
	}
	return codeHash, nil
}

func (s *Service) enrichBytecodeSource(ctx context.Context, contract common.Address, codeHash common.Hash) error {
	record, err := s.store.GetBytecode(ctx, codeHash)
	if err != nil {
		return err
	}
	if record == nil {
		return nil
	}
	if strings.TrimSpace(record.SourceCode) == "" && s.apiFetcher != nil {
		sourceCode, err := s.fetchSourceCode(ctx, contract)
		if err == nil && len(strings.TrimSpace(sourceCode)) > 100 {
			sourceCodeHash := crypto.Keccak256Hash([]byte(sourceCode))
			if err := s.store.UpdateBytecodeSourceCode(ctx, codeHash, sourceCode, sourceCodeHash, sourceOriginThirdPartyAPI); err != nil {
				return err
			}
			record.SourceCode = sourceCode
			record.SourceCodeHash = sourceCodeHash
		}
	}
	if strings.TrimSpace(record.SourceCode) == "" || strings.TrimSpace(record.SourceQualityReport) != "" || s.sourceQualityAnalyzer == nil {
		return nil
	}
	report, err := s.sourceQualityAnalyzer.AnalyzeContractSource(ctx, record.SourceCode)
	if err != nil {
		return nil
	}
	report = strings.TrimSpace(report)
	if report == "" {
		return nil
	}
	return s.store.UpdateBytecodeSourceQualityReport(ctx, codeHash, report, sourceOriginThirdPartyAPI)
}

func (s *Service) contractSourceInfo(ctx context.Context, chainID int64, contract common.Address, codeHash common.Hash) (*v1alpha1.ContractSourceInfo, error) {
	record, err := s.store.GetBytecode(ctx, codeHash)
	if err != nil {
		return nil, err
	}
	blacklisted, err := s.store.IsBytecodeBlacklisted(ctx, codeHash)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return &v1alpha1.ContractSourceInfo{
			Contract:              contract.Hex(),
			ChainID:               chainID,
			CodeBinHash:           codeHash.Hex(),
			IsBytecodeBlacklisted: blacklisted,
		}, nil
	}
	return bytecodeToContractSourceInfo(chainID, contract, *record, blacklisted), nil
}

func (s *Service) blacklistRequestCodeHash(ctx context.Context, req *apiclient.AddBytecodeBlacklistEntryRequest) (common.Hash, int64, common.Address, error) {
	if strings.TrimSpace(req.GetCodeHash()) != "" {
		codeHash, err := parseHash(req.GetCodeHash())
		if err != nil {
			return common.Hash{}, 0, common.Address{}, status.Errorf(codes.InvalidArgument, "invalid code hash %q", req.GetCodeHash())
		}
		return codeHash, req.GetSourceChainId(), common.HexToAddress(req.GetSourceContract()), nil
	}
	sourceChainID := s.resolveRequestChainID(req.GetSourceChainId())
	if err := s.validateChainID(sourceChainID); err != nil {
		return common.Hash{}, 0, common.Address{}, err
	}
	if !common.IsHexAddress(req.GetSourceContract()) {
		return common.Hash{}, 0, common.Address{}, status.Errorf(codes.InvalidArgument, "invalid source contract %q", req.GetSourceContract())
	}
	sourceContract := common.HexToAddress(req.GetSourceContract())
	codeHash, err := s.resolveContractBytecode(ctx, sourceChainID, sourceContract)
	if err != nil {
		return common.Hash{}, 0, common.Address{}, err
	}
	return codeHash, sourceChainID, sourceContract, nil
}

func (s *Service) fetchContractBytecode(ctx context.Context, contract common.Address) ([]byte, error) {
	if s.codeAtFunc != nil {
		return s.codeAtFunc(ctx, contract)
	}
	if s.nodeClient == nil {
		return nil, errors.New("node client is not configured")
	}
	return s.nodeClient.CodeAt(ctx, contract, nil)
}

func (s *Service) fetchSourceCode(ctx context.Context, contract common.Address) (string, error) {
	response, err := s.apiFetcher.GetSourceCode(ctx, contract.Hex())
	if err != nil {
		return "", err
	}
	if response == nil || len(response.Result) == 0 {
		return "", errors.New("etherscan getsourcecode returned empty result")
	}
	return response.Result[0].SourceCode, nil
}

func parseHash(value string) (common.Hash, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return common.Hash{}, errors.New("empty hash")
	}
	raw := common.FromHex(trimmed)
	if len(raw) != common.HashLength {
		return common.Hash{}, fmt.Errorf("hash has %d bytes, want %d", len(raw), common.HashLength)
	}
	return common.BytesToHash(raw), nil
}

func bytecodeToContractSourceInfo(chainID int64, contract common.Address, item soliditystore.Bytecode, blacklisted bool) *v1alpha1.ContractSourceInfo {
	return &v1alpha1.ContractSourceInfo{
		Contract:                     contract.Hex(),
		ChainID:                      chainID,
		CodeBinHash:                  item.CodeHash.Hex(),
		SourceCode:                   item.SourceCode,
		SourceCodeHash:               hashHex(item.SourceCodeHash),
		SourceCodeFetchedAt:          formatTime(item.SourceCodeFetchedAt),
		SourceCodeOrigin:             item.SourceCodeOrigin,
		SourceQualityReport:          item.SourceQualityReport,
		SourceQualityReportFetchedAt: formatTime(item.SourceQualityReportFetchedAt),
		SourceQualityReportOrigin:    item.SourceQualityReportOrigin,
		IsOpenSource:                 strings.TrimSpace(item.SourceCode) != "",
		IsBytecodeBlacklisted:        blacklisted,
	}
}

func bytecodeBlacklistEntryToAPI(item soliditystore.BytecodeBlacklistEntry) *v1alpha1.BytecodeBlacklistEntry {
	return &v1alpha1.BytecodeBlacklistEntry{
		CodeHash:       item.CodeHash.Hex(),
		Note:           item.Note,
		SourceChainID:  item.SourceChainID,
		SourceContract: addressHex(item.SourceContract),
		CreatedAt:      formatTime(item.CreatedAt),
	}
}

func hashHex(value common.Hash) string {
	if value == (common.Hash{}) {
		return ""
	}
	return value.Hex()
}

func addressHex(value common.Address) string {
	if value == (common.Address{}) {
		return ""
	}
	return value.Hex()
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
