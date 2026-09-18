package wallet

import (
	"context"
	"strconv"
	"strings"

	operationlogrecord "github.com/useryege/athena/internal/operationlog/record"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	walletpkg "github.com/useryege/athena/pkg/apiclient/wallet"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	walletpkg.UnimplementedWalletServiceServer
	walletClientSet     walletapiclient.Clientset
	avatarObjectCleanup func(string)
}

func NewServer(walletClientSet walletapiclient.Clientset, avatarObjectCleanup func(string)) *Server {
	return &Server{walletClientSet: walletClientSet, avatarObjectCleanup: avatarObjectCleanup}
}

func (s *Server) GetWalletStatus(ctx context.Context, _ *walletpkg.GetWalletStatusRequest) (*v1alpha1.WalletStatus, error) {
	return s.walletClientSet.Wallet().GetWalletStatus(ctx, &walletapiclient.GetWalletStatusRequest{})
}

func (s *Server) ListWallets(ctx context.Context, req *walletpkg.ListWalletsRequest) (*walletpkg.ListWalletsResponse, error) {
	accountID, err := requesterAccountID(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.walletClientSet.Wallet().ListWallets(ctx, &walletapiclient.ListWalletsRequest{
		WalletType:         req.GetWalletType(),
		Query:              req.GetQuery(),
		Page:               req.GetPage(),
		PageSize:           req.GetPageSize(),
		RequesterAccountId: accountID,
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.ListWalletsResponse{
		Items:    resp.GetItems(),
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
	}, nil
}

func (s *Server) GetWallet(ctx context.Context, req *walletpkg.GetWalletRequest) (*walletpkg.GetWalletResponse, error) {
	accountID, err := requesterAccountID(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.walletClientSet.Wallet().GetWallet(ctx, &walletapiclient.GetWalletRequest{
		Id:                 req.GetId(),
		RequesterAccountId: accountID,
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.GetWalletResponse{Item: resp.GetItem()}, nil
}

func (s *Server) BatchCreateWallets(ctx context.Context, req *walletpkg.BatchCreateWalletsRequest) (*walletpkg.BatchCreateWalletsResponse, error) {
	accountID, err := requesterAccountID(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.walletClientSet.Wallet().BatchCreateWallets(ctx, &walletapiclient.BatchCreateWalletsRequest{
		WalletType:         req.GetWalletType(),
		Count:              req.GetCount(),
		Remark:             req.GetRemark(),
		AvatarPresetId:     req.GetAvatarPresetId(),
		RequesterAccountId: accountID,
	})
	if err != nil {
		return nil, err
	}
	walletIDs := make([]string, 0, len(resp.GetResults()))
	for _, result := range resp.GetResults() {
		if result.GetItem() != nil {
			walletIDs = append(walletIDs, strconv.FormatInt(result.GetItem().ID, 10))
		}
	}
	operationlogrecord.CaptureString(ctx, "walletType", normalizedWalletTypeForLog(req.GetWalletType()))
	operationlogrecord.CaptureInt64(ctx, "requestedCount", int64(req.GetCount()))
	operationlogrecord.CaptureInt64(ctx, "confirmedCount", int64(len(walletIDs)))
	operationlogrecord.CaptureStrings(ctx, "walletIds", walletIDs)
	operationlogrecord.Commit(ctx, "WALLET_BATCH_CREATE")
	results := make([]*walletpkg.BatchCreateWalletResult, 0, len(resp.GetResults()))
	for _, result := range resp.GetResults() {
		results = append(results, &walletpkg.BatchCreateWalletResult{
			Item:       result.GetItem(),
			PrivateKey: result.GetPrivateKey(),
		})
	}
	return &walletpkg.BatchCreateWalletsResponse{Results: results}, nil
}

func (s *Server) BatchImportWallets(ctx context.Context, req *walletpkg.BatchImportWalletsRequest) (*walletpkg.BatchImportWalletsResponse, error) {
	accountID, err := requesterAccountID(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.walletClientSet.Wallet().BatchImportWallets(ctx, &walletapiclient.BatchImportWalletsRequest{
		WalletType:         req.GetWalletType(),
		PrivateKeys:        req.GetPrivateKeys(),
		Remark:             req.GetRemark(),
		AvatarPresetId:     req.GetAvatarPresetId(),
		RequesterAccountId: accountID,
	})
	if err != nil {
		return nil, err
	}
	walletIDs := make([]string, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		if item != nil {
			walletIDs = append(walletIDs, strconv.FormatInt(item.ID, 10))
		}
	}
	operationlogrecord.CaptureString(ctx, "walletType", normalizedWalletTypeForLog(req.GetWalletType()))
	operationlogrecord.CaptureInt64(ctx, "requestedCount", int64(len(req.GetPrivateKeys())))
	operationlogrecord.CaptureInt64(ctx, "confirmedCount", int64(len(walletIDs)))
	operationlogrecord.CaptureStrings(ctx, "walletIds", walletIDs)
	operationlogrecord.Commit(ctx, "WALLET_BATCH_IMPORT")
	return &walletpkg.BatchImportWalletsResponse{Items: resp.GetItems()}, nil
}

func (s *Server) UpdateWalletRemark(ctx context.Context, req *walletpkg.UpdateWalletRemarkRequest) (*walletpkg.UpdateWalletRemarkResponse, error) {
	accountID, err := requesterAccountID(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.walletClientSet.Wallet().UpdateWalletRemark(ctx, &walletapiclient.UpdateWalletRemarkRequest{
		Id:                 req.GetId(),
		Remark:             req.GetRemark(),
		ExpectedRevision:   req.GetExpectedRevision(),
		RequesterAccountId: accountID,
	})
	if err != nil {
		return nil, err
	}
	if item := resp.GetItem(); item != nil {
		operationlogrecord.CaptureResource(ctx, "wallet", strconv.FormatInt(item.ID, 10))
		operationlogrecord.CaptureUint64(ctx, "expectedRevision", req.GetExpectedRevision())
		operationlogrecord.CaptureUint64(ctx, "confirmedRevision", item.Revision)
		operationlogrecord.CaptureStrings(ctx, "changedFields", []string{"remark"})
		operationlogrecord.Commit(ctx, "WALLET_REMARK_UPDATE")
	}
	return &walletpkg.UpdateWalletRemarkResponse{Item: resp.GetItem()}, nil
}

func (s *Server) UpdateWalletAvatarPreset(ctx context.Context, req *walletpkg.UpdateWalletAvatarPresetRequest) (*walletpkg.UpdateWalletAvatarPresetResponse, error) {
	accountID, err := requesterAccountID(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.walletClientSet.Wallet().UpdateWalletAvatarPreset(ctx, &walletapiclient.UpdateWalletAvatarPresetRequest{
		Id:                 req.GetId(),
		AvatarPresetId:     req.GetAvatarPresetId(),
		ExpectedRevision:   req.GetExpectedRevision(),
		RequesterAccountId: accountID,
	})
	if err != nil {
		return nil, err
	}
	if previousObjectKey := resp.GetPreviousAvatarObjectKey(); previousObjectKey != "" && s.avatarObjectCleanup != nil {
		s.avatarObjectCleanup(previousObjectKey)
	}
	if item := resp.GetItem(); item != nil {
		operationlogrecord.CaptureResource(ctx, "wallet", strconv.FormatInt(item.ID, 10))
		operationlogrecord.CaptureString(ctx, "avatarPresetId", req.GetAvatarPresetId())
		operationlogrecord.CaptureUint64(ctx, "expectedRevision", req.GetExpectedRevision())
		operationlogrecord.CaptureUint64(ctx, "confirmedRevision", item.Revision)
		operationlogrecord.Commit(ctx, "WALLET_AVATAR_PRESET_UPDATE")
	}
	return &walletpkg.UpdateWalletAvatarPresetResponse{Item: resp.GetItem()}, nil
}

func requesterAccountID(ctx context.Context) (string, error) {
	accountID := session.AccountID(ctx)
	if accountID == "" {
		return "", status.Error(codes.Unauthenticated, "authenticated account ID is missing")
	}
	return accountID, nil
}

func normalizedWalletTypeForLog(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
