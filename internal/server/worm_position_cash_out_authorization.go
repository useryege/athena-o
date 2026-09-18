package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"math"
	"net/http"
	"slices"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/googleoidc"
	"github.com/useryege/athena/internal/phantomauth"
	"github.com/useryege/athena/internal/walletsecret"
	wormtradingapiclient "github.com/useryege/athena/internal/wormtrading/apiclient"
)

const wormPositionCashOutDevelopmentProofKind = "DEVELOPMENT"

type wormPositionCashOutVerifiedProof struct {
	CashOutID              string
	CommandID              string
	ExpectedRevision       int64
	AccountID              string
	SessionJTIDigestSHA256 []byte
	AccessRevision         uint64
	IntentDigestSHA256     []byte
	ProofKind              string
}

func (server *AthenaServer) enableWormPositionCashOutAuthorization() error {
	authenticate := func(request *http.Request) (context.Context, accountcredentials.AuthenticatedCredential, error) {
		return server.authenticateWormTradingIdentityHTTP(request, accountaccess.AccessLevelReadWrite)
	}
	if server.googleOIDC != nil {
		if err := server.googleOIDC.EnableWormPositionCashOutAuthorization(
			server.RedisClient,
			authenticate,
			server.admitWormAccess,
			server.credentialMgr,
			server.loadGoogleWormPositionCashOutDescriptor,
			server.authorizeGoogleWormPositionCashOut,
			writeWormPositionCashOutAuthorizationError,
		); err != nil {
			return err
		}
	}
	if server.phantomAuth != nil {
		if err := server.phantomAuth.EnableWormPositionCashOutAuthorization(
			server.RedisClient,
			authenticate,
			server.admitWormAccess,
			server.credentialMgr,
			server.loadPhantomWormPositionCashOutDescriptor,
			server.authorizePhantomWormPositionCashOut,
			writeWormPositionCashOutAuthorizationError,
		); err != nil {
			return err
		}
	}
	return nil
}

func (server *AthenaServer) authorizeGoogleWormPositionCashOut(
	ctx context.Context,
	request googleoidc.WormPositionCashOutAuthorizationRequest,
) (any, error) {
	return server.authorizeWormPositionCashOutProof(ctx, wormPositionCashOutVerifiedProof{
		CashOutID: request.CashOutID, CommandID: request.CommandID,
		ExpectedRevision: request.ExpectedRevision, AccountID: request.AccountID,
		SessionJTIDigestSHA256: request.SessionJTIDigestSHA256,
		AccessRevision:         request.AccessRevision, IntentDigestSHA256: request.IntentDigestSHA256,
		ProofKind: request.ProofKind,
	})
}

func (server *AthenaServer) authorizePhantomWormPositionCashOut(
	ctx context.Context,
	request phantomauth.WormPositionCashOutAuthorizationRequest,
) (any, error) {
	return server.authorizeWormPositionCashOutProof(ctx, wormPositionCashOutVerifiedProof{
		CashOutID: request.CashOutID, CommandID: request.CommandID,
		ExpectedRevision: request.ExpectedRevision, AccountID: request.AccountID,
		SessionJTIDigestSHA256: request.SessionJTIDigestSHA256,
		AccessRevision:         request.AccessRevision, IntentDigestSHA256: request.IntentDigestSHA256,
		ProofKind: request.ProofKind,
	})
}

func (server *AthenaServer) authorizeWormPositionCashOutProof(
	ctx context.Context,
	request wormPositionCashOutVerifiedProof,
) (wormPositionCashOutResponse, error) {
	cashOutID, err := canonicalWormExecutionID(request.CashOutID, "position Cash Out ID")
	if err != nil || cashOutID != request.CashOutID {
		return wormPositionCashOutResponse{}, status.Error(codes.InvalidArgument, "position Cash Out authorization binding is invalid")
	}
	commandID, err := canonicalWormExecutionID(request.CommandID, "commandId")
	if err != nil || commandID != request.CommandID {
		return wormPositionCashOutResponse{}, status.Error(codes.InvalidArgument, "position Cash Out authorization command is invalid")
	}
	accountID, err := accountcredentials.CanonicalAccountID(request.AccountID)
	if err != nil || accountID != request.AccountID || request.ExpectedRevision <= 0 ||
		len(request.SessionJTIDigestSHA256) != sha256.Size || len(request.IntentDigestSHA256) != sha256.Size ||
		request.AccessRevision == 0 || request.AccessRevision > math.MaxInt64 {
		return wormPositionCashOutResponse{}, status.Error(codes.InvalidArgument, "position Cash Out authorization account or intent is invalid")
	}
	if request.ProofKind != googleoidc.WormPositionCashOutProofKindGoogle &&
		request.ProofKind != phantomauth.WormPositionCashOutProofKindPhantom &&
		request.ProofKind != wormPositionCashOutDevelopmentProofKind {
		return wormPositionCashOutResponse{}, status.Error(codes.InvalidArgument, "position Cash Out proof kind is invalid")
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		return wormPositionCashOutResponse{}, err
	}
	result, err := client.AuthorizePositionCashOut(ctx, &wormtradingapiclient.AuthorizePositionCashOutRequest{
		OwnerAccountId: accountID, Id: cashOutID, CommandId: commandID,
		ExpectedRevision: request.ExpectedRevision, ProofKind: request.ProofKind,
		SessionJtiDigest: append([]byte(nil), request.SessionJTIDigestSHA256...),
		AccessRevision:   int64(request.AccessRevision),
	})
	if err != nil {
		return wormPositionCashOutResponse{}, err
	}
	cashOut := result.GetCashOut()
	if cashOut == nil || len(cashOut.GetIntentSha256()) != sha256.Size ||
		subtle.ConstantTimeCompare(cashOut.GetIntentSha256(), request.IntentDigestSHA256) != 1 {
		return wormPositionCashOutResponse{}, status.Error(codes.Internal, "Worm Trading returned a mismatched Cash Out intent")
	}
	return projectWormPositionCashOut(cashOut, accountID, cashOutID)
}

func (server *AthenaServer) loadGoogleWormPositionCashOutDescriptor(
	ctx context.Context,
	request googleoidc.WormPositionCashOutDescriptorRequest,
) (googleoidc.WormPositionCashOutDescriptor, error) {
	descriptor, err := server.loadWormPositionCashOutDescriptor(ctx, request.CashOutID, request.AccountID, request.ExpectedRevision)
	if err != nil {
		return googleoidc.WormPositionCashOutDescriptor{}, err
	}
	return googleoidc.WormPositionCashOutDescriptor{
		CashOutID: descriptor.CashOutID, Revision: descriptor.Revision,
		IntentDigestSHA256: descriptor.IntentDigestSHA256,
	}, nil
}

func (server *AthenaServer) loadPhantomWormPositionCashOutDescriptor(
	ctx context.Context,
	request phantomauth.WormPositionCashOutDescriptorRequest,
) (phantomauth.WormPositionCashOutDescriptor, error) {
	descriptor, err := server.loadWormPositionCashOutDescriptor(ctx, request.CashOutID, request.AccountID, request.ExpectedRevision)
	if err != nil {
		return phantomauth.WormPositionCashOutDescriptor{}, err
	}
	return phantomauth.WormPositionCashOutDescriptor{
		CashOutID: descriptor.CashOutID, Revision: descriptor.Revision,
		IntentDigestSHA256: descriptor.IntentDigestSHA256,
	}, nil
}

type wormPositionCashOutDescriptor struct {
	CashOutID          string
	Revision           int64
	IntentDigestSHA256 []byte
}

func (server *AthenaServer) loadWormPositionCashOutDescriptor(
	ctx context.Context,
	cashOutID string,
	accountID string,
	expectedRevision int64,
) (wormPositionCashOutDescriptor, error) {
	if expectedRevision <= 0 {
		return wormPositionCashOutDescriptor{}, status.Error(codes.InvalidArgument, "position Cash Out revision is invalid")
	}
	canonicalID, err := canonicalWormExecutionID(cashOutID, "position Cash Out ID")
	if err != nil || canonicalID != cashOutID {
		return wormPositionCashOutDescriptor{}, status.Error(codes.InvalidArgument, "position Cash Out ID is invalid")
	}
	canonicalAccountID, err := accountcredentials.CanonicalAccountID(accountID)
	if err != nil || canonicalAccountID != accountID {
		return wormPositionCashOutDescriptor{}, status.Error(codes.InvalidArgument, "position Cash Out account is invalid")
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		return wormPositionCashOutDescriptor{}, err
	}
	result, err := client.GetPositionCashOut(ctx, &wormtradingapiclient.GetPositionCashOutRequest{
		OwnerAccountId: accountID, Id: cashOutID,
	})
	if err != nil {
		return wormPositionCashOutDescriptor{}, err
	}
	cashOut := result.GetCashOut()
	projection, err := projectWormPositionCashOut(cashOut, accountID, cashOutID)
	if err != nil {
		return wormPositionCashOutDescriptor{}, err
	}
	if projection.Revision != expectedRevision || !slices.Contains(projection.AllowedActions, "AUTHORIZE_CASH_OUT") ||
		cashOut == nil || len(cashOut.GetIntentSha256()) != sha256.Size {
		return wormPositionCashOutDescriptor{}, status.Error(codes.FailedPrecondition, "position Cash Out cannot be authorized")
	}
	return wormPositionCashOutDescriptor{
		CashOutID: cashOutID, Revision: projection.Revision,
		IntentDigestSHA256: append([]byte(nil), cashOut.GetIntentSha256()...),
	}, nil
}

func (server *AthenaServer) developmentWormPositionCashOutAuthorization(w http.ResponseWriter, request *http.Request) {
	observed := false
	cashOutID := ""
	defer func() {
		if !observed {
			observeWormAuthorization(request.Context(), "cash_out", cashOutID, "WORM_CASH_OUT_AUTHORIZE", "DEVELOPMENT", "authorization_failed", false, operationLogHTTPStatus(w))
		}
	}()
	walletsecret.SetSecretResponseHeaders(w)
	if request.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !server.DisableAuth || !requestIsLoopback(request) || !server.validWormConnectionOrigin(request) {
		writeWormPositionCashOutAuthorizationError(w, walletsecret.ErrWormLoginSessionRequired)
		return
	}
	if err := rejectWormCombinationQuery(request); err != nil {
		writeWormPositionCashOutAuthorizationError(w, err)
		return
	}
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelReadWrite)
	if err != nil {
		writeWormPositionCashOutAuthorizationError(w, err)
		return
	}
	cashOutID, err = canonicalWormExecutionID(request.PathValue("cashOutId"), "position Cash Out ID")
	if err != nil {
		writeWormPositionCashOutAuthorizationError(w, err)
		return
	}
	var input wormPositionCashOutCommandInput
	if err := decodeWormExecutionJSON(w, request, &input); err != nil {
		writeWormPositionCashOutAuthorizationError(w, err)
		return
	}
	commandID, err := canonicalWormExecutionID(input.CommandID, "commandId")
	if err != nil || validateWormExecutionExpectedRevision(input.ExpectedRevision) != nil {
		writeWormPositionCashOutAuthorizationError(w, status.Error(codes.InvalidArgument, "position Cash Out authorization binding is invalid"))
		return
	}
	sessionDigest, accessRevision, err := wormExecutionCredentialBinding(credential)
	if err != nil {
		writeWormPositionCashOutAuthorizationError(w, err)
		return
	}
	descriptor, err := server.loadWormPositionCashOutDescriptor(ctx, cashOutID, credential.AccountID, input.ExpectedRevision)
	if err != nil {
		writeWormPositionCashOutAuthorizationError(w, err)
		return
	}
	projection, err := server.authorizeWormPositionCashOutProof(ctx, wormPositionCashOutVerifiedProof{
		CashOutID: cashOutID, CommandID: commandID, ExpectedRevision: input.ExpectedRevision,
		AccountID: credential.AccountID, SessionJTIDigestSHA256: sessionDigest,
		AccessRevision: uint64(accessRevision), IntentDigestSHA256: descriptor.IntentDigestSHA256,
		ProofKind: wormPositionCashOutDevelopmentProofKind,
	})
	if err != nil {
		writeWormPositionCashOutAuthorizationError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, projection)
	observeWormAuthorization(request.Context(), "cash_out", cashOutID, "WORM_CASH_OUT_AUTHORIZE", "DEVELOPMENT", "authorization_verified", true, operationLogHTTPStatus(w))
	observed = true
}

func writeWormPositionCashOutAuthorizationError(w http.ResponseWriter, err error) {
	walletsecret.WriteError(w, stableWormPositionCashOutAuthorizationError(err))
}

func stableWormPositionCashOutAuthorizationError(err error) error {
	if err == nil {
		return nil
	}
	if reason := walletsecret.Reason(err); reason == "MODULE_ACCESS_CLOSED" || reason == "MODULE_ACCESS_UNAVAILABLE" {
		return err
	}
	code := status.Code(err)
	reason := googleoidc.WormPositionCashOutAuthorizationUnavailableReason
	stableCode := codes.Unavailable
	switch code {
	case codes.Unauthenticated:
		reason = googleoidc.WormPositionCashOutLoginSessionRequiredReason
		stableCode = codes.Unauthenticated
	case codes.InvalidArgument, codes.PermissionDenied, codes.NotFound, codes.AlreadyExists, codes.Aborted, codes.FailedPrecondition:
		reason = googleoidc.WormPositionCashOutAuthorizationRequiredReason
		stableCode = code
	case codes.ResourceExhausted:
		stableCode = codes.ResourceExhausted
	}
	errStatus := status.New(stableCode, reason)
	withDetails, detailErr := errStatus.WithDetails(&errdetails.ErrorInfo{
		Reason: reason, Domain: walletsecret.WormErrorDomain,
	})
	if detailErr != nil {
		return errStatus.Err()
	}
	return withDetails.Err()
}
