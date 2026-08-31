package worm

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"time"
)

// WebSignInClient is the shared Worm Web authentication surface used by
// higher-level market-position operations.
type WebSignInClient interface {
	GetSignInChallenge(ctx context.Context, walletAddress string) (*WebSignInChallenge, error)
	SignIn(ctx context.Context, request WebSignInRequest) (*WebSignInResponse, error)
}

// WebSignInSigner signs only the exact Worm Web sign-in message. Trading
// operations that also require a Solana transaction signature expose a wider
// signer interface.
type WebSignInSigner interface {
	SignWebSignInMessage(
		ctx context.Context,
		request WebSignInMessageSigningRequest,
	) (WebSignInMessageSigningResponse, error)
}

type WebSignInMessageSigningRequest struct {
	WalletAddress string
	Nonce         string
	Message       string
	MessageSHA256 [sha256.Size]byte
}

type WebSignInMessageSigningResponse struct {
	Signature string
}

func authenticateWebWallet(
	ctx context.Context,
	client WebSignInClient,
	signer WebSignInSigner,
	walletAddress string,
) (string, error) {
	challenge, err := client.GetSignInChallenge(ctx, walletAddress)
	if err != nil {
		return "", err
	}
	if challenge == nil {
		return "", errors.New("Worm Web sign-in challenge returned no nonce")
	}
	if err := validateWebSignInNonce(challenge.Nonce); err != nil {
		return "", err
	}

	message := BuildWebSignInMessage(walletAddress, challenge.Nonce, time.Now())
	messageDigest := sha256.Sum256([]byte(message))
	signedMessage, err := signer.SignWebSignInMessage(ctx, WebSignInMessageSigningRequest{
		WalletAddress: walletAddress,
		Nonce:         challenge.Nonce,
		Message:       message,
		MessageSHA256: messageDigest,
	})
	if err != nil {
		return "", err
	}
	canonicalSignInSignature, err := validateWebSignInSignature(
		walletAddress,
		message,
		signedMessage.Signature,
	)
	if err != nil {
		return "", err
	}

	signIn, err := client.SignIn(ctx, WebSignInRequest{
		Message:   message,
		Signature: canonicalSignInSignature,
		Address:   walletAddress,
		Nonce:     challenge.Nonce,
	})
	if err != nil {
		return "", err
	}
	if signIn == nil || strings.TrimSpace(signIn.AccessToken) == "" {
		return "", errors.New("Worm Web sign-in returned no access token")
	}
	return signIn.AccessToken, nil
}
