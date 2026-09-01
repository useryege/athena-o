package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"math"
	"net/http"

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

const wormPositionCashOutBatchDevelopmentProofKind = "DEVELOPMENT"

type wormPositionCashOutBatchVerifiedProof struct {
	BatchID                string
	CommandID              string
	ExpectedRevision       int64
	AccountID              string
	SessionJTIDigestSHA256 []byte
	AccessRevision         uint64
	IntentDigestSHA256     []byte
	ProofKind              string
}

func (server *AthenaServer) enableWormPositionCashOutBatchAuthorization() error {
	authenticate := func(request *http.Request) (context.Context, accountcredentials.AuthenticatedCredential, error) {
		return server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelReadWrite)
	}
	if server.googleOIDC != nil {
		if err := server.googleOIDC.EnableWormPositionCashOutBatchAuthorization(
			server.RedisClient,
			authenticate,
			server.credentialMgr,
			server.loadGoogleWormPositionCashOutBatchDescriptor,
			server.authorizeGoogleWormPositionCashOutBatch,
			writeWormPositionCashOutBatchAuthorizationError,
		); err != nil {
			return err
		}
	}
	if server.phantomAuth != nil {
		if err := server.phantomAuth.EnableWormPositionCashOutBatchAuthorization(
			server.RedisClient,
			authenticate,
			server.credentialMgr,
			server.loadPhantomWormPositionCashOutBatchDescriptor,
			server.authorizePhantomWormPositionCashOutBatch,
			writeWormPositionCashOutBatchAuthorizationError,
		); err != nil {
			return err
		}
	}
	return nil
}

func (server *AthenaServer) authorizeGoogleWormPositionCashOutBatch(
	ctx context.Context,
	request googleoidc.WormPositionCashOutBatchAuthorizationRequest,
) (any, error) {
	return server.authorizeWormPositionCashOutBatchProof(ctx, wormPositionCashOutBatchVerifiedProof{
		BatchID: request.BatchID, CommandID: request.CommandID,
		ExpectedRevision: request.ExpectedRevision, AccountID: request.AccountID,
		SessionJTIDigestSHA256: request.SessionJTIDigestSHA256,
		AccessRevision:         request.AccessRevision, IntentDigestSHA256: request.IntentDigestSHA256,
		ProofKind: request.ProofKind,
	})
}

func (server *AthenaServer) authorizePhantomWormPositionCashOutBatch(
	ctx context.Context,
	request phantomauth.WormPositionCashOutBatchAuthorizationRequest,
) (any, error) {
	return server.authorizeWormPositionCashOutBatchProof(ctx, wormPositionCashOutBatchVerifiedProof{
		BatchID: request.BatchID, CommandID: request.CommandID,
		ExpectedRevision: request.ExpectedRevision, AccountID: request.AccountID,
		SessionJTIDigestSHA256: request.SessionJTIDigestSHA256,
		AccessRevision:         request.AccessRevision, IntentDigestSHA256: request.IntentDigestSHA256,
		ProofKind: request.ProofKind,
	})
}

func (server *AthenaServer) authorizeWormPositionCashOutBatchProof(
	ctx context.Context,
	request wormPositionCashOutBatchVerifiedProof,
) (wormPositionCashOutBatchResponse, error) {
	batchID, err := canonicalWormExecutionID(request.BatchID, "position Cash Out Batch ID")
	if err != nil || batchID != request.BatchID {
		return wormPositionCashOutBatchResponse{}, status.Error(codes.InvalidArgument, "position Cash Out Batch authorization binding is invalid")
	}
	commandID, err := canonicalWormExecutionID(request.CommandID, "commandId")
	if err != nil || commandID != request.CommandID {
		return wormPositionCashOutBatchResponse{}, status.Error(codes.InvalidArgument, "position Cash Out Batch authorization command is invalid")
	}
	accountID, err := accountcredentials.CanonicalAccountID(request.AccountID)
	if err != nil || accountID != request.AccountID || request.ExpectedRevision <= 0 ||
		len(request.SessionJTIDigestSHA256) != sha256.Size || len(request.IntentDigestSHA256) != sha256.Size ||
		request.AccessRevision == 0 || request.AccessRevision > math.MaxInt64 {
		return wormPositionCashOutBatchResponse{}, status.Error(codes.InvalidArgument, "position Cash Out Batch authorization account or intent is invalid")
	}
	if request.ProofKind != googleoidc.WormPositionCashOutBatchProofKindGoogle &&
		request.ProofKind != phantomauth.WormPositionCashOutBatchProofKindPhantom &&
		request.ProofKind != wormPositionCashOutBatchDevelopmentProofKind {
		return wormPositionCashOutBatchResponse{}, status.Error(codes.InvalidArgument, "position Cash Out Batch proof kind is invalid")
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		return wormPositionCashOutBatchResponse{}, err
	}
	result, err := client.AuthorizePositionCashOutBatch(ctx, &wormtradingapiclient.AuthorizePositionCashOutBatchRequest{
		OwnerAccountId: accountID, Id: batchID, CommandId: commandID,
		ExpectedRevision: request.ExpectedRevision, ProofKind: request.ProofKind,
		SessionJtiDigest: append([]byte(nil), request.SessionJTIDigestSHA256...),
		AccessRevision:   int64(request.AccessRevision),
	})
	if err != nil {
		return wormPositionCashOutBatchResponse{}, err
	}
	batch := result.GetBatch()
	if batch == nil || len(batch.GetIntentSha256()) != sha256.Size ||
		subtle.ConstantTimeCompare(batch.GetIntentSha256(), request.IntentDigestSHA256) != 1 {
		return wormPositionCashOutBatchResponse{}, status.Error(codes.Internal, "Worm Trading returned a mismatched Cash Out Batch intent")
	}
	return projectWormPositionCashOutBatch(batch, accountID, batchID)
}

func (server *AthenaServer) loadGoogleWormPositionCashOutBatchDescriptor(
	ctx context.Context,
	request googleoidc.WormPositionCashOutBatchDescriptorRequest,
) (googleoidc.WormPositionCashOutBatchDescriptor, error) {
	descriptor, err := server.loadWormPositionCashOutBatchDescriptor(ctx, request.BatchID, request.AccountID, request.ExpectedRevision)
	if err != nil {
		return googleoidc.WormPositionCashOutBatchDescriptor{}, err
	}
	return googleoidc.WormPositionCashOutBatchDescriptor{
		BatchID: descriptor.BatchID, Revision: descriptor.Revision,
		IntentDigestSHA256: descriptor.IntentDigestSHA256,
	}, nil
}

func (server *AthenaServer) loadPhantomWormPositionCashOutBatchDescriptor(
	ctx context.Context,
	request phantomauth.WormPositionCashOutBatchDescriptorRequest,
) (phantomauth.WormPositionCashOutBatchDescriptor, error) {
	descriptor, err := server.loadWormPositionCashOutBatchDescriptor(ctx, request.BatchID, request.AccountID, request.ExpectedRevision)
	if err != nil {
		return phantomauth.WormPositionCashOutBatchDescriptor{}, err
	}
	return phantomauth.WormPositionCashOutBatchDescriptor{
		BatchID: descriptor.BatchID, Revision: descriptor.Revision,
		IntentDigestSHA256: descriptor.IntentDigestSHA256,
	}, nil
}

type wormPositionCashOutBatchDescriptor struct {
	BatchID            string
	Revision           int64
	IntentDigestSHA256 []byte
}

func (server *AthenaServer) loadWormPositionCashOutBatchDescriptor(
	ctx context.Context,
	batchID string,
	accountID string,
	expectedRevision int64,
) (wormPositionCashOutBatchDescriptor, error) {
	if expectedRevision <= 0 {
		return wormPositionCashOutBatchDescriptor{}, status.Error(codes.InvalidArgument, "position Cash Out Batch revision is invalid")
	}
	canonicalID, err := canonicalWormExecutionID(batchID, "position Cash Out Batch ID")
	if err != nil || canonicalID != batchID {
		return wormPositionCashOutBatchDescriptor{}, status.Error(codes.InvalidArgument, "position Cash Out Batch ID is invalid")
	}
	canonicalAccountID, err := accountcredentials.CanonicalAccountID(accountID)
	if err != nil || canonicalAccountID != accountID {
		return wormPositionCashOutBatchDescriptor{}, status.Error(codes.InvalidArgument, "position Cash Out Batch account is invalid")
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		return wormPositionCashOutBatchDescriptor{}, err
	}
	result, err := client.GetPositionCashOutBatch(ctx, &wormtradingapiclient.GetPositionCashOutBatchRequest{
		OwnerAccountId: accountID, Id: batchID,
	})
	if err != nil {
		return wormPositionCashOutBatchDescriptor{}, err
	}
	batch := result.GetBatch()
	projection, err := projectWormPositionCashOutBatch(batch, accountID, batchID)
	if err != nil {
		return wormPositionCashOutBatchDescriptor{}, err
	}
	authorizableState := batch != nil &&
		(batch.GetState() == "AWAITING_AUTHORIZATION" ||
			(batch.GetState() == "PAUSED" && batch.GetCurrentItem() == nil))
	if projection.Revision != expectedRevision || !authorizableState ||
		batch == nil || len(batch.GetIntentSha256()) != sha256.Size {
		return wormPositionCashOutBatchDescriptor{}, status.Error(codes.FailedPrecondition, "position Cash Out Batch cannot be authorized")
	}
	return wormPositionCashOutBatchDescriptor{
		BatchID: batchID, Revision: projection.Revision,
		IntentDigestSHA256: append([]byte(nil), batch.GetIntentSha256()...),
	}, nil
}

func (server *AthenaServer) developmentWormPositionCashOutBatchAuthorization(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	if request.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !server.DisableAuth || !requestIsLoopback(request) || !server.validWormConnectionOrigin(request) {
		writeWormPositionCashOutBatchAuthorizationError(w, walletsecret.ErrWormLoginSessionRequired)
		return
	}
	if err := rejectWormCombinationQuery(request); err != nil {
		writeWormPositionCashOutBatchAuthorizationError(w, err)
		return
	}
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelReadWrite)
	if err != nil {
		writeWormPositionCashOutBatchAuthorizationError(w, err)
		return
	}
	batchID, err := canonicalWormExecutionID(request.PathValue("batchId"), "position Cash Out Batch ID")
	if err != nil {
		writeWormPositionCashOutBatchAuthorizationError(w, err)
		return
	}
	var input wormPositionCashOutBatchCommandInput
	if err := decodeWormExecutionJSON(w, request, &input); err != nil {
		writeWormPositionCashOutBatchAuthorizationError(w, err)
		return
	}
	commandID, err := canonicalWormExecutionID(input.CommandID, "commandId")
	if err != nil || validateWormExecutionExpectedRevision(input.ExpectedRevision) != nil {
		writeWormPositionCashOutBatchAuthorizationError(w, status.Error(codes.InvalidArgument, "position Cash Out Batch authorization binding is invalid"))
		return
	}
	sessionDigest, accessRevision, err := wormExecutionCredentialBinding(credential)
	if err != nil {
		writeWormPositionCashOutBatchAuthorizationError(w, err)
		return
	}
	descriptor, err := server.loadWormPositionCashOutBatchDescriptor(ctx, batchID, credential.AccountID, input.ExpectedRevision)
	if err != nil {
		writeWormPositionCashOutBatchAuthorizationError(w, err)
		return
	}
	projection, err := server.authorizeWormPositionCashOutBatchProof(ctx, wormPositionCashOutBatchVerifiedProof{
		BatchID: batchID, CommandID: commandID, ExpectedRevision: input.ExpectedRevision,
		AccountID: credential.AccountID, SessionJTIDigestSHA256: sessionDigest,
		AccessRevision: uint64(accessRevision), IntentDigestSHA256: descriptor.IntentDigestSHA256,
		ProofKind: wormPositionCashOutBatchDevelopmentProofKind,
	})
	if err != nil {
		writeWormPositionCashOutBatchAuthorizationError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, projection)
}

func writeWormPositionCashOutBatchAuthorizationError(w http.ResponseWriter, err error) {
	walletsecret.WriteError(w, stableWormPositionCashOutBatchAuthorizationError(err))
}

func stableWormPositionCashOutBatchAuthorizationError(err error) error {
	if err == nil {
		return nil
	}
	code := status.Code(err)
	reason := googleoidc.WormPositionCashOutBatchAuthorizationUnavailableReason
	stableCode := codes.Unavailable
	switch code {
	case codes.Unauthenticated:
		reason = googleoidc.WormPositionCashOutBatchLoginSessionRequiredReason
		stableCode = codes.Unauthenticated
	case codes.InvalidArgument, codes.PermissionDenied, codes.NotFound, codes.AlreadyExists, codes.Aborted, codes.FailedPrecondition:
		reason = googleoidc.WormPositionCashOutBatchAuthorizationRequiredReason
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
