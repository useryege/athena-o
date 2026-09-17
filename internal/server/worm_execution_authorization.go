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

const wormExecutionDevelopmentProofKind = "DEVELOPMENT"

type wormExecutionVerifiedProof struct {
	RunID                  string
	CommandID              string
	ExpectedRevision       int64
	AccountID              string
	SessionJTIDigestSHA256 []byte
	AccessRevision         uint64
	PlanDigestSHA256       []byte
	ProofKind              string
}

func (server *AthenaServer) enableWormExecutionAuthorization() error {
	if server.googleOIDC != nil {
		if err := server.googleOIDC.EnableWormExecutionAuthorization(
			server.RedisClient,
			func(request *http.Request) (context.Context, accountcredentials.AuthenticatedCredential, error) {
				return server.authenticateWormTradingIdentityHTTP(request, accountaccess.AccessLevelReadWrite)
			},
			server.admitWormAccess,
			server.credentialMgr,
			server.authorizeGoogleWormExecution,
			writeWormExecutionAuthorizationError,
		); err != nil {
			return err
		}
	}
	if server.phantomAuth != nil {
		if err := server.phantomAuth.EnableWormExecutionAuthorization(
			server.RedisClient,
			func(request *http.Request) (context.Context, accountcredentials.AuthenticatedCredential, error) {
				return server.authenticateWormTradingIdentityHTTP(request, accountaccess.AccessLevelReadWrite)
			},
			server.admitWormAccess,
			server.credentialMgr,
			server.loadPhantomWormExecutionDescriptor,
			server.authorizePhantomWormExecution,
			writeWormExecutionAuthorizationError,
		); err != nil {
			return err
		}
	}
	return nil
}

func (server *AthenaServer) authorizeGoogleWormExecution(
	ctx context.Context,
	request googleoidc.WormExecutionAuthorizationRequest,
) (any, error) {
	return server.authorizeWormExecutionProof(ctx, wormExecutionVerifiedProof{
		RunID:                  request.RunID,
		CommandID:              request.CommandID,
		ExpectedRevision:       request.ExpectedRevision,
		AccountID:              request.AccountID,
		SessionJTIDigestSHA256: request.SessionJTIDigestSHA256,
		AccessRevision:         request.AccessRevision,
		ProofKind:              request.ProofKind,
	})
}

func (server *AthenaServer) authorizePhantomWormExecution(
	ctx context.Context,
	request phantomauth.WormExecutionAuthorizationRequest,
) (any, error) {
	return server.authorizeWormExecutionProof(ctx, wormExecutionVerifiedProof{
		RunID:                  request.RunID,
		CommandID:              request.CommandID,
		ExpectedRevision:       request.ExpectedRevision,
		AccountID:              request.AccountID,
		SessionJTIDigestSHA256: request.SessionJTIDigestSHA256,
		AccessRevision:         request.AccessRevision,
		PlanDigestSHA256:       request.PlanDigestSHA256,
		ProofKind:              request.ProofKind,
	})
}

func (server *AthenaServer) authorizeWormExecutionProof(
	ctx context.Context,
	request wormExecutionVerifiedProof,
) (wormExecutionRunResponse, error) {
	runID, err := canonicalWormExecutionID(request.RunID, "execution run ID")
	if err != nil || runID != request.RunID {
		return wormExecutionRunResponse{}, status.Error(codes.InvalidArgument, "execution authorization run binding is invalid")
	}
	commandID, err := canonicalWormExecutionID(request.CommandID, "commandId")
	if err != nil || commandID != request.CommandID {
		return wormExecutionRunResponse{}, status.Error(codes.InvalidArgument, "execution authorization command binding is invalid")
	}
	accountID, err := accountcredentials.CanonicalAccountID(request.AccountID)
	if err != nil || accountID != request.AccountID || request.ExpectedRevision <= 0 ||
		len(request.SessionJTIDigestSHA256) != sha256.Size || request.AccessRevision == 0 || request.AccessRevision > math.MaxInt64 {
		return wormExecutionRunResponse{}, status.Error(codes.InvalidArgument, "execution authorization account or session binding is invalid")
	}
	if request.ProofKind != googleoidc.WormExecutionProofKindGoogle &&
		request.ProofKind != phantomauth.WormExecutionProofKindPhantom &&
		request.ProofKind != wormExecutionDevelopmentProofKind {
		return wormExecutionRunResponse{}, status.Error(codes.InvalidArgument, "execution authorization proof kind is invalid")
	}
	if request.ProofKind == phantomauth.WormExecutionProofKindPhantom && len(request.PlanDigestSHA256) != sha256.Size {
		return wormExecutionRunResponse{}, status.Error(codes.InvalidArgument, "execution authorization plan binding is invalid")
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		return wormExecutionRunResponse{}, err
	}
	result, err := client.AuthorizeExecutionRun(ctx, &wormtradingapiclient.AuthorizeExecutionRunRequest{
		OwnerAccountId:   accountID,
		Id:               runID,
		CommandId:        commandID,
		ExpectedRevision: request.ExpectedRevision,
		ProofKind:        request.ProofKind,
		SessionJtiDigest: append([]byte(nil), request.SessionJTIDigestSHA256...),
		AccessRevision:   int64(request.AccessRevision),
	})
	if err != nil {
		return wormExecutionRunResponse{}, err
	}
	run := result.GetRun()
	if request.ProofKind == phantomauth.WormExecutionProofKindPhantom &&
		(run == nil || len(run.GetPlanDigestSha256()) != sha256.Size ||
			subtle.ConstantTimeCompare(run.GetPlanDigestSha256(), request.PlanDigestSHA256) != 1) {
		return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading returned a mismatched execution plan binding")
	}
	return projectWormExecutionRunWithBinding(
		run,
		accountID,
		request.SessionJTIDigestSHA256,
		int64(request.AccessRevision),
		runID,
	)
}

func (server *AthenaServer) loadPhantomWormExecutionDescriptor(
	ctx context.Context,
	request phantomauth.WormExecutionDescriptorRequest,
) (phantomauth.WormExecutionDescriptor, error) {
	if request.ExpectedRevision <= 0 || len(request.SessionJTIDigestSHA256) != sha256.Size ||
		request.AccessRevision == 0 || request.AccessRevision > math.MaxInt64 {
		return phantomauth.WormExecutionDescriptor{}, status.Error(codes.InvalidArgument, "execution authorization descriptor binding is invalid")
	}
	runID, err := canonicalWormExecutionID(request.RunID, "execution run ID")
	if err != nil || runID != request.RunID {
		return phantomauth.WormExecutionDescriptor{}, status.Error(codes.InvalidArgument, "execution authorization run binding is invalid")
	}
	accountID, err := accountcredentials.CanonicalAccountID(request.AccountID)
	if err != nil || accountID != request.AccountID {
		return phantomauth.WormExecutionDescriptor{}, status.Error(codes.InvalidArgument, "execution authorization account binding is invalid")
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		return phantomauth.WormExecutionDescriptor{}, err
	}
	result, err := client.GetExecutionRun(ctx, &wormtradingapiclient.GetExecutionRunRequest{OwnerAccountId: accountID, Id: runID})
	if err != nil {
		return phantomauth.WormExecutionDescriptor{}, err
	}
	run := result.GetRun()
	projection, err := projectWormExecutionRunWithBinding(
		run,
		accountID,
		request.SessionJTIDigestSHA256,
		int64(request.AccessRevision),
		runID,
	)
	if err != nil {
		return phantomauth.WormExecutionDescriptor{}, err
	}
	if projection.Revision != request.ExpectedRevision {
		return phantomauth.WormExecutionDescriptor{}, status.Error(codes.Aborted, "execution run revision changed")
	}
	if !slices.Contains(projection.AllowedActions, "AUTHORIZE") || run == nil || len(run.GetPlanDigestSha256()) != sha256.Size {
		return phantomauth.WormExecutionDescriptor{}, status.Error(codes.FailedPrecondition, "execution run cannot be authorized")
	}
	return phantomauth.WormExecutionDescriptor{
		RunID:            runID,
		Revision:         projection.Revision,
		PlanDigestSHA256: append([]byte(nil), run.GetPlanDigestSha256()...),
	}, nil
}

func (server *AthenaServer) developmentWormExecutionAuthorization(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	if request.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !server.DisableAuth || !requestIsLoopback(request) || !server.validWormConnectionOrigin(request) {
		writeWormExecutionAuthorizationError(w, walletsecret.ErrWormLoginSessionRequired)
		return
	}
	if err := rejectWormCombinationQuery(request); err != nil {
		writeWormExecutionAuthorizationError(w, err)
		return
	}
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelReadWrite)
	if err != nil {
		writeWormExecutionAuthorizationError(w, err)
		return
	}
	runID, err := canonicalWormExecutionID(request.PathValue("runId"), "execution run ID")
	if err != nil {
		writeWormExecutionAuthorizationError(w, err)
		return
	}
	var input wormExecutionRevisionCommandInput
	if err := decodeWormExecutionJSON(w, request, &input); err != nil {
		writeWormExecutionAuthorizationError(w, err)
		return
	}
	commandID, err := canonicalWormExecutionID(input.CommandID, "commandId")
	if err != nil || validateWormExecutionExpectedRevision(input.ExpectedRevision) != nil {
		writeWormExecutionAuthorizationError(w, status.Error(codes.InvalidArgument, "execution authorization command binding is invalid"))
		return
	}
	sessionDigest, accessRevision, err := wormExecutionCredentialBinding(credential)
	if err != nil {
		writeWormExecutionAuthorizationError(w, err)
		return
	}
	projection, err := server.authorizeWormExecutionProof(ctx, wormExecutionVerifiedProof{
		RunID:                  runID,
		CommandID:              commandID,
		ExpectedRevision:       input.ExpectedRevision,
		AccountID:              credential.AccountID,
		SessionJTIDigestSHA256: sessionDigest,
		AccessRevision:         uint64(accessRevision),
		ProofKind:              wormExecutionDevelopmentProofKind,
	})
	if err != nil {
		writeWormExecutionAuthorizationError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, projection)
}

func writeWormExecutionAuthorizationError(w http.ResponseWriter, err error) {
	walletsecret.WriteError(w, stableWormExecutionAuthorizationError(err))
}

func stableWormExecutionAuthorizationError(err error) error {
	if err == nil {
		return nil
	}
	if reason := walletsecret.Reason(err); reason == "MODULE_ACCESS_CLOSED" || reason == "MODULE_ACCESS_UNAVAILABLE" {
		return err
	}
	code := status.Code(err)
	reason := googleoidc.WormExecutionAuthorizationUnavailableReason
	stableCode := codes.Unavailable
	switch code {
	case codes.Unauthenticated:
		reason = googleoidc.WormExecutionLoginSessionRequiredReason
		stableCode = codes.Unauthenticated
	case codes.InvalidArgument, codes.PermissionDenied, codes.NotFound, codes.AlreadyExists, codes.Aborted, codes.FailedPrecondition:
		reason = googleoidc.WormExecutionAuthorizationRequiredReason
		stableCode = code
	case codes.ResourceExhausted:
		stableCode = codes.ResourceExhausted
	}
	errStatus := status.New(stableCode, reason)
	withDetails, detailErr := errStatus.WithDetails(&errdetails.ErrorInfo{Reason: reason, Domain: walletsecret.WormErrorDomain})
	if detailErr != nil {
		return errStatus.Err()
	}
	return withDetails.Err()
}
