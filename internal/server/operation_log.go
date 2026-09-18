package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/accountcredentials"
	operationlogapiclient "github.com/useryege/athena/internal/operationlog/apiclient"
	"github.com/useryege/athena/internal/operationlog/event"
	"github.com/useryege/athena/internal/operationlog/ingest"
	"github.com/useryege/athena/internal/operationlog/record"
	operationlogrpcconfig "github.com/useryege/athena/internal/operationlog/rpcconfig"
	operationlogstore "github.com/useryege/athena/internal/operationlog/store"
	serveroperationlog "github.com/useryege/athena/internal/server/operationlog"
	pb "github.com/useryege/athena/pkg/apiclient/operationlog"
	util_session "github.com/useryege/athena/util/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// beginOperationLog installs the one request-scoped recorder for a catalogued
// gRPC method. Unknown and query methods remain untouched, and a missing or
// unhealthy producer degrades to a no-op recorder without affecting the RPC.
func (server *AthenaServer) beginOperationLog(ctx context.Context, method string) (context.Context, *record.Recorder) {
	if server == nil || server.operationLogProducer == nil {
		return ctx, nil
	}
	ctx = record.WithRequest(ctx, server.operationLogProducer)
	r := record.BeginEntry(ctx, method)
	if r == nil {
		return ctx, nil
	}
	return r.Context(), r
}

func (server *AthenaServer) bindOperationLogActor(ctx context.Context, r *record.Recorder) {
	if r == nil {
		return
	}
	credential, ok := util_session.AuthenticatedCredentialFromContext(ctx)
	if !ok {
		return
	}
	actor := event.Actor{Role: "UNKNOWN", Realm: "UNKNOWN", CredentialKind: operationLogCredentialKind(credential.Capability), IdentityVerified: true}
	if credential.AccountID != "" {
		id := credential.AccountID
		actor.AccountID = &id
	}
	if server.credentialMgr != nil {
		if account, err := server.credentialMgr.Get(credential.AccountID); err == nil {
			actor.UsernameSnapshot = stringPtrIfNonEmpty(account.Username)
			if account.Administrator {
				actor.Role, actor.Realm = "ADMINISTRATOR", "ADMIN"
			} else {
				actor.Role, actor.Realm = "MEMBER", "MEMBER"
			}
			actor.IdentitySnapshotComplete = actor.AccountID != nil && actor.UsernameSnapshot != nil
			provider := string(account.IdentityProvider)
			if provider != "" {
				actor.Provider = &provider
			}
		}
	}
	_ = r.BindActor(actor)
}

// initializeOperationLog keeps collection and query dependencies optional for
// the API process. A missing or invalid service configuration yields a
// bounded unavailable facade and never prevents ordinary API startup.
func (server *AthenaServer) initializeOperationLog(ctx context.Context) {
	server.operationLogClient = operationlogapiclient.NewUnavailableClient("client_configuration_invalid")
	if dsn := strings.TrimSpace(os.Getenv("ATHENA_OPERATION_LOG_POSTGRES_DSN")); dsn != "" {
		store, err := operationlogstore.OpenProducer(ctx, dsn)
		if err != nil {
			log.WithField("reason", "producer_configuration_invalid").Warn("operation log producer unavailable")
		} else {
			server.operationLogStore = store
			server.operationLogProducer = ingest.New(store)
		}
	}
	if cfg, err := operationlogrpcconfig.LoadClient(os.LookupEnv); err == nil {
		client, closer, clientErr := operationlogapiclient.NewClient(operationlogapiclient.Config{Address: cfg.Address, Token: cfg.Token, Transport: cfg.Transport, CAFile: cfg.CAFile, ServerName: cfg.ServerName, MaxMessageBytes: cfg.MaxMessageBytes})
		if clientErr == nil {
			server.operationLogClient = client
			server.operationLogClientCloser = closeFunc(closer)
		} else {
			log.WithField("reason", "client_configuration_invalid").Warn("operation log facade unavailable")
		}
	} else {
		log.WithField("reason", "client_configuration_invalid").Warn("operation log facade unavailable")
	}
	server.operationLogService = serveroperationlog.NewForwardingServer(server.operationLogClient, server.operationLogViewerProto)
	server.operationLogService.Capture = server.operationLogCaptureStatus
	server.operationLogService.Actions = operationLogActions
}

func (server *AthenaServer) closeOperationLog() error {
	if server == nil {
		return nil
	}
	var result error
	if server.operationLogClientCloser != nil {
		result = errors.Join(result, server.operationLogClientCloser.Close())
		server.operationLogClientCloser = nil
	}
	if server.operationLogProducer != nil {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		result = errors.Join(result, server.operationLogProducer.Close(closeCtx))
		cancel()
		server.operationLogProducer = nil
	}
	if server.operationLogStore != nil {
		server.operationLogStore.Close()
		server.operationLogStore = nil
	}
	return result
}

type closeFunc func() error

func (f closeFunc) Close() error { return f() }

func (server *AthenaServer) operationLogViewerProto(ctx context.Context) (*operationlogapiclient.Viewer, error) {
	accountID, realm, kind, binding, revision, ok := operationLogViewer(ctx)
	if !ok || accountID == "" {
		return nil, status.Error(codes.Unauthenticated, "operation log viewer identity required")
	}
	if server != nil && server.credentialMgr != nil {
		if account, err := server.credentialMgr.Get(accountID); err == nil && account.Administrator {
			realm = "ADMIN"
		}
	}
	return &operationlogapiclient.Viewer{AccountId: accountID, Realm: realm, CredentialKind: kind, SessionBinding: binding, AccessRevision: revision}, nil
}

func (server *AthenaServer) operationLogCaptureStatus(context.Context) (*pb.CaptureStatus, error) {
	statusValue := ingest.Status{}
	if server != nil && server.operationLogProducer != nil {
		statusValue = server.operationLogProducer.Snapshot()
	}
	optionalTime := func(v *time.Time) *pb.NullableString {
		if v == nil || v.IsZero() {
			return nil
		}
		return &pb.NullableString{Value: v.Format(time.RFC3339Nano)}
	}
	optionalString := func(v *string) *pb.NullableString {
		if v == nil || *v == "" {
			return nil
		}
		return &pb.NullableString{Value: *v}
	}
	return &pb.CaptureStatus{
		InstanceId:             "api",
		StartedAt:              optionalTime(&statusValue.StartedAt),
		ObservedAt:             optionalTime(&statusValue.ObservedAt),
		AttemptedEvents:        &pb.NullableString{Value: fmt.Sprintf("%d", statusValue.AttemptedEvents)},
		ConfirmedEvents:        &pb.NullableString{Value: fmt.Sprintf("%d", statusValue.ConfirmedEvents)},
		UnconfirmedEvents:      &pb.NullableString{Value: fmt.Sprintf("%d", statusValue.UnconfirmedEvents)},
		InvalidEvents:          &pb.NullableString{Value: fmt.Sprintf("%d", statusValue.InvalidEvents)},
		CapacityRejectedEvents: &pb.NullableString{Value: fmt.Sprintf("%d", statusValue.CapacityRejectedEvents)},
		InFlightEvents:         &pb.NullableString{Value: fmt.Sprintf("%d", statusValue.InFlightEvents)},
		LastFailureCode:        optionalString(statusValue.LastFailureCode),
		PersistenceReachable:   optionalBool(statusValue.PersistenceReachable),
		LastConfirmedAt:        optionalTime(statusValue.LastConfirmedAt),
	}, nil
}

func optionalBool(v *bool) *pb.NullableBool {
	if v == nil {
		return nil
	}
	return &pb.NullableBool{Value: *v}
}

func operationLogActions(context.Context) (*pb.ListOperationLogActionsResponse, error) {
	seen := map[string]bool{}
	out := &pb.ListOperationLogActionsResponse{CatalogVersion: "v1"}
	for _, entry := range event.Catalog() {
		if seen[entry.ActionCode] {
			continue
		}
		seen[entry.ActionCode] = true
		out.Actions = append(out.Actions, &pb.OperationLogAction{Code: entry.ActionCode, ModuleCode: entry.ModuleCode, Label: entry.ActionCode, ResourceType: entry.ResourceType})
	}
	return out, nil
}

func operationLogCredentialKind(c accountcredentials.Capability) string {
	switch c {
	case accountcredentials.CapabilityLogin:
		return "LOGIN_SESSION"
	case accountcredentials.CapabilityAPIKey:
		return "API_KEY"
	case accountcredentials.CapabilityDevelopment:
		return "DEVELOPMENT"
	default:
		return "UNAUTHENTICATED"
	}
}

func stringPtrIfNonEmpty(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func operationLogSessionBinding(ctx context.Context) []byte {
	credential, ok := util_session.AuthenticatedCredentialFromContext(ctx)
	if !ok {
		return nil
	}
	v := credential.JTI
	if v == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(v))
	return sum[:]
}

func operationLogViewer(ctx context.Context) (string, string, string, []byte, uint64, bool) {
	credential, ok := util_session.AuthenticatedCredentialFromContext(ctx)
	if !ok {
		return "", "", "", nil, 0, false
	}
	account := "MEMBER"
	if realm, err := applicationRealmFromIncomingContext(ctx); err == nil && realm == accountcredentials.ApplicationRealmAdmin {
		account = "ADMIN"
	}
	return credential.AccountID, account, operationLogCredentialKind(credential.Capability), operationLogSessionBinding(ctx), credential.AccessRevision, true
}

func finishOperationLog(r *record.Recorder, err error) {
	if r == nil {
		return
	}
	if err != nil {
		r.ObserveError(err)
	} else {
		r.ObserveGRPC(codes.OK)
	}
	r.Finish()
}

// observeOperationLogResult captures only fields and outcomes explicitly mapped
// for this exact catalog entry. It is intentionally a conservative transport
// adapter: a successful gRPC response is not itself evidence of a durable
// mutation, so the per-entry evidence paths below must be present before a
// success/accepted result or mutation effect is recorded.
func observeOperationLogResult(r *record.Recorder, method string, req, response any) {
	if r == nil {
		return
	}
	entry, ok := event.Lookup(method)
	if !ok || entry.Transport != "grpc" {
		return
	}
	requestJSON := operationLogJSON(req)
	responseJSON := operationLogJSON(response)
	for _, key := range entry.AllowedDetails {
		if value, found := operationLogDetailValue(entry.Entry, key, requestJSON, responseJSON); found {
			if detail, ok := operationLogValueForEntry(entry.Entry, key, value); ok {
				r.Detail(key, detail)
			}
		}
	}
	if resourceType, resourceID, ok := operationLogResource(entry.Entry, entry.ResourceType, requestJSON, responseJSON); ok {
		resource := event.Resource{Type: resourceType, ID: resourceID, ReferenceVerified: true, Primary: true}
		r.Resources([]event.Resource{resource}, nil, true)
		if resourceType == "account" && strings.HasPrefix(entry.ActionCode, "account.") {
			r.TargetAccountID(resourceID)
		}
	}
	if !operationLogResponseEvidence(entry.Entry, responseJSON) {
		return
	}
	if entry.SuccessPolicy == "ACCEPTED" {
		r.Result(event.Accepted, "")
	} else {
		r.Result(event.Succeeded, "")
	}
	r.Effect(operationLogEffect(entry.ActionCode))
}

type operationLogJSONValue map[string]any

type operationLogFieldPath struct {
	response bool
	path     string
}

func operationLogJSON(v any) operationLogJSONValue {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil || len(b) == 0 || string(b) == "null" {
		return nil
	}
	var out map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(b)))
	decoder.UseNumber()
	if decoder.Decode(&out) != nil {
		return nil
	}
	return out
}

func operationLogDetailValue(entry, key string, request, response operationLogJSONValue) (any, bool) {
	for _, candidate := range operationLogDetailPaths(entry, key) {
		input := request
		if candidate.response {
			input = response
		}
		if value, ok := operationLogExactPath(input, candidate.path); ok {
			return value, true
		}
	}
	if value, ok := operationLogStaticDetail(entry, key); ok {
		return value, true
	}
	return nil, false
}

// operationLogDetailPaths is deliberately an allowlist of protobuf field
// paths. It never searches arbitrary nested maps or accepts an alias merely
// because a similarly named field happened to be returned by a facade.
func operationLogDetailPaths(entry, key string) []operationLogFieldPath {
	r := func(path string) operationLogFieldPath { return operationLogFieldPath{path: path} }
	q := func(path string) operationLogFieldPath { return operationLogFieldPath{response: true, path: path} }
	switch entry {
	case "/account.AccountService/UpdateAccountAccess":
		return map[string][]operationLogFieldPath{
			"requestedAccess":   {r("access")},
			"expectedRevision":  {r("access.revision")},
			"confirmedAccess":   {q("access")},
			"confirmedRevision": {q("access.revision")},
		}[key]
	case "/account.AccountService/UpdateAccountProfile":
		return map[string][]operationLogFieldPath{
			"expectedRevision":  {r("expectedRevision")},
			"confirmedRevision": {q("revision")},
		}[key]
	case "/account.AccountService/UpdateAccountTier":
		return map[string][]operationLogFieldPath{
			"requestedTier": {r("tier")}, "expectedRevision": {r("expectedRevision")},
			"confirmedTier": {q("tier")}, "confirmedRevision": {q("revision")},
		}[key]
	case "/account.AccountService/CreateToken":
		if key == "expiresIn" {
			return []operationLogFieldPath{r("expiresIn")}
		}
		if key == "displayId" {
			return []operationLogFieldPath{r("id")}
		}
	case "/account.AccountService/DeleteToken":
		if key == "displayId" {
			return []operationLogFieldPath{r("id")}
		}
	case "/managedoo.ManagedOOService/ScanManagedOOBlock":
		return map[string][]operationLogFieldPath{
			"blockNumber":   {r("blockNumber"), q("blockNumber")},
			"proposalCount": {q("proposalCount")}, "disputeCount": {q("disputeCount")},
		}[key]
	case "/moduleaccess.ModuleAccessService/UpdateModuleAccessSetting":
		return map[string][]operationLogFieldPath{
			"moduleKey":     {r("moduleKey"), q("setting.moduleKey")},
			"requestedOpen": {r("state")}, "confirmedOpen": {q("setting.state")},
		}[key]
	case "/notification.NotificationService/CreateTelegramBindingAttempt":
		return map[string][]operationLogFieldPath{"attemptStatus": {q("attempt.status")}, "revision": {q("attempt.revision")}}[key]
	case "/notification.NotificationService/DeleteTelegramBindingAttempt":
		return nil // the public response exposes only Deleted; no status or ID
	case "/notification.NotificationService/DeleteTelegramBinding":
		return nil // the public response exposes only Deleted; no binding status
	case "/notification.NotificationService/SendSystemNotificationTest":
		return map[string][]operationLogFieldPath{"deliveryId": {q("notificationId")}, "deliveryStatus": {q("status")}}[key]
	case "/profitsharing.ProfitSharingService/CreateRound", "/profitsharing.ProfitSharingService/OpenRound", "/profitsharing.ProfitSharingService/PublishRound":
		return map[string][]operationLogFieldPath{"slug": {q("round.slug"), r("slug")}, "revision": {q("round.revision")}, "state": {q("round.phase")}}[key]
	case "/profitsharing.ProfitSharingService/UpdateRound":
		return map[string][]operationLogFieldPath{"oldSlug": {r("currentSlug")}, "newSlug": {r("slug"), q("round.slug")}, "revision": {q("round.revision")}}[key]
	case "/profitsharing.ProfitSharingService/CloseBallot":
		return map[string][]operationLogFieldPath{"slug": {r("slug"), q("round.slug")}, "ballotId": {q("round.ballotNumber")}, "state": {q("round.phase")}}[key]
	case "/profitsharing.ProfitSharingService/UpdateProposal":
		return map[string][]operationLogFieldPath{"slug": {r("slug")}, "revision": {q("proposal.revision")}}[key]
	case "/profitsharing.ProfitSharingService/SubmitProposal", "/profitsharing.ProfitSharingService/ReopenProposal":
		return map[string][]operationLogFieldPath{"slug": {r("slug")}, "revision": {q("proposal.revision")}, "state": {q("proposal.status")}}[key]
	case "/profitsharing.ProfitSharingService/SubmitVote":
		return map[string][]operationLogFieldPath{"slug": {r("slug")}, "ballotId": {q("ballotNumber")}}[key]
	case "/servicestatus.ServiceStatusService/RunEtherscanGatewayProbe":
		return map[string][]operationLogFieldPath{"runId": {q("runId")}, "requestsPerKey": {q("requestsPerKey"), r("requestsPerKey")}, "gatewayCount": {q("gatewayCount")}, "state": {q("status")}}[key]
	case "/tokenapi.TokenOperationsService/UpdateChainCheckpoint":
		return map[string][]operationLogFieldPath{"chainId": {r("chainId"), q("checkpoint.chainId")}, "blockNumber": {q("checkpoint.cursorBlockNumber")}}[key]
	case "/tokenapi.TokenPolicyService/CreateContractCodeBlocklistEntry", "/tokenapi.TokenPolicyService/UpdateContractCodeBlocklistEntry", "/tokenapi.TokenPolicyService/DeleteContractCodeBlocklistEntry":
		return map[string][]operationLogFieldPath{"codeHash": {r("codeHash")}}[key]
	case "/tokenapi.TokenPolicyService/CreateWalletBlocklistEntry", "/tokenapi.TokenPolicyService/UpdateWalletBlocklistEntry", "/tokenapi.TokenPolicyService/DeleteWalletBlocklistEntry":
		return map[string][]operationLogFieldPath{"walletAddress": {r("wallet")}}[key]
	case "/tradersync.TraderSyncService/CreateSubscription":
		return map[string][]operationLogFieldPath{"subscriptionId": {q("subscription.id")}, "revision": {q("subscription.revision"), r("expectedRevision")}}[key]
	case "/tradersync.TraderSyncService/PauseSubscription", "/tradersync.TraderSyncService/ResumeSubscription", "/tradersync.TraderSyncService/CancelSubscription":
		return map[string][]operationLogFieldPath{"subscriptionId": {r("subscriptionId"), q("subscription.id")}, "expectedRevision": {r("expectedRevision")}, "confirmedRevision": {q("subscription.revision")}, "state": {q("subscription.status")}}[key]
	case "/tradersync.TraderSyncService/UpdateTargetNote":
		return map[string][]operationLogFieldPath{"walletAddress": {r("wallet"), q("note.wallet")}, "expectedRevision": {r("expectedRevision")}, "confirmedRevision": {q("note.revision")}}[key]
	case "/wallet.WalletService/BatchCreateWallets", "/wallet.WalletService/BatchImportWallets":
		return map[string][]operationLogFieldPath{"walletType": {r("walletType")}, "requestedCount": {r("count"), r("privateKeys")}, "confirmedCount": {q("results"), q("items")}, "walletIds": {q("results"), q("items")}}[key]
	case "/wallet.WalletService/UpdateWalletRemark", "/wallet.WalletService/UpdateWalletAvatarPreset":
		return map[string][]operationLogFieldPath{"expectedRevision": {r("expectedRevision")}, "confirmedRevision": {q("item.revision")}, "avatarPresetId": {r("avatarPresetId")}}[key]
	}
	return nil
}

func operationLogStaticDetail(entry, key string) (any, bool) {
	switch {
	case entry == "/account.AccountService/UpdateAccountAccess" && key == "beforeAvailable":
		return false, true
	case entry == "/account.AccountService/UpdateAccountProfile" && key == "changedFields":
		return []any{"displayName"}, true
	case entry == "/account.AccountService/UpdateAccountTier" && key == "beforeAvailable":
		return false, true
	case entry == "/moduleaccess.ModuleAccessService/UpdateModuleAccessSetting" && key == "beforeAvailable":
		return false, true
	case entry == "/profitsharing.ProfitSharingService/UpdateRound" && key == "changedFields":
		return []any{"slug", "title", "participants"}, true
	case entry == "/profitsharing.ProfitSharingService/UpdateProposal" && key == "changedFields":
		return []any{"items"}, true
	case strings.Contains(entry, "ContractCodeBlocklistEntry") && key == "changedFields":
		return []any{"note"}, true
	case strings.Contains(entry, "WalletBlocklistEntry") && key == "changedFields":
		return []any{"note"}, true
	case entry == "/tradersync.TraderSyncService/CreateSubscription" && key == "noteChanged":
		return nil, false
	case entry == "/tradersync.TraderSyncService/UpdateTargetNote" && key == "changedFields":
		return []any{"note"}, true
	case strings.Contains(entry, "WalletService/UpdateWalletRemark") && key == "changedFields":
		return []any{"remark"}, true
	case strings.Contains(entry, "WalletService/UpdateWalletAvatarPreset") && key == "changedFields":
		return []any{"avatarPresetId"}, true
	}
	return nil, false
}

func operationLogExactPath(input map[string]any, path string) (any, bool) {
	if input == nil || path == "" {
		return nil, false
	}
	var current any = input
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func operationLogResponseEvidence(entry string, response operationLogJSONValue) bool {
	paths := map[string]string{
		"/account.AccountService/UpdateAccountAccess":                    "access.revision",
		"/account.AccountService/UpdateAccountProfile":                   "revision",
		"/account.AccountService/UpdateAccountTier":                      "revision",
		"/account.AccountService/CreateToken":                            "token",
		"/managedoo.ManagedOOService/ScanManagedOOBlock":                 "blockNumber",
		"/moduleaccess.ModuleAccessService/UpdateModuleAccessSetting":    "setting.moduleKey",
		"/notification.NotificationService/CreateTelegramBindingAttempt": "attempt.id",
		"/notification.NotificationService/DeleteTelegramBindingAttempt": "deleted",
		"/notification.NotificationService/DeleteTelegramBinding":        "deleted",
		"/notification.NotificationService/SendSystemNotificationTest":   "notificationId",
		"/profitsharing.ProfitSharingService/CreateRound":                "round.slug",
		"/profitsharing.ProfitSharingService/UpdateRound":                "round.slug",
		"/profitsharing.ProfitSharingService/OpenRound":                  "round.slug",
		"/profitsharing.ProfitSharingService/PublishRound":               "round.slug",
		"/profitsharing.ProfitSharingService/CloseBallot":                "round.slug",
		"/profitsharing.ProfitSharingService/UpdateProposal":             "proposal.id",
		"/profitsharing.ProfitSharingService/SubmitProposal":             "proposal.id",
		"/profitsharing.ProfitSharingService/ReopenProposal":             "proposal.id",
		"/profitsharing.ProfitSharingService/SubmitVote":                 "ballotNumber",
		"/servicestatus.ServiceStatusService/RunEtherscanGatewayProbe":   "runId",
		"/tokenapi.TokenOperationsService/UpdateChainCheckpoint":         "checkpoint.chainId",
		"/tokenapi.TokenPolicyService/CreateContractCodeBlocklistEntry":  "",
		"/tokenapi.TokenPolicyService/UpdateContractCodeBlocklistEntry":  "updatedCount",
		"/tokenapi.TokenPolicyService/DeleteContractCodeBlocklistEntry":  "deletedCount",
		"/tokenapi.TokenPolicyService/CreateWalletBlocklistEntry":        "",
		"/tokenapi.TokenPolicyService/UpdateWalletBlocklistEntry":        "updatedCount",
		"/tokenapi.TokenPolicyService/DeleteWalletBlocklistEntry":        "deletedCount",
		"/tradersync.TraderSyncService/CreateSubscription":               "subscription.id",
		"/tradersync.TraderSyncService/PauseSubscription":                "subscription.id",
		"/tradersync.TraderSyncService/ResumeSubscription":               "subscription.id",
		"/tradersync.TraderSyncService/CancelSubscription":               "subscription.id",
		"/tradersync.TraderSyncService/UpdateTargetNote":                 "note.revision",
		"/wallet.WalletService/BatchCreateWallets":                       "results",
		"/wallet.WalletService/BatchImportWallets":                       "items",
		"/wallet.WalletService/UpdateWalletRemark":                       "item.id",
		"/wallet.WalletService/UpdateWalletAvatarPreset":                 "item.id",
	}
	path, ok := paths[entry]
	if !ok {
		return false
	}
	if path == "" {
		// Empty response messages are still not evidence unless the facade
		// returned a typed count; callers can add a durable recorder hook.
		return false
	}
	value, ok := operationLogExactPath(response, path)
	if !ok {
		return false
	}
	if list, ok := value.([]any); ok {
		return len(list) > 0
	}
	if text, ok := value.(string); ok {
		return text != ""
	}
	if number, ok := value.(json.Number); ok {
		return number != "0"
	}
	return true
}

func operationLogEffect(action string) string {
	return strings.ToUpper(strings.ReplaceAll(action, ".", "_"))
}

func operationLogValueForEntry(entry, key string, value any) (event.Value, bool) {
	if key == "requestedCount" || key == "confirmedCount" {
		if values, ok := value.([]any); ok {
			value = json.Number(strconv.Itoa(len(values)))
		}
	}
	if key == "requestedTier" || key == "confirmedTier" {
		if n, ok := value.(json.Number); ok {
			switch string(n) {
			case "1":
				value = "standard"
			case "2":
				value = "pro"
			default:
				return event.Value{}, false
			}
		}
	}
	if key == "requestedOpen" || key == "confirmedOpen" {
		if n, ok := value.(json.Number); ok {
			switch string(n) {
			case "1":
				value = true
			case "2":
				value = false
			default:
				return event.Value{}, false
			}
		}
	}
	if key == "state" {
		if n, ok := value.(json.Number); ok {
			states := map[string]string{"1": "draft", "2": "collecting", "3": "voting", "4": "closed"}
			if strings.Contains(entry, "proposal.") {
				states = map[string]string{"1": "draft", "2": "submitted"}
			}
			if state, ok := states[string(n)]; ok {
				value = state
			} else {
				return event.Value{}, false
			}
		}
	}
	if key == "walletIds" {
		if values, ok := value.([]any); ok {
			converted := make([]any, 0, len(values))
			for _, item := range values {
				if s, ok := item.(string); ok {
					converted = append(converted, s)
					continue
				}
				if n, ok := item.(json.Number); ok {
					converted = append(converted, string(n))
					continue
				}
				return event.Value{}, false
			}
			value = converted
		}
	}
	return operationLogValue(key, value)
}

func operationLogValue(key string, value any) (event.Value, bool) {
	switch v := value.(type) {
	case bool:
		return event.Bool(v), true
	case string:
		if operationLogNumericDetail(key) {
			if _, err := strconv.ParseUint(v, 10, 64); err != nil {
				return event.Value{}, false
			}
		}
		return event.String(v), true
	case json.Number:
		if !operationLogNumericDetail(key) {
			return event.Value{}, false
		}
		n, err := strconv.ParseUint(string(v), 10, 64)
		if err != nil {
			return event.Value{}, false
		}
		return event.String(strconv.FormatUint(n, 10)), true
	case []any:
		values := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				values = append(values, s)
			}
		}
		if len(values) != len(v) {
			return event.Value{}, false
		}
		return event.Strings(values), true
	case map[string]any:
		if key != "requestedAccess" && key != "confirmedAccess" {
			return event.Value{}, false
		}
		access := event.Access{
			LoginEnabled:         operationLogBoolField(v, "loginEnabled"),
			APIKeyEnabled:        operationLogBoolField(v, "apiKeyEnabled"),
			ProfitSharingEnabled: operationLogBoolField(v, "profitSharingEnabled"),
		}
		if revision, ok := operationLogStringField(v, "revision"); ok {
			if _, err := strconv.ParseUint(revision, 10, 64); err != nil {
				return event.Value{}, false
			}
			access.Revision = revision
		}
		modules, modulesPresent := v["moduleAccess"].([]any)
		if !modulesPresent {
			return event.Value{}, false
		}
		if modules, ok := v["moduleAccess"].([]any); ok {
			for _, item := range modules {
				module, ok := item.(map[string]any)
				if !ok {
					continue
				}
				name, nameOK := operationLogModuleName(module["module"])
				level, levelOK := operationLogAccessLevel(module["dataAccess"])
				if !levelOK {
					level, levelOK = operationLogAccessLevel(module["access"])
				}
				if nameOK && levelOK {
					access.ModuleAccess = append(access.ModuleAccess, event.ModuleAccess{Module: name, Access: level})
				}
			}
		}
		if len(access.ModuleAccess) != len(modules) {
			return event.Value{}, false
		}
		return event.AccessValue(access), true
	default:
		return event.Value{}, false
	}
}

func operationLogBoolField(value map[string]any, key string) bool {
	v, _ := value[key].(bool)
	return v
}

func operationLogModuleName(value any) (string, bool) {
	if text, ok := value.(string); ok {
		if strings.HasPrefix(text, "ACCOUNT_DATA_MODULE_") {
			text = strings.TrimPrefix(text, "ACCOUNT_DATA_MODULE_")
			text = strings.ToLower(text)
		}
		allowed := map[string]bool{
			"market_radar": true,
			"managed_oo":   true,
			"token":        true,
			"wallet":       true,
			"worm_trading": true,
			"trader_sync":  true,
			"solana":       true,
		}
		return text, allowed[text]
	}
	values := map[string]string{"1": "market_radar", "4": "managed_oo", "8": "token", "9": "wallet", "11": "worm_trading", "12": "trader_sync", "13": "solana"}
	if n, ok := value.(json.Number); ok {
		name, found := values[string(n)]
		return name, found
	}
	return "", false
}

func operationLogAccessLevel(value any) (string, bool) {
	if text, ok := value.(string); ok {
		if strings.HasPrefix(text, "ACCOUNT_DATA_ACCESS_") {
			text = strings.TrimPrefix(text, "ACCOUNT_DATA_ACCESS_")
			text = strings.ToLower(text)
		}
		return text, text == "none" || text == "read" || text == "read_write"
	}
	values := map[string]string{"0": "none", "1": "read", "2": "read_write"}
	if n, ok := value.(json.Number); ok {
		level, found := values[string(n)]
		return level, found
	}
	return "", false
}

func operationLogStringField(value map[string]any, key string) (string, bool) {
	switch v := value[key].(type) {
	case string:
		return v, true
	case json.Number:
		return string(v), true
	default:
		return "", false
	}
}

func operationLogNumericDetail(key string) bool {
	switch key {
	case "addedCount", "ballotId", "blockNumber", "chainId", "confirmedCount", "confirmedRevision", "disputeCount", "expectedRevision", "expiresIn", "gatewayCount", "itemCount", "proposalCount", "removedCount", "requestedCount", "requestsPerKey", "revision", "stepCount":
		return true
	default:
		return false
	}
}

var operationLogUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
var operationLogCodeHashPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)
var operationLogWalletAddressPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

func operationLogResource(entry, resourceType string, request, response operationLogJSONValue) (string, string, bool) {
	paths := map[string][]operationLogFieldPath{
		"/account.AccountService/UpdateAccountAccess":                    {{path: "id"}},
		"/account.AccountService/UpdateAccountProfile":                   {{path: "id"}},
		"/account.AccountService/UpdateAccountTier":                      {{path: "id"}},
		"/account.AccountService/CreateToken":                            {{path: "id"}},
		"/account.AccountService/DeleteToken":                            {{path: "id"}},
		"/managedoo.ManagedOOService/ScanManagedOOBlock":                 {{path: "blockNumber"}, {response: true, path: "blockNumber"}},
		"/moduleaccess.ModuleAccessService/UpdateModuleAccessSetting":    {{path: "moduleKey"}, {response: true, path: "setting.moduleKey"}},
		"/notification.NotificationService/CreateTelegramBindingAttempt": {{response: true, path: "attempt.id"}},
		"/notification.NotificationService/SendSystemNotificationTest":   {{response: true, path: "notificationId"}},
		"/profitsharing.ProfitSharingService/CreateRound":                {{path: "slug"}, {response: true, path: "round.slug"}},
		"/profitsharing.ProfitSharingService/UpdateRound":                {{path: "currentSlug"}, {response: true, path: "round.slug"}},
		"/profitsharing.ProfitSharingService/OpenRound":                  {{path: "slug"}, {response: true, path: "round.slug"}},
		"/profitsharing.ProfitSharingService/PublishRound":               {{path: "slug"}, {response: true, path: "round.slug"}},
		"/profitsharing.ProfitSharingService/CloseBallot":                {{path: "slug"}, {response: true, path: "round.slug"}},
		"/profitsharing.ProfitSharingService/UpdateProposal":             {{path: "slug"}, {response: true, path: "proposal.id"}},
		"/profitsharing.ProfitSharingService/SubmitProposal":             {{path: "slug"}, {response: true, path: "proposal.id"}},
		"/profitsharing.ProfitSharingService/ReopenProposal":             {{path: "slug"}, {response: true, path: "proposal.id"}},
		"/profitsharing.ProfitSharingService/SubmitVote":                 {{path: "slug"}, {response: true, path: "ballotNumber"}},
		"/servicestatus.ServiceStatusService/RunEtherscanGatewayProbe":   {{response: true, path: "runId"}},
		"/tokenapi.TokenOperationsService/UpdateChainCheckpoint":         {{path: "chainId"}, {response: true, path: "checkpoint.chainId"}},
		"/tokenapi.TokenPolicyService/CreateContractCodeBlocklistEntry":  {{path: "codeHash"}},
		"/tokenapi.TokenPolicyService/UpdateContractCodeBlocklistEntry":  {{path: "codeHash"}},
		"/tokenapi.TokenPolicyService/DeleteContractCodeBlocklistEntry":  {{path: "codeHash"}},
		"/tokenapi.TokenPolicyService/CreateWalletBlocklistEntry":        {{path: "wallet"}},
		"/tokenapi.TokenPolicyService/UpdateWalletBlocklistEntry":        {{path: "wallet"}},
		"/tokenapi.TokenPolicyService/DeleteWalletBlocklistEntry":        {{path: "wallet"}},
		"/tradersync.TraderSyncService/CreateSubscription":               {{response: true, path: "subscription.id"}},
		"/tradersync.TraderSyncService/PauseSubscription":                {{path: "subscriptionId"}, {response: true, path: "subscription.id"}},
		"/tradersync.TraderSyncService/ResumeSubscription":               {{path: "subscriptionId"}, {response: true, path: "subscription.id"}},
		"/tradersync.TraderSyncService/CancelSubscription":               {{path: "subscriptionId"}, {response: true, path: "subscription.id"}},
		"/tradersync.TraderSyncService/UpdateTargetNote":                 {{path: "wallet"}, {response: true, path: "note.wallet"}},
		"/wallet.WalletService/UpdateWalletRemark":                       {{response: true, path: "item.id"}},
		"/wallet.WalletService/UpdateWalletAvatarPreset":                 {{response: true, path: "item.id"}},
	}
	for _, candidate := range paths[entry] {
		input := request
		if candidate.response {
			input = response
		}
		value, ok := operationLogExactPath(input, candidate.path)
		if !ok {
			continue
		}
		id, ok := operationLogIDString(value)
		if ok && operationLogResourceIDValid(resourceType, id) {
			return resourceType, id, true
		}
	}
	return "", "", false
}

func operationLogIDString(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case json.Number:
		return string(v), true
	default:
		return "", false
	}
}

func operationLogResourceIDValid(resourceType, id string) bool {
	if id == "" || len(id) > 128 || strings.TrimSpace(id) != id {
		return false
	}
	switch resourceType {
	case "account", "binding_attempt", "subscription", "combination", "execution_plan", "execution", "execution_step", "cash_out", "cash_out_batch", "probe_run", "proposal":
		return operationLogUUIDPattern.MatchString(id)
	case "wallet", "notification_delivery":
		_, err := strconv.ParseUint(id, 10, 64)
		return err == nil && id != "0"
	case "wallet_address":
		return operationLogWalletAddressPattern.MatchString(id)
	case "block", "chain":
		_, err := strconv.ParseUint(id, 10, 64)
		return err == nil
	case "code_hash":
		return operationLogCodeHashPattern.MatchString(id)
	default:
		for _, r := range id {
			if r > 127 || !(r == '_' || r == '-' || r == '.' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') {
				return false
			}
		}
		return true
	}
}

func operationLogPanicError() error {
	return status.Error(codes.Internal, "operation handler panicked")
}
