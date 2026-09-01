package wormtrading

import (
	"bytes"
	"context"
	"errors"
	"strings"

	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	utilworm "github.com/useryege/athena/util/worm"
)

// executionWebSigner adapts the capability-scoped Wallet RPCs to util/worm's
// protocol signer interfaces. The adapter binds every request to the frozen
// Run, Step, Wallet, owner, and intent; util/worm independently verifies the
// returned cryptographic material before it can become a Finalize command.
type executionWebSigner struct {
	service *Service
	task    *executionWorkerTask
}

func newExecutionWebSigner(service *Service, task *executionWorkerTask) (*executionWebSigner, error) {
	if service == nil || task == nil || task.run.ID == "" || task.step.ID == "" ||
		task.wallet.WalletID <= 0 || strings.TrimSpace(task.wallet.Address) == "" || len(task.intent) == 0 {
		return nil, errors.New("execution Web signer binding is incomplete")
	}
	return &executionWebSigner{service: service, task: task}, nil
}

func (s *executionWebSigner) SignWebSignInMessage(
	ctx context.Context,
	request utilworm.WebSignInMessageSigningRequest,
) (utilworm.WebSignInMessageSigningResponse, error) {
	if s == nil || s.service == nil || s.task == nil || request.WalletAddress != s.task.wallet.Address {
		return utilworm.WebSignInMessageSigningResponse{}, &executionWorkerFailure{
			code:  executionWorkerReasonWalletSignerInvalid,
			cause: errors.New("execution Web sign-in signer binding changed"),
		}
	}

	signerCtx, cancel := context.WithTimeout(ctx, s.service.wormPositionBudget)
	defer cancel()
	response, err := s.service.walletSignerClientset.Signer().SignWormWebSignInMessage(
		signerCtx,
		&walletapiclient.SignWormWebSignInMessageRequest{
			Id: s.task.wallet.WalletID, RequesterAccountId: s.task.run.OwnerAccountID,
			ExpectedAddress: request.WalletAddress, Nonce: request.Nonce, Message: request.Message,
			ExpectedMessageSha256: request.MessageSHA256[:], ExecutionRunId: s.task.run.ID,
			ExecutionStepId: s.task.step.ID, IntentSha256: append([]byte(nil), s.task.intent...),
		},
	)
	if err != nil {
		return utilworm.WebSignInMessageSigningResponse{}, executionWorkerSignerFailure(err)
	}
	if response == nil || strings.TrimSpace(response.GetSignature()) == "" ||
		!bytes.Equal(response.GetMessageSha256(), request.MessageSHA256[:]) ||
		response.GetExecutionRunId() != s.task.run.ID || response.GetExecutionStepId() != s.task.step.ID ||
		!bytes.Equal(response.GetIntentSha256(), s.task.intent) {
		return utilworm.WebSignInMessageSigningResponse{}, &executionWorkerFailure{
			code:  executionWorkerReasonWalletSignerInvalid,
			cause: errors.New("Wallet returned an invalid Worm Web sign-in response"),
		}
	}
	return utilworm.WebSignInMessageSigningResponse{Signature: response.GetSignature()}, nil
}

func (s *executionWebSigner) SignWebPositionTransaction(
	ctx context.Context,
	request utilworm.WebPositionTransactionSigningRequest,
) (utilworm.WebPositionTransactionSigningResponse, error) {
	if s == nil || s.service == nil || s.task == nil || request.WalletAddress != s.task.wallet.Address ||
		request.PositionRequestID <= 0 || request.PositionRequestID != s.task.step.PositionRequestID ||
		!bytes.Equal(request.TransactionSHA256[:], s.task.step.TransactionMessageSHA256) {
		return utilworm.WebPositionTransactionSigningResponse{}, &executionWorkerFailure{
			code:  executionWorkerReasonWalletSignerInvalid,
			cause: errors.New("execution Web transaction signer binding changed"),
		}
	}

	signerCtx, cancel := context.WithTimeout(ctx, s.service.wormPositionBudget)
	defer cancel()
	response, err := s.service.walletSignerClientset.Signer().SignWormPositionRequestTransaction(
		signerCtx,
		&walletapiclient.SignWormPositionRequestTransactionRequest{
			Id: s.task.wallet.WalletID, RequesterAccountId: s.task.run.OwnerAccountID,
			ExpectedAddress: request.WalletAddress, PositionRequestId: request.PositionRequestID,
			TransactionHex: request.TransactionHex, ExpectedTransactionSha256: request.TransactionSHA256[:],
			ExecutionRunId: s.task.run.ID, ExecutionStepId: s.task.step.ID,
			IntentSha256: append([]byte(nil), s.task.intent...),
		},
	)
	if err != nil {
		return utilworm.WebPositionTransactionSigningResponse{}, executionWorkerSignerFailure(err)
	}
	if response == nil || !bytes.Equal(response.GetTransactionSha256(), request.TransactionSHA256[:]) ||
		response.GetExecutionRunId() != s.task.run.ID || response.GetExecutionStepId() != s.task.step.ID ||
		!bytes.Equal(response.GetIntentSha256(), s.task.intent) ||
		response.GetPositionRequestId() != request.PositionRequestID ||
		strings.TrimSpace(response.GetTransactionVersion()) == "" ||
		response.GetRequiredSignatures() <= 0 || response.GetSignerIndex() < 0 ||
		response.GetSignerIndex() >= response.GetRequiredSignatures() {
		return utilworm.WebPositionTransactionSigningResponse{}, &executionWorkerFailure{
			code:  executionWorkerReasonWalletSignerInvalid,
			cause: errors.New("Wallet returned an invalid Worm Web transaction response"),
		}
	}

	var payload utilworm.WebPositionFinalizePayload
	switch value := response.GetFinalizePayload().(type) {
	case *walletapiclient.SignWormPositionRequestTransactionResponse_Signature:
		payload = utilworm.WebPositionFinalizePayload{
			Mode:  utilworm.WebFinalizeModeSignature,
			Value: value.Signature,
		}
	case *walletapiclient.SignWormPositionRequestTransactionResponse_SignedTransaction:
		payload = utilworm.WebPositionFinalizePayload{
			Mode:  utilworm.WebFinalizeModeSignedTransaction,
			Value: value.SignedTransaction,
		}
	default:
		return utilworm.WebPositionTransactionSigningResponse{}, &executionWorkerFailure{
			code:  executionWorkerReasonWalletSignerInvalid,
			cause: errors.New("Wallet omitted the Worm Web finalize payload"),
		}
	}
	return utilworm.WebPositionTransactionSigningResponse{Payload: payload}, nil
}

var _ utilworm.WebMarketPositionSigner = (*executionWebSigner)(nil)
