package server

import (
	"context"

	"encoding/base64"
	"encoding/json"
	"github.com/useryege/athena/internal/moduleaccess"
	"github.com/useryege/athena/internal/walletsecret"
	grpcutil "github.com/useryege/athena/util/grpc"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"reflect"
	"strings"
)

const (
	coreAdmission     moduleaccess.Key = "core"
	deferredAdmission moduleaccess.Key = "deferred"
)

// Explicit classifications are independent from account authorization. New RPCs
// must be assigned deliberately; the registered-descriptor test guards drift.
var grpcAdmission = map[string]moduleaccess.Key{
	"/account.AccountService/CreateToken":                                   coreAdmission,
	"/account.AccountService/DeleteToken":                                   coreAdmission,
	"/account.AccountService/GetAccount":                                    coreAdmission,
	"/account.AccountService/ListAccounts":                                  coreAdmission,
	"/account.AccountService/ListTokens":                                    coreAdmission,
	"/account.AccountService/UpdateAccountAccess":                           coreAdmission,
	"/account.AccountService/UpdateAccountProfile":                          coreAdmission,
	"/account.AccountService/UpdateAccountTier":                             coreAdmission,
	"/appbootstrap.AppBootstrapService/GetAppBootstrap":                     coreAdmission,
	"/grpc.health.v1.Health/Check":                                          coreAdmission,
	"/grpc.health.v1.Health/List":                                           coreAdmission,
	"/grpc.health.v1.Health/Watch":                                          coreAdmission,
	"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo":             coreAdmission,
	"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo":        coreAdmission,
	"/managedoo.ManagedOOService/GetManagedOOStatus":                        moduleaccess.ManagedOO,
	"/managedoo.ManagedOOService/ListManagedOODisputes":                     moduleaccess.ManagedOO,
	"/managedoo.ManagedOOService/ListManagedOOProposals":                    moduleaccess.ManagedOO,
	"/managedoo.ManagedOOService/ScanManagedOOBlock":                        moduleaccess.ManagedOO,
	"/marketradar.MarketRadarService/GetMarketRadarStatus":                  moduleaccess.MarketRadar,
	"/marketradar.MarketRadarService/ListHotMarkets":                        moduleaccess.MarketRadar,
	"/marketradar.MarketRadarService/ListMarketMovers":                      moduleaccess.MarketRadar,
	"/marketradar.MarketRadarService/ListRealtimeMarkets":                   moduleaccess.MarketRadar,
	"/moduleaccess.ModuleAccessService/ListModuleAccessSettings":            coreAdmission,
	"/moduleaccess.ModuleAccessService/ListModuleAccessStates":              coreAdmission,
	"/moduleaccess.ModuleAccessService/UpdateModuleAccessSetting":           coreAdmission,
	"/operationlog.OperationLogService/ListOperationLogs":                   coreAdmission,
	"/operationlog.OperationLogService/GetOperationLog":                     coreAdmission,
	"/operationlog.OperationLogService/GetOperationLogRuntimeStatus":        coreAdmission,
	"/operationlog.OperationLogService/GetOperationLogCaptureStatus":        coreAdmission,
	"/operationlog.OperationLogService/ListOperationLogActions":             coreAdmission,
	"/notification.NotificationService/CreateTelegramBindingAttempt":        coreAdmission,
	"/notification.NotificationService/DeleteTelegramBinding":               coreAdmission,
	"/notification.NotificationService/DeleteTelegramBindingAttempt":        coreAdmission,
	"/notification.NotificationService/GetNotificationRuntimeStatus":        coreAdmission,
	"/notification.NotificationService/GetSystemNotificationDelivery":       coreAdmission,
	"/notification.NotificationService/GetTelegramBinding":                  coreAdmission,
	"/notification.NotificationService/ListSystemNotificationDeliveries":    coreAdmission,
	"/notification.NotificationService/SendSystemNotificationTest":          coreAdmission,
	"/profitsharing.ProfitSharingService/CloseBallot":                       moduleaccess.ProfitSharing,
	"/profitsharing.ProfitSharingService/CreateRound":                       moduleaccess.ProfitSharing,
	"/profitsharing.ProfitSharingService/GetRound":                          moduleaccess.ProfitSharing,
	"/profitsharing.ProfitSharingService/ListRounds":                        moduleaccess.ProfitSharing,
	"/profitsharing.ProfitSharingService/OpenRound":                         moduleaccess.ProfitSharing,
	"/profitsharing.ProfitSharingService/PublishRound":                      moduleaccess.ProfitSharing,
	"/profitsharing.ProfitSharingService/ReopenProposal":                    moduleaccess.ProfitSharing,
	"/profitsharing.ProfitSharingService/SubmitProposal":                    moduleaccess.ProfitSharing,
	"/profitsharing.ProfitSharingService/SubmitVote":                        moduleaccess.ProfitSharing,
	"/profitsharing.ProfitSharingService/UpdateProposal":                    moduleaccess.ProfitSharing,
	"/profitsharing.ProfitSharingService/UpdateRound":                       moduleaccess.ProfitSharing,
	"/servicestatus.ServiceStatusService/GetEtherscanGatewayProbeRun":       coreAdmission,
	"/servicestatus.ServiceStatusService/GetLatestEtherscanGatewayProbeRun": coreAdmission,
	"/servicestatus.ServiceStatusService/ListEtherscanGatewayStatuses":      coreAdmission,
	"/servicestatus.ServiceStatusService/ListServiceStatuses":               coreAdmission,
	"/servicestatus.ServiceStatusService/RunEtherscanGatewayProbe":          coreAdmission,
	"/session.SessionService/GetUserInfo":                                   coreAdmission,
	"/solana.SolanaService/GetDiscoveryStatus":                              moduleaccess.Solana,
	"/solana.SolanaService/ListProjects":                                    moduleaccess.Solana,
	"/tokenapi.TokenCatalogService/GetContractCode":                         deferredAdmission,
	"/tokenapi.TokenCatalogService/GetProjectDetail":                        deferredAdmission,
	"/tokenapi.TokenCatalogService/GetProjectProfile":                       deferredAdmission,
	"/tokenapi.TokenCatalogService/ListContractCodes":                       deferredAdmission,
	"/tokenapi.TokenCatalogService/ListProjectWalletNormalTransactions":     deferredAdmission,
	"/tokenapi.TokenCatalogService/ListProjects":                            deferredAdmission,
	"/tokenapi.TokenCollectionService/GetCollectionTask":                    deferredAdmission,
	"/tokenapi.TokenCollectionService/ListCollectionTasks":                  deferredAdmission,
	"/tokenapi.TokenOperationsService/GetChainCheckpoint":                   deferredAdmission,
	"/tokenapi.TokenOperationsService/GetChainProcessingSummary":            deferredAdmission,
	"/tokenapi.TokenOperationsService/GetRuntimeConfiguration":              deferredAdmission,
	"/tokenapi.TokenOperationsService/ListChainCheckpoints":                 deferredAdmission,
	"/tokenapi.TokenOperationsService/ListChainProcessingAttempts":          deferredAdmission,
	"/tokenapi.TokenOperationsService/ListNodeStatuses":                     deferredAdmission,
	"/tokenapi.TokenOperationsService/UpdateChainCheckpoint":                deferredAdmission,
	"/tokenapi.TokenPolicyService/CreateContractCodeBlocklistEntry":         deferredAdmission,
	"/tokenapi.TokenPolicyService/CreateWalletBlocklistEntry":               deferredAdmission,
	"/tokenapi.TokenPolicyService/DeleteContractCodeBlocklistEntry":         deferredAdmission,
	"/tokenapi.TokenPolicyService/DeleteWalletBlocklistEntry":               deferredAdmission,
	"/tokenapi.TokenPolicyService/GetContractCodeBlocklistEntry":            deferredAdmission,
	"/tokenapi.TokenPolicyService/GetWalletBlocklistEntry":                  deferredAdmission,
	"/tokenapi.TokenPolicyService/ListContractCodeBlocklistEntries":         deferredAdmission,
	"/tokenapi.TokenPolicyService/ListWalletBlocklistEntries":               deferredAdmission,
	"/tokenapi.TokenPolicyService/UpdateContractCodeBlocklistEntry":         deferredAdmission,
	"/tokenapi.TokenPolicyService/UpdateWalletBlocklistEntry":               deferredAdmission,
	"/tradersync.TraderSyncService/CancelSubscription":                      moduleaccess.TraderSync,
	"/tradersync.TraderSyncService/CreateSubscription":                      moduleaccess.TraderSync,
	"/tradersync.TraderSyncService/GetActivity":                             moduleaccess.TraderSync,
	"/tradersync.TraderSyncService/GetSubscription":                         moduleaccess.TraderSync,
	"/tradersync.TraderSyncService/GetSubscriptionSummary":                  moduleaccess.TraderSync,
	"/tradersync.TraderSyncService/GetSummaryBatch":                         moduleaccess.TraderSync,
	"/tradersync.TraderSyncService/GetTraderSyncRuntimeStatus":              coreAdmission,
	"/tradersync.TraderSyncService/ListActivities":                          moduleaccess.TraderSync,
	"/tradersync.TraderSyncService/ListSubscriptionHistory":                 moduleaccess.TraderSync,
	"/tradersync.TraderSyncService/ListSubscriptionSummaries":               moduleaccess.TraderSync,
	"/tradersync.TraderSyncService/ListSubscriptions":                       moduleaccess.TraderSync,
	"/tradersync.TraderSyncService/ListSummaryParts":                        moduleaccess.TraderSync,
	"/tradersync.TraderSyncService/PauseSubscription":                       moduleaccess.TraderSync,
	"/tradersync.TraderSyncService/ResolveTarget":                           moduleaccess.TraderSync,
	"/tradersync.TraderSyncService/ResumeSubscription":                      moduleaccess.TraderSync,
	"/tradersync.TraderSyncService/UpdateTargetNote":                        moduleaccess.TraderSync,
	"/version.VersionService/Version":                                       coreAdmission,
	"/wallet.WalletService/BatchCreateWallets":                              coreAdmission,
	"/wallet.WalletService/BatchImportWallets":                              coreAdmission,
	"/wallet.WalletService/GetWallet":                                       coreAdmission,
	"/wallet.WalletService/GetWalletStatus":                                 coreAdmission,
	"/wallet.WalletService/ListWallets":                                     coreAdmission,
	"/wallet.WalletService/UpdateWalletAvatarPreset":                        coreAdmission,
	"/wallet.WalletService/UpdateWalletRemark":                              coreAdmission,
	"/wormtrading.WormTradingService/GetWormTradingStatus":                  moduleaccess.Worm,
	"/wormtrading.WormTradingService/ListWalletBalances":                    moduleaccess.Worm,
	"/wormtrading.WormTradingService/ListWalletTradingActivity":             moduleaccess.Worm,
}

func (server *AthenaServer) checkModuleAdmission(ctx context.Context, method string) error {
	key, ok := grpcAdmission[method]
	if !ok {
		return status.Error(codes.Internal, "module access classification is missing for "+method)
	}
	if key == coreAdmission || key == deferredAdmission {
		return nil
	}
	return moduleaccess.Check(ctx, server.moduleAccessStore, key)
}

// Forward only the admission metadata using native HTTP header names. Other
// gateway metadata keeps its existing grpc-gateway prefix.
func moduleAccessOutgoingHeader(key string) (string, bool) {
	switch strings.ToLower(key) {
	case "x-athena-error-reason", "cache-control", "pragma":
		return key, true
	}
	return "Grpc-Metadata-" + key, true
}
func moduleAdmissionHeaders(ctx context.Context, err error) {
	reason := walletsecret.Reason(err)
	if reason == moduleaccess.ClosedReason || reason == moduleaccess.UnavailableReason {
		_ = grpc.SetHeader(ctx, metadata.Pairs("x-athena-error-reason", reason, "cache-control", "no-store, private", "pragma", "no-cache"))
	}
}

// The established gateway marshaler uses encoding/json, which serializes Any
// as base64. Expand module ErrorInfo so HTTP clients can read module_key while
// retaining the existing error/code/message envelope and all other errors.
type moduleAccessJSONMarshaler struct{ grpcutil.JSONMarshaler }

func (m *moduleAccessJSONMarshaler) Marshal(value any) ([]byte, error) {
	raw, err := m.JSONMarshaler.Marshal(value)
	if err != nil {
		return nil, err
	}
	if isOperationLogMessage(value) {
		raw = flattenOperationLogNullables(raw)
	}
	if _, ok := value.(interface{ GetDetails() []*anypb.Any }); !ok {
		return raw, nil
	}
	var body map[string]json.RawMessage
	if json.Unmarshal(raw, &body) != nil {
		return raw, nil
	}
	var details []json.RawMessage
	if json.Unmarshal(body["details"], &details) != nil {
		return raw, nil
	}
	for i, detail := range details {
		var wire struct {
			TypeURL string `json:"type_url"`
			Value   string `json:"value"`
		}
		if json.Unmarshal(detail, &wire) != nil || wire.TypeURL != "type.googleapis.com/google.rpc.ErrorInfo" {
			continue
		}
		data, e := base64.StdEncoding.DecodeString(wire.Value)
		if e != nil {
			continue
		}
		info := new(errdetails.ErrorInfo)
		if proto.Unmarshal(data, info) != nil || info.Domain != moduleaccess.Domain {
			continue
		}
		details[i], _ = json.Marshal(map[string]any{"@type": wire.TypeURL, "domain": info.Domain, "reason": info.Reason, "metadata": info.Metadata})
		body["reason"], _ = json.Marshal(info.Reason)
		if key := info.Metadata["module_key"]; key != "" {
			body["module_key"], _ = json.Marshal(key)
		}
		body["details"], _ = json.Marshal(details)
		return json.Marshal(body)
	}
	return raw, nil
}

func isOperationLogMessage(value any) bool {
	t := reflect.TypeOf(value)
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t != nil && strings.Contains(t.PkgPath(), "/pkg/apiclient/operationlog")
}

var operationLogNullableFields = map[string]bool{
	"startedAt": true, "identityVerified": true, "identitySnapshotComplete": true,
	"durationMs": true, "resourceCount": true, "resourcesComplete": true, "responseWriteFailed": true,
	"finishedAt": true, "beforeValue": true, "afterValue": true, "beforeAvailable": true,
	"referenceVerified": true,
	"actorAccountId":    true, "actorUsername": true, "primaryResourceType": true, "primaryResourceId": true,
	"reasonCode": true, "businessState": true, "provider": true, "targetAccountId": true,
	"requestId": true, "parentOperationId": true, "businessRequestId": true,
	"grpcCode": true, "httpStatus": true, "producerId": true,
	"firstReceivedAt": true, "lastReceivedAt": true, "snapshotSequence": true,
	"checkedAt": true, "queryReady": true, "lastPublishedAt": true, "pendingEvents": true,
	"oldestPendingReceivedAt": true, "totalObserved": true, "producersComplete": true,
	"lastSeenAt": true, "stoppedAt": true, "attemptedEvents": true,
	"confirmedEvents": true, "unconfirmedEvents": true, "invalidEvents": true,
	"capacityRejectedEvents": true, "lastFailureAt": true, "lastFailureCode": true,
	"lastRecoveredAt": true, "persistenceReachable": true, "lastConfirmedAt": true,
	"observedAt": true, "inFlightEvents": true,
	"requested": true, "confirmed": true, "failed": true, "unknown": true,
}

func flattenOperationLogNullables(raw []byte) []byte {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return raw
	}
	var flatten func(any, string) any
	flatten = func(node any, key string) any {
		if operationLogNullableFields[key] {
			if object, ok := node.(map[string]any); ok && len(object) == 1 {
				if scalar, exists := object["value"]; exists {
					return scalar
				}
			}
		}
		switch object := node.(type) {
		case map[string]any:
			for childKey, child := range object {
				object[childKey] = flatten(child, childKey)
			}
		case []any:
			for i, child := range object {
				object[i] = flatten(child, key)
			}
		}
		return node
	}
	return func() []byte {
		flattened := flatten(value, "")
		data, err := json.Marshal(flattened)
		if err != nil {
			return raw
		}
		return data
	}()
}
