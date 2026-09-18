package server

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"mime"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	operationlogrecord "github.com/useryege/athena/internal/operationlog/record"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	"github.com/useryege/athena/internal/walletsecret"
	wormtradingapiclient "github.com/useryege/athena/internal/wormtrading/apiclient"
	applicationv1alpha1 "github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

const (
	wormWalletSelectionPath           = "/api/v1/worm-trading/wallet-selection"
	wormWalletSelectionMaximumWallets = 20
	wormWalletSelectionPageSize       = int32(100)
	wormWalletSelectionMaximumBody    = int64(1024 * 1024)
)

type wormWalletSelectionInput struct {
	ExpectedRevision *int64   `json:"expectedRevision"`
	WalletIDs        *[]int64 `json:"walletIds"`
}

type wormWalletSelectionItemResponse struct {
	Ordinal  int32  `json:"ordinal"`
	WalletID int64  `json:"walletId"`
	Address  string `json:"address"`
}

type wormWalletRetirementResponse struct {
	WalletID            int64  `json:"walletId"`
	Address             string `json:"address"`
	PriorOrdinal        int32  `json:"priorOrdinal"`
	RetiredFromRevision int64  `json:"retiredFromRevision"`
	RetiredAt           int64  `json:"retiredAt"`
}

type wormWalletSelectionResponse struct {
	Configured     bool                              `json:"configured"`
	Revision       int64                             `json:"revision"`
	SelectedItems  []wormWalletSelectionItemResponse `json:"selectedItems"`
	Retirements    []wormWalletRetirementResponse    `json:"retirements"`
	UpdatedAt      int64                             `json:"updatedAt"`
	MaximumWallets int32                             `json:"maximumWallets"`
}

func registerWormWalletSelectionHandlers(mux *http.ServeMux, server *AthenaServer) {
	if mux == nil || server == nil || server.WalletClientset == nil || server.WormTradingClientset == nil {
		return
	}
	mux.Handle("GET "+wormWalletSelectionPath, traceHTTP(http.HandlerFunc(server.getWormWalletSelection)))
	mux.Handle("PUT "+wormWalletSelectionPath, traceHTTP(http.HandlerFunc(server.replaceWormWalletSelection)))
}

func (server *AthenaServer) getWormWalletSelection(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelRead)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	if err := rejectWormWalletSelectionQuery(request); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	selection, err := server.loadWormWalletSelection(ctx, credential.AccountID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormWalletSelection(w, selection)
}

func (server *AthenaServer) replaceWormWalletSelection(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	if !server.validWormConnectionOrigin(request) {
		walletsecret.WriteError(w, status.Error(codes.PermissionDenied, "a same-origin Worm Trading request is required"))
		return
	}
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelReadWrite)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	if err := rejectWormWalletSelectionQuery(request); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	input, err := decodeWormWalletSelectionInput(w, request)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	current, err := server.loadWormWalletSelection(ctx, credential.AccountID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	if current.Configured {
		if *input.ExpectedRevision != current.Revision {
			walletsecret.WriteError(w, status.Error(codes.Aborted, "Worm wallet selection revision changed"))
			return
		}
	} else if *input.ExpectedRevision != 0 {
		walletsecret.WriteError(w, status.Error(codes.Aborted, "Worm wallet selection revision changed"))
		return
	}

	wallets, err := server.listAllOwnedSolanaWallets(ctx, credential.AccountID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	selectedItems, selectedSet, err := resolveWormWalletSelectionItems(wallets, *input.WalletIDs)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	retiringWallets, err := server.resolveWormWalletRetirements(ctx, credential.AccountID, current, wallets, selectedSet)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := server.WormTradingClientset.WormTrading().ReplaceWalletSelection(ctx, &wormtradingapiclient.ReplaceWalletSelectionRequest{
		OwnerAccountId:   credential.AccountID,
		ExpectedRevision: *input.ExpectedRevision,
		SelectedItems:    selectedItems,
		RetiringWallets:  retiringWallets,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormWalletSelectionError(err))
		return
	}
	selection, err := projectWormWalletSelection(result.GetSelection())
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	currentIDs := make(map[int64]struct{}, len(current.SelectedItems))
	for _, item := range current.SelectedItems {
		currentIDs[item.WalletID] = struct{}{}
	}
	selectedIDs := make([]string, 0, len(selection.SelectedItems))
	addedCount, removedCount := 0, 0
	for _, item := range selection.SelectedItems {
		selectedIDs = append(selectedIDs, strconv.FormatInt(item.WalletID, 10))
		if _, exists := currentIDs[item.WalletID]; !exists {
			addedCount++
		}
	}
	newSelectedSet := make(map[int64]struct{}, len(selection.SelectedItems))
	for _, item := range selection.SelectedItems {
		newSelectedSet[item.WalletID] = struct{}{}
	}
	for walletID := range currentIDs {
		if _, exists := newSelectedSet[walletID]; !exists {
			removedCount++
		}
	}
	operationlogrecord.CaptureResource(ctx, "wallet_selection", credential.AccountID)
	operationlogrecord.CaptureStrings(ctx, "walletIds", selectedIDs)
	operationlogrecord.CaptureInt64(ctx, "requestedCount", int64(len(*input.WalletIDs)))
	operationlogrecord.CaptureInt64(ctx, "confirmedRevision", selection.Revision)
	operationlogrecord.CaptureInt64(ctx, "removedCount", int64(removedCount))
	operationlogrecord.CaptureInt64(ctx, "addedCount", int64(addedCount))
	operationlogrecord.Commit(ctx, "WORM_WALLET_SELECTION_REPLACE")
	writeWormWalletSelection(w, selection)
}

func (server *AthenaServer) loadWormWalletSelection(ctx context.Context, ownerAccountID string) (wormWalletSelectionResponse, error) {
	ownerAccountID, err := accountcredentials.CanonicalAccountID(ownerAccountID)
	if err != nil {
		return wormWalletSelectionResponse{}, status.Error(codes.Internal, "current account is invalid")
	}
	result, err := server.WormTradingClientset.WormTrading().GetWalletSelection(ctx, &wormtradingapiclient.GetWalletSelectionRequest{
		OwnerAccountId: ownerAccountID,
	})
	if err != nil {
		return wormWalletSelectionResponse{}, sanitizeWormWalletSelectionReadError(err)
	}
	return projectWormWalletSelection(result.GetSelection())
}

func projectWormWalletSelection(selection *wormtradingapiclient.WalletSelection) (wormWalletSelectionResponse, error) {
	response := wormWalletSelectionResponse{
		SelectedItems:  []wormWalletSelectionItemResponse{},
		Retirements:    []wormWalletRetirementResponse{},
		MaximumWallets: wormWalletSelectionMaximumWallets,
	}
	if selection == nil {
		return response, status.Error(codes.Internal, "Worm Trading returned an empty wallet selection")
	}
	if selection.GetMaximumWallets() != wormWalletSelectionMaximumWallets {
		return response, status.Error(codes.Internal, "Worm Trading returned an invalid wallet selection limit")
	}
	response.Configured = selection.GetConfigured()
	response.Revision = selection.GetRevision()
	response.UpdatedAt = selection.GetUpdatedAt()
	if !response.Configured {
		if response.Revision != 0 || response.UpdatedAt != 0 || len(selection.GetSelectedItems()) != 0 || len(selection.GetRetirements()) != 0 {
			return response, status.Error(codes.Internal, "Worm Trading returned an invalid unconfigured wallet selection")
		}
		return response, nil
	}
	if response.Revision <= 0 || response.UpdatedAt <= 0 || len(selection.GetSelectedItems()) > wormWalletSelectionMaximumWallets {
		return response, status.Error(codes.Internal, "Worm Trading returned an invalid wallet selection")
	}
	selectedIDs := make(map[int64]struct{}, len(selection.GetSelectedItems()))
	selectedAddresses := make(map[string]struct{}, len(selection.GetSelectedItems()))
	for index, item := range selection.GetSelectedItems() {
		if item == nil || item.GetOrdinal() != int32(index+1) || item.GetWalletId() <= 0 ||
			strings.TrimSpace(item.GetAddress()) == "" || item.GetAddress() != strings.TrimSpace(item.GetAddress()) {
			return response, status.Error(codes.Internal, "Worm Trading returned an invalid selected wallet")
		}
		if _, exists := selectedIDs[item.GetWalletId()]; exists {
			return response, status.Error(codes.Internal, "Worm Trading returned duplicate selected wallet IDs")
		}
		if _, exists := selectedAddresses[item.GetAddress()]; exists {
			return response, status.Error(codes.Internal, "Worm Trading returned duplicate selected wallet addresses")
		}
		selectedIDs[item.GetWalletId()] = struct{}{}
		selectedAddresses[item.GetAddress()] = struct{}{}
		response.SelectedItems = append(response.SelectedItems, wormWalletSelectionItemResponse{
			Ordinal: item.GetOrdinal(), WalletID: item.GetWalletId(), Address: item.GetAddress(),
		})
	}
	retirementIDs := make(map[int64]struct{}, len(selection.GetRetirements()))
	retirementAddresses := make(map[string]struct{}, len(selection.GetRetirements()))
	for _, retirement := range selection.GetRetirements() {
		if retirement == nil || retirement.GetWalletId() <= 0 || retirement.GetPriorOrdinal() <= 0 ||
			retirement.GetRetiredFromRevision() <= 0 || retirement.GetRetiredFromRevision() > response.Revision ||
			retirement.GetRetiredAt() <= 0 || strings.TrimSpace(retirement.GetAddress()) == "" ||
			retirement.GetAddress() != strings.TrimSpace(retirement.GetAddress()) {
			return response, status.Error(codes.Internal, "Worm Trading returned an invalid retiring wallet")
		}
		if _, selected := selectedIDs[retirement.GetWalletId()]; selected {
			return response, status.Error(codes.Internal, "Worm Trading returned a selected wallet as retiring")
		}
		if _, selected := selectedAddresses[retirement.GetAddress()]; selected {
			return response, status.Error(codes.Internal, "Worm Trading returned a selected wallet address as retiring")
		}
		if _, exists := retirementIDs[retirement.GetWalletId()]; exists {
			return response, status.Error(codes.Internal, "Worm Trading returned duplicate retiring wallet IDs")
		}
		if _, exists := retirementAddresses[retirement.GetAddress()]; exists {
			return response, status.Error(codes.Internal, "Worm Trading returned duplicate retiring wallet addresses")
		}
		retirementIDs[retirement.GetWalletId()] = struct{}{}
		retirementAddresses[retirement.GetAddress()] = struct{}{}
		response.Retirements = append(response.Retirements, wormWalletRetirementResponse{
			WalletID: retirement.GetWalletId(), Address: retirement.GetAddress(), PriorOrdinal: retirement.GetPriorOrdinal(),
			RetiredFromRevision: retirement.GetRetiredFromRevision(), RetiredAt: retirement.GetRetiredAt(),
		})
	}
	return response, nil
}

func decodeWormWalletSelectionInput(w http.ResponseWriter, request *http.Request) (wormWalletSelectionInput, error) {
	contentType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return wormWalletSelectionInput{}, status.Error(codes.InvalidArgument, "Content-Type must be application/json")
	}
	request.Body = http.MaxBytesReader(w, request.Body, wormWalletSelectionMaximumBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var input wormWalletSelectionInput
	if err := decoder.Decode(&input); err != nil {
		return wormWalletSelectionInput{}, status.Error(codes.InvalidArgument, "request body must be one valid JSON object")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return wormWalletSelectionInput{}, status.Error(codes.InvalidArgument, "request body must contain exactly one JSON object")
	}
	if input.ExpectedRevision == nil {
		return wormWalletSelectionInput{}, status.Error(codes.InvalidArgument, "expectedRevision is required")
	}
	if *input.ExpectedRevision < 0 || *input.ExpectedRevision == math.MaxInt64 {
		return wormWalletSelectionInput{}, status.Error(codes.InvalidArgument, "expectedRevision must be a non-negative incrementable integer")
	}
	if input.WalletIDs == nil {
		return wormWalletSelectionInput{}, status.Error(codes.InvalidArgument, "walletIds must be an array")
	}
	if len(*input.WalletIDs) > wormWalletSelectionMaximumWallets {
		return wormWalletSelectionInput{}, status.Errorf(codes.InvalidArgument, "walletIds must contain at most %d wallets", wormWalletSelectionMaximumWallets)
	}
	seen := make(map[int64]struct{}, len(*input.WalletIDs))
	for _, walletID := range *input.WalletIDs {
		if walletID <= 0 {
			return wormWalletSelectionInput{}, status.Error(codes.InvalidArgument, "walletIds must contain only positive integers")
		}
		if _, duplicate := seen[walletID]; duplicate {
			return wormWalletSelectionInput{}, status.Error(codes.InvalidArgument, "walletIds must not contain duplicates")
		}
		seen[walletID] = struct{}{}
	}
	return input, nil
}

func (server *AthenaServer) listAllOwnedSolanaWallets(ctx context.Context, ownerAccountID string) ([]*applicationv1alpha1.WalletItem, error) {
	ownerAccountID, err := accountcredentials.CanonicalAccountID(ownerAccountID)
	if err != nil {
		return nil, status.Error(codes.Internal, "current account is invalid")
	}
	items := make([]*applicationv1alpha1.WalletItem, 0)
	seenIDs := make(map[int64]struct{})
	seenAddresses := make(map[string]struct{})
	expectedTotal := int64(-1)
	for page := int32(1); ; page++ {
		result, err := server.WalletClientset.Wallet().ListWallets(ctx, &walletapiclient.ListWalletsRequest{
			WalletType: "SOLANA", Page: page, PageSize: wormWalletSelectionPageSize, RequesterAccountId: ownerAccountID,
		})
		if err != nil {
			return nil, sanitizeWormConnectionInventoryDependencyError(err, "Wallet")
		}
		if result == nil || result.GetPage() != page || result.GetPageSize() != wormWalletSelectionPageSize ||
			result.GetTotal() < int64(len(items)+len(result.GetItems())) || len(result.GetItems()) > int(wormWalletSelectionPageSize) {
			return nil, status.Error(codes.Internal, "Wallet returned an invalid Solana wallet page")
		}
		if expectedTotal < 0 {
			expectedTotal = result.GetTotal()
		} else if result.GetTotal() != expectedTotal {
			return nil, status.Error(codes.Aborted, "Solana wallet inventory changed while saving the Worm wallet selection")
		}
		for _, wallet := range result.GetItems() {
			if wallet == nil || wallet.ID <= 0 || wallet.WalletType != "SOLANA" || wallet.Address != strings.TrimSpace(wallet.Address) || wallet.Address == "" {
				return nil, status.Error(codes.Internal, "Wallet returned an invalid Solana wallet")
			}
			if _, duplicate := seenIDs[wallet.ID]; duplicate {
				return nil, status.Error(codes.Internal, "Wallet returned duplicate wallet IDs")
			}
			if _, duplicate := seenAddresses[wallet.Address]; duplicate {
				return nil, status.Error(codes.Internal, "Wallet returned duplicate wallet addresses")
			}
			seenIDs[wallet.ID] = struct{}{}
			seenAddresses[wallet.Address] = struct{}{}
			items = append(items, wallet)
		}
		if int64(len(items)) == expectedTotal {
			return items, nil
		}
		if len(result.GetItems()) == 0 {
			return nil, status.Error(codes.Internal, "Wallet returned an incomplete Solana wallet inventory")
		}
	}
}

func resolveWormWalletSelectionItems(
	wallets []*applicationv1alpha1.WalletItem,
	walletIDs []int64,
) ([]*wormtradingapiclient.WalletSelectionInput, map[int64]struct{}, error) {
	requested := make(map[int64]struct{}, len(walletIDs))
	for _, walletID := range walletIDs {
		requested[walletID] = struct{}{}
	}
	selected := make([]*wormtradingapiclient.WalletSelectionInput, 0, len(walletIDs))
	selectedSet := make(map[int64]struct{}, len(walletIDs))
	for _, wallet := range wallets {
		if _, wanted := requested[wallet.ID]; !wanted {
			continue
		}
		selected = append(selected, &wormtradingapiclient.WalletSelectionInput{WalletId: wallet.ID, Address: wallet.Address})
		selectedSet[wallet.ID] = struct{}{}
	}
	if len(selected) != len(walletIDs) {
		return nil, nil, status.Error(codes.FailedPrecondition, "every selected Worm wallet must be an owned Solana wallet")
	}
	return selected, selectedSet, nil
}

func (server *AthenaServer) resolveWormWalletRetirements(
	ctx context.Context,
	ownerAccountID string,
	current wormWalletSelectionResponse,
	wallets []*applicationv1alpha1.WalletItem,
	selected map[int64]struct{},
) ([]*wormtradingapiclient.WalletRetirementInput, error) {
	if current.Configured {
		result := make([]*wormtradingapiclient.WalletRetirementInput, 0)
		for _, item := range current.SelectedItems {
			if _, retained := selected[item.WalletID]; retained {
				continue
			}
			result = append(result, &wormtradingapiclient.WalletRetirementInput{
				WalletId: item.WalletID, Address: item.Address, PriorOrdinal: item.Ordinal,
			})
		}
		return result, nil
	}

	result := make([]*wormtradingapiclient.WalletRetirementInput, 0)
	for start := 0; start < len(wallets); start += int(wormWalletSelectionPageSize) {
		end := start + int(wormWalletSelectionPageSize)
		if end > len(wallets) {
			end = len(wallets)
		}
		refs := make([]*wormtradingapiclient.WalletConnectionReference, 0, end-start)
		refWallets := make([]*applicationv1alpha1.WalletItem, 0, end-start)
		refOrdinals := make([]int32, 0, end-start)
		for offset, wallet := range wallets[start:end] {
			if _, retained := selected[wallet.ID]; retained {
				continue
			}
			refs = append(refs, &wormtradingapiclient.WalletConnectionReference{WalletId: wallet.ID, Address: wallet.Address})
			refWallets = append(refWallets, wallet)
			refOrdinals = append(refOrdinals, int32(start+offset+1))
		}
		if len(refs) == 0 {
			continue
		}
		connections, err := server.WormTradingClientset.WormTrading().BatchGetWalletConnections(ctx, &wormtradingapiclient.BatchGetWalletConnectionsRequest{Refs: refs})
		if err != nil {
			return nil, sanitizeWormConnectionInventoryDependencyError(err, "Worm Trading")
		}
		if connections == nil || len(connections.GetItems()) != len(refs) {
			return nil, status.Error(codes.Internal, "Worm Trading returned an incomplete legacy connection inventory")
		}
		for index, connection := range connections.GetItems() {
			wallet := refWallets[index]
			if !validWormConnectionInventoryProjection(connection, wallet.ID, wallet.Address) {
				return nil, status.Error(codes.Internal, "Worm Trading returned an invalid legacy connection")
			}
			if connection.GetState() == "NOT_CONNECTED" && connection.GetWarningCode() == "" {
				continue
			}
			result = append(result, &wormtradingapiclient.WalletRetirementInput{
				WalletId: wallet.ID, Address: wallet.Address, PriorOrdinal: refOrdinals[index],
			})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].PriorOrdinal < result[j].PriorOrdinal })
	return result, nil
}

func rejectWormWalletSelectionQuery(request *http.Request) error {
	if request.URL.RawQuery != "" {
		return status.Error(codes.InvalidArgument, "wallet selection does not accept query parameters")
	}
	return nil
}

func sanitizeWormWalletSelectionError(err error) error {
	switch status.Code(err) {
	case codes.InvalidArgument, codes.Aborted, codes.AlreadyExists, codes.FailedPrecondition, codes.ResourceExhausted:
		return err
	case codes.Canceled:
		return err
	case codes.Unavailable, codes.DeadlineExceeded:
		return status.Error(codes.Unavailable, "Worm wallet selection is unavailable")
	default:
		return status.Error(codes.Internal, "Worm wallet selection could not be loaded")
	}
}

func sanitizeWormWalletSelectionReadError(err error) error {
	switch status.Code(err) {
	case codes.Canceled:
		return err
	case codes.FailedPrecondition, codes.Unavailable, codes.DeadlineExceeded, codes.Unauthenticated, codes.PermissionDenied, codes.Internal:
		return status.Error(codes.Unavailable, "Worm wallet selection is unavailable")
	default:
		return status.Error(codes.Internal, "Worm wallet selection could not be loaded")
	}
}

func writeWormWalletSelection(w http.ResponseWriter, response wormWalletSelectionResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(response)
}
