# Worm Client Tests

This package has two test layers:

- Unit tests in `worm_test.go` use `httptest` and do not call the real Worm API.
- The integration test in `worm_integration_test.go` calls the real public Worm API and is skipped by default.

## Run Unit Tests

Use this for normal local development and CI:

```bash
go test ./util/worm
```

These tests validate request construction, response decoding, signing behavior, and error handling against local fake HTTP servers.

## Create API Credentials From a Solana Private Key

`CreateAPIKeyFromPrivateKey` performs the Worm API key bootstrap flow:

1. request an auth challenge for the wallet derived from the private key;
2. sign the challenge message with the Solana keypair;
3. exchange the signed challenge for `WORM_API_KEY` and `WORM_API_SECRET`.

The private key input can be a Solana base58 keypair, a 128-character hex keypair, a Solana CLI JSON byte array, or a 32-byte seed in one of those encodings.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/useryege/athena/util/worm"
)

func main() {
	client, err := worm.NewClient(worm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	creds, err := client.CreateAPIKeyFromPrivateKey(context.Background(), os.Getenv("WORM_PRIVATE_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("export WORM_API_KEY=%q\n", creds.APIKey)
	fmt.Printf("export WORM_API_SECRET=%q\n", creds.Secret)
}
```

The secret is returned by Worm only once. Store it in a local environment variable, CI secret, or secret manager. Do not commit private keys, API secrets, `.env` files, or generated credentials to git.

## Run Public READ Integration Tests

The public READ integration tests are opt-in because they depend on the external Worm API and network availability:

```bash
WORM_INTEGRATION=1 go test -v ./util/worm -run '^TestIntegrationPublic'
```

The tests use the default base URL:

```text
https://api.worm.wtf
```

They only call public READ endpoints:

- `Search`
- `ListMarkets`
- `GetMarket`
- `GetMarketStats`
- `GetMarketPrice`
- `GetMarketOrderBook`
- `GetMarketCandles`
- `ListMarketTrades`
- `ListMarketMarginActivity`
- `ListEvents`
- `GetEvent`

They do not require `WORM_API_KEY` or `WORM_API_SECRET`, and they do not call account, order, redeem, position, submit, cancel, or other WRITE/authenticated endpoints.

Example successful output:

```text
=== RUN   TestIntegrationPublicGetMarketOrderBook
    worm_integration_test.go:155: GetMarketOrderBook returned market="..." is_yes=false bid_levels=0 ask_levels=0
--- PASS: TestIntegrationPublicGetMarketOrderBook (1.91s)
PASS
ok  	github.com/useryege/athena/util/worm	1.908s
```

If the integration tests fail, first check network connectivity and whether `https://api.worm.wtf` is reachable. Empty bid or ask levels, candles, trades, and margin activity rows are valid external API states; the tests only validate returned rows when the API returns them.

## Run Auth-Key Integration Tests

These authenticated integration tests cover the Worm auth-key bootstrap and key-management endpoints with a real Solana private key from the environment. They create real temporary Worm API keys and revoke the temporary keys they create.

It is protected by two opt-in switches:

```bash
WORM_INTEGRATION=1 \
WORM_AUTH_KEYS_INTEGRATION=1 \
WORM_PRIVATE_KEY='59mJJLBC22xe2Bg9mTozn47fYdwfeDkwswE9t8RFnmrNmn3Lr6bf3Abo8ua4GUpFdaEnikfLhrhAfkykWWwyoejN' \
go test -v ./util/worm -run '^TestIntegrationAuthKeys'
```

The suite covers:

- `CreateAuthChallenge`
- `CreateAPIKey`
- `CreateAPIKeyFromPrivateKey`
- `ListAPIKeys`
- `RevokeAPIKey`

The tests do not use `WORM_API_KEY` or `WORM_API_SECRET`, and they do not revoke user-provided credentials. They do not log the private key, signatures, request payloads, API secrets, or full credentials. The secret is returned by Worm only once; store any manually generated credentials securely.

## Run CreateOrderDraft Integration Test

This authenticated integration test bootstraps Worm API credentials from `WORM_PRIVATE_KEY`, selects one open market, then calls `CreateOrderDraft` with a market buy draft for YES using `funds="1.00"`. It may create a server-side order draft, but it does not submit the order and does not call `SubmitOrder`, `SubmitOrderCancel`, cancel, redeem, position, or other finalization endpoints.

It is protected by two opt-in switches:

```bash
WORM_INTEGRATION=1 \
WORM_ORDER_DRAFT_INTEGRATION=1 \
WORM_PRIVATE_KEY='59mJJLBC22xe2Bg9mTozn47fYdwfeDkwswE9t8RFnmrNmn3Lr6bf3Abo8ua4GUpFdaEnikfLhrhAfkykWWwyoejN' \
go test -v ./util/worm -run TestIntegrationCreateOrderDraft
```

The test fails if the private key is missing, no open market is returned, or the draft response does not include a pubkey or message. It does not log the private key, API key, API secret, signatures, or request payloads. The test creates a real Worm API key and the secret is returned only once; store generated credentials securely, or revoke the generated key if the test was only for validation.



## NOTES

vpn is required to run the integration test.

## Refferences

- [Worm API Reference](https://docs.worm.wtf/api-reference)
