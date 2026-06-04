package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	"github.com/useryege/athena/internal/application/sourcequality"
	appstore "github.com/useryege/athena/internal/application/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	sourceOriginThirdPartyAPI = "third_party_api"
	defaultPage               = int64(1)
	defaultPageSize           = int64(20)
	maxPageSize               = int64(100)
)

func (s *Service) GetContractSourceInfo(ctx context.Context, req *applicationpkg.GetContractSourceInfoRequest) (*v1alpha1.ContractSourceInfo, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
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

func (s *Service) ListBytecodes(ctx context.Context, req *applicationpkg.ListBytecodesRequest) (*applicationpkg.ListBytecodesResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	page, pageSize, offset, err := normalizePagination(req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	var codeHash *common.Hash
	if strings.TrimSpace(req.GetCodeHash()) != "" {
		parsed, err := parseHash(req.GetCodeHash())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid code hash %q", req.GetCodeHash())
		}
		codeHash = &parsed
	}
	records, total, err := s.store.ListBytecodes(ctx, codeHash, pageSize, offset)
	if err != nil {
		return nil, err
	}
	items := make([]*v1alpha1.BytecodeListItem, 0, len(records))
	for _, record := range records {
		items = append(items, bytecodeListRecordToAPI(record))
	}
	return &applicationpkg.ListBytecodesResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) GetBytecode(ctx context.Context, req *applicationpkg.GetBytecodeRequest) (*v1alpha1.BytecodeDetail, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	codeHash, err := parseHash(req.GetCodeHash())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid code hash %q", req.GetCodeHash())
	}
	record, err := s.store.GetBytecodeDetail(ctx, codeHash)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, status.Errorf(codes.NotFound, "bytecode %s not found", codeHash.Hex())
	}
	if err := s.refreshBytecodeSourceQualityReport(ctx, codeHash, &record.Bytecode); err != nil {
		return nil, err
	}
	if updated, err := s.store.GetBytecodeDetail(ctx, codeHash); err != nil {
		return nil, err
	} else if updated != nil {
		record = updated
	}
	return bytecodeDetailRecordToAPI(*record), nil
}

func (s *Service) ListBytecodeDeployments(ctx context.Context, req *applicationpkg.ListBytecodeDeploymentsRequest) (*applicationpkg.ListBytecodeDeploymentsResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	codeHash, err := parseHash(req.GetCodeHash())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid code hash %q", req.GetCodeHash())
	}
	page, pageSize, offset, err := normalizePagination(req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	var contract *common.Address
	if strings.TrimSpace(req.GetContract()) != "" {
		if !common.IsHexAddress(req.GetContract()) {
			return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
		}
		parsedContract := common.HexToAddress(req.GetContract())
		contract = &parsedContract
	}
	if req.GetChainId() < 0 {
		return nil, status.Error(codes.InvalidArgument, "chain_id must be non-negative")
	}
	records, total, err := s.store.ListBytecodeDeployments(ctx, codeHash, req.GetChainId(), contract, pageSize, offset)
	if err != nil {
		return nil, err
	}
	items := make([]*v1alpha1.BytecodeDeployment, 0, len(records))
	for _, record := range records {
		items = append(items, bytecodeDeploymentRecordToAPI(record))
	}
	return &applicationpkg.ListBytecodeDeploymentsResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) ListBytecodeBlacklistEntries(ctx context.Context, _ *applicationpkg.ListBytecodeBlacklistEntriesRequest) (*applicationpkg.ListBytecodeBlacklistEntriesResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	records, err := s.store.ListBytecodeBlacklistEntries(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*v1alpha1.BytecodeBlacklistEntry, 0, len(records))
	for _, record := range records {
		items = append(items, bytecodeBlacklistEntryToAPI(record))
	}
	return &applicationpkg.ListBytecodeBlacklistEntriesResponse{Items: items}, nil
}

func (s *Service) AddBytecodeBlacklistEntry(ctx context.Context, req *applicationpkg.AddBytecodeBlacklistEntryRequest) (*applicationpkg.AddBytecodeBlacklistEntryResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	codeHash, sourceChainID, sourceContract, err := s.blacklistRequestCodeHash(ctx, req)
	if err != nil {
		return nil, err
	}
	record := appstore.BytecodeBlacklistEntry{
		CodeHash:       codeHash,
		Note:           req.GetNote(),
		SourceChainID:  sourceChainID,
		SourceContract: sourceContract,
	}
	if err := s.store.AddBytecodeBlacklistEntry(ctx, record); err != nil {
		if errors.Is(err, appstore.ErrBytecodeBlacklistAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "bytecode blacklist entry %s already exists", codeHash.Hex())
		}
		return nil, err
	}
	created, err := s.store.GetBytecodeBlacklistEntry(ctx, codeHash)
	if err != nil {
		return nil, err
	}
	if created == nil {
		return &applicationpkg.AddBytecodeBlacklistEntryResponse{Item: bytecodeBlacklistEntryToAPI(record)}, nil
	}
	return &applicationpkg.AddBytecodeBlacklistEntryResponse{Item: bytecodeBlacklistEntryToAPI(*created)}, nil
}

func (s *Service) UpdateBytecodeBlacklistNote(ctx context.Context, req *applicationpkg.UpdateBytecodeBlacklistNoteRequest) (*applicationpkg.UpdateBytecodeBlacklistNoteResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	codeHash, err := parseHash(req.GetCodeHash())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid code hash %q", req.GetCodeHash())
	}
	if err := s.store.UpdateBytecodeBlacklistNote(ctx, codeHash, req.GetNote()); err != nil {
		if errors.Is(err, appstore.ErrBytecodeBlacklistNotFound) {
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
	return &applicationpkg.UpdateBytecodeBlacklistNoteResponse{Item: bytecodeBlacklistEntryToAPI(*updated)}, nil
}

func (s *Service) DeleteBytecodeBlacklist(ctx context.Context, req *applicationpkg.DeleteBytecodeBlacklistRequest) (*applicationpkg.DeleteBytecodeBlacklistResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	codeHash, err := parseHash(req.GetCodeHash())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid code hash %q", req.GetCodeHash())
	}
	if err := s.store.DeleteBytecodeBlacklist(ctx, codeHash); err != nil {
		if errors.Is(err, appstore.ErrBytecodeBlacklistNotFound) {
			return nil, status.Errorf(codes.NotFound, "bytecode blacklist entry %s not found", codeHash.Hex())
		}
		return nil, err
	}
	return &applicationpkg.DeleteBytecodeBlacklistResponse{}, nil
}

func (s *Service) ListSourceQualityPrompts(ctx context.Context, _ *applicationpkg.ListSourceQualityPromptsRequest) (*applicationpkg.ListSourceQualityPromptsResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	records, err := s.store.ListSourceQualityPrompts(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*v1alpha1.SourceQualityPrompt, 0, len(records))
	for _, record := range records {
		items = append(items, sourceQualityPromptToAPI(record))
	}
	return &applicationpkg.ListSourceQualityPromptsResponse{Items: items}, nil
}

func (s *Service) GetSourceQualityPrompt(ctx context.Context, req *applicationpkg.GetSourceQualityPromptRequest) (*v1alpha1.SourceQualityPrompt, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be positive")
	}
	record, err := s.store.GetSourceQualityPrompt(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, status.Errorf(codes.NotFound, "source quality prompt %d not found", req.GetId())
	}
	return sourceQualityPromptToAPI(*record), nil
}

func (s *Service) CreateSourceQualityPrompt(ctx context.Context, req *applicationpkg.CreateSourceQualityPromptRequest) (*applicationpkg.CreateSourceQualityPromptResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	if err := validateSourceQualityPromptInput(req.GetName(), req.GetSystemPrompt()); err != nil {
		return nil, err
	}
	record, err := s.store.CreateSourceQualityPrompt(ctx, req.GetName(), req.GetSystemPrompt())
	if err != nil {
		return nil, err
	}
	return &applicationpkg.CreateSourceQualityPromptResponse{Item: sourceQualityPromptToAPI(*record)}, nil
}

func (s *Service) UpdateSourceQualityPrompt(ctx context.Context, req *applicationpkg.UpdateSourceQualityPromptRequest) (*applicationpkg.UpdateSourceQualityPromptResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be positive")
	}
	if err := validateSourceQualityPromptInput(req.GetName(), req.GetSystemPrompt()); err != nil {
		return nil, err
	}
	record, err := s.store.UpdateSourceQualityPrompt(ctx, req.GetId(), req.GetName(), req.GetSystemPrompt())
	if err != nil {
		if errors.Is(err, appstore.ErrSourceQualityPromptNotFound) {
			return nil, status.Errorf(codes.NotFound, "source quality prompt %d not found", req.GetId())
		}
		return nil, err
	}
	return &applicationpkg.UpdateSourceQualityPromptResponse{Item: sourceQualityPromptToAPI(*record)}, nil
}

func (s *Service) ActivateSourceQualityPrompt(ctx context.Context, req *applicationpkg.ActivateSourceQualityPromptRequest) (*applicationpkg.ActivateSourceQualityPromptResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be positive")
	}
	record, err := s.store.ActivateSourceQualityPrompt(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, appstore.ErrSourceQualityPromptNotFound) {
			return nil, status.Errorf(codes.NotFound, "source quality prompt %d not found", req.GetId())
		}
		return nil, err
	}
	return &applicationpkg.ActivateSourceQualityPromptResponse{Item: sourceQualityPromptToAPI(*record)}, nil
}

func (s *Service) DeleteSourceQualityPrompt(ctx context.Context, req *applicationpkg.DeleteSourceQualityPromptRequest) (*applicationpkg.DeleteSourceQualityPromptResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be positive")
	}
	if err := s.store.DeleteSourceQualityPrompt(ctx, req.GetId()); err != nil {
		switch {
		case errors.Is(err, appstore.ErrSourceQualityPromptNotFound):
			return nil, status.Errorf(codes.NotFound, "source quality prompt %d not found", req.GetId())
		case errors.Is(err, appstore.ErrSourceQualityPromptActiveDelete):
			return nil, status.Error(codes.FailedPrecondition, "active source quality prompt cannot be deleted")
		default:
			return nil, err
		}
	}
	return &applicationpkg.DeleteSourceQualityPromptResponse{}, nil
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
	if err := s.store.UpsertContractBytecodeDeployment(ctx, appstore.ContractBytecodeDeployment{
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
	return s.refreshBytecodeSourceQualityReport(ctx, codeHash, record)
}

func (s *Service) refreshBytecodeSourceQualityReport(ctx context.Context, codeHash common.Hash, record *appstore.Bytecode) error {
	if record == nil || strings.TrimSpace(record.SourceCode) == "" || s.sourceQualityAnalyzer == nil {
		return nil
	}
	prompt, err := s.currentSourceQualityPrompt(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(record.SourceQualityReport) != "" && record.SourceQualityPromptVersion >= prompt.Version {
		return nil
	}
	report, err := s.sourceQualityAnalyzer.AnalyzeContractSource(ctx, prompt.SystemPrompt, record.SourceCode)
	if err != nil {
		return nil
	}
	report = strings.TrimSpace(report)
	if report == "" {
		return nil
	}
	if err := s.store.UpdateBytecodeSourceQualityReport(ctx, codeHash, report, sourceOriginThirdPartyAPI, prompt.Version); err != nil {
		return err
	}
	record.SourceQualityReport = report
	record.SourceQualityReportOrigin = sourceOriginThirdPartyAPI
	record.SourceQualityPromptVersion = prompt.Version
	record.SourceQualityReportFetchedAt = time.Now().UTC()
	return nil
}

func (s *Service) currentSourceQualityPrompt(ctx context.Context) (*appstore.SourceQualityPrompt, error) {
	prompt, err := s.store.GetActiveSourceQualityPrompt(ctx)
	if err != nil {
		return nil, err
	}
	if prompt != nil {
		return prompt, nil
	}
	return &appstore.SourceQualityPrompt{
		Name:         "Default Solidity Source Quality Prompt",
		SystemPrompt: sourcequality.DefaultSystemPrompt,
	}, nil
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

func (s *Service) blacklistRequestCodeHash(ctx context.Context, req *applicationpkg.AddBytecodeBlacklistEntryRequest) (common.Hash, int64, common.Address, error) {
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

func normalizePagination(page, pageSize int64) (int64, int64, int64, error) {
	if page < 0 {
		return 0, 0, 0, status.Error(codes.InvalidArgument, "page must be non-negative")
	}
	if pageSize < 0 {
		return 0, 0, 0, status.Error(codes.InvalidArgument, "page_size must be non-negative")
	}
	if page == 0 {
		page = defaultPage
	}
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize, (page - 1) * pageSize, nil
}

func bytecodeToContractSourceInfo(chainID int64, contract common.Address, item appstore.Bytecode, blacklisted bool) *v1alpha1.ContractSourceInfo {
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
		SourceQualityPromptVersion:   item.SourceQualityPromptVersion,
	}
}

func bytecodeListRecordToAPI(item appstore.BytecodeListRecord) *v1alpha1.BytecodeListItem {
	return &v1alpha1.BytecodeListItem{
		CodeHash:              item.CodeHash.Hex(),
		RuntimeBytecodeSize:   item.RuntimeBytecodeSize,
		DeploymentCount:       item.DeploymentCount,
		IsOpenSource:          item.IsOpenSource,
		IsBytecodeBlacklisted: item.IsBytecodeBlacklisted,
		CreatedAt:             formatTime(item.CreatedAt),
		UpdatedAt:             formatTime(item.UpdatedAt),
	}
}

func bytecodeDetailRecordToAPI(item appstore.BytecodeDetailRecord) *v1alpha1.BytecodeDetail {
	return &v1alpha1.BytecodeDetail{
		CodeHash:                     item.CodeHash.Hex(),
		RuntimeBytecodeSize:          item.RuntimeBytecodeSize,
		DeploymentCount:              item.DeploymentCount,
		IsOpenSource:                 strings.TrimSpace(item.SourceCode) != "",
		IsBytecodeBlacklisted:        item.IsBytecodeBlacklisted,
		CreatedAt:                    formatTime(item.CreatedAt),
		UpdatedAt:                    formatTime(item.UpdatedAt),
		RuntimeBytecode:              hexutil.Encode(item.RuntimeBytecode),
		SourceCode:                   item.SourceCode,
		SourceCodeHash:               hashHex(item.SourceCodeHash),
		SourceCodeFetchedAt:          formatTime(item.SourceCodeFetchedAt),
		SourceCodeOrigin:             item.SourceCodeOrigin,
		SourceQualityReport:          item.SourceQualityReport,
		SourceQualityReportFetchedAt: formatTime(item.SourceQualityReportFetchedAt),
		SourceQualityReportOrigin:    item.SourceQualityReportOrigin,
		SourceQualityPromptVersion:   item.SourceQualityPromptVersion,
	}
}

func bytecodeDeploymentRecordToAPI(item appstore.BytecodeDeploymentRecord) *v1alpha1.BytecodeDeployment {
	return &v1alpha1.BytecodeDeployment{
		ChainID:     item.ChainID,
		Contract:    item.Contract.Hex(),
		FirstSeenAt: formatTime(item.FirstSeenAt),
		UpdatedAt:   formatTime(item.UpdatedAt),
	}
}

func bytecodeBlacklistEntryToAPI(item appstore.BytecodeBlacklistEntry) *v1alpha1.BytecodeBlacklistEntry {
	return &v1alpha1.BytecodeBlacklistEntry{
		CodeHash:       item.CodeHash.Hex(),
		Note:           item.Note,
		SourceChainID:  item.SourceChainID,
		SourceContract: addressHex(item.SourceContract),
		CreatedAt:      formatTime(item.CreatedAt),
	}
}

func sourceQualityPromptToAPI(item appstore.SourceQualityPrompt) *v1alpha1.SourceQualityPrompt {
	return &v1alpha1.SourceQualityPrompt{
		ID:           item.ID,
		Version:      item.Version,
		Name:         item.Name,
		SystemPrompt: item.SystemPrompt,
		IsActive:     item.IsActive,
		CreatedAt:    formatTime(item.CreatedAt),
		UpdatedAt:    formatTime(item.UpdatedAt),
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

func validateSourceQualityPromptInput(name, systemPrompt string) error {
	if strings.TrimSpace(name) == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	if strings.TrimSpace(systemPrompt) == "" {
		return status.Error(codes.InvalidArgument, "system_prompt is required")
	}
	return nil
}
