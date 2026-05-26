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
3. exchange the signed challenge for a Worm API key and API secret.

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
WORM_PRIVATE_KEY='<your-solana-private-key>' \
go test -v ./util/worm -run '^TestIntegrationAuthKeys'
```

The suite covers:

- `CreateAuthChallenge`
- `CreateAPIKey`
- `CreateAPIKeyFromPrivateKey`
- `ListAPIKeys`
- `RevokeAPIKey`

The tests do not use `WORM_API_KEY` or `WORM_API_SECRET`, and they do not revoke user-provided credentials. They do not log the private key, signatures, request payloads, API secrets, or full credentials. The secret is returned by Worm only once; store any manually generated credentials securely.

## Run Authenticated READ Integration Tests

These authenticated read-only integration tests cover account-scoped Worm API endpoints. They do not create or cancel orders, create or close positions, set or delete TP/SL, submit signatures, claim settlements, create redeems, or finalize anything on chain.

They are protected by two opt-in switches and can use existing API credentials:

```bash
WORM_INTEGRATION=1 \
WORM_AUTH_READ_INTEGRATION=1 \
WORM_API_KEY='<your-worm-api-key>' \
WORM_API_SECRET='<your-worm-api-secret>' \
go test -v ./util/worm -run '^TestIntegrationAuthRead'
```

If `WORM_API_KEY` and `WORM_API_SECRET` are not set, the tests can bootstrap a temporary API key from a Solana private key and revoke that temporary key during cleanup:

```bash
WORM_INTEGRATION=1 \
WORM_AUTH_READ_INTEGRATION=1 \
WORM_PRIVATE_KEY='<your-solana-private-key>' \
go test -v ./util/worm -run '^TestIntegrationAuthRead'
```

The suite covers:

- `ListTrades`
- `ListOrders`
- `GetOrder`, only when the authenticated account already has at least one order
- `GetAccountSummary`
- `GetAccountPnL`
- `ListAccountAssets`
- `ListRedeems`
- `GetRedeem`, only when the authenticated account already has at least one redeem
- `EstimateMarginPosition`, using an open margin-enabled market when one is returned by the public market list
- `ListPositionRequests`
- `GetPositionRequest`, only when the authenticated account already has at least one position request
- `ListMarginPositions`
- `GetMarginPosition`, only when the authenticated account already has at least one margin position
- `ListMarginSettlements`

Empty trades, orders, assets, redeems, position requests, margin positions, and settlements are valid account states; the tests only validate row fields when the API returns rows. Detail tests derive a pubkey from the corresponding list response and skip when there is no existing row to fetch. `StartRedeem`, `SubmitRedeem`, order submit/cancel, position create/submit/cancel/close, TP/SL changes, and settlement claims are intentionally excluded because they create, mutate, or finalize real state and require a stronger opt-in test plan.

## Run EstimateMarginPosition Integration Test

This public read-only integration test calls `EstimateMarginPosition` with parameters from the environment. Use it before a leverage submit test to confirm a market, side, funds, and leverage combination can be estimated and to inspect whether Worm reports `is_fully_filled=true`.

It does not require `WORM_PRIVATE_KEY`, does not create API credentials, does not create a draft, and does not submit an order.

It is protected by two opt-in switches:

```bash
WORM_INTEGRATION=1 \
WORM_MARGIN_ESTIMATE_INTEGRATION=1 \
WORM_MARGIN_ESTIMATE_MARKET_CONDITION_ID='4YnAc9NUg1beqUafykRLP7hKVVpHZmJGGDhW7Ei81cYW' \
WORM_MARGIN_ESTIMATE_IS_YES='true' \
WORM_MARGIN_ESTIMATE_LEVERAGE='2' \
WORM_MARGIN_ESTIMATE_FUNDS='6' \
go test -v ./util/worm -run '^TestIntegrationEstimateMarginPositionFromEnv$'
```

Required environment variables:

- `WORM_MARGIN_ESTIMATE_MARKET_CONDITION_ID`: target market condition id.
- `WORM_MARGIN_ESTIMATE_IS_YES`: `true` or `false`.
- `WORM_MARGIN_ESTIMATE_LEVERAGE`: leverage multiplier, such as `2` or `2.5`.
- `WORM_MARGIN_ESTIMATE_FUNDS`: funds amount, for example `6`.

The test validates the estimate response fields and logs `average_price`, `total_shares`, `total_cost`, `user_funds_needed`, `is_fully_filled`, and whether `liquidation_price` is present. `is_fully_filled=false` does not fail this estimate-only test, but that parameter set should not be used for a real submit test.

## Run CreatePositionRequest Integration Test

This authenticated integration test creates a real Worm margin position request draft from environment variables. It bootstraps temporary API credentials from `WORM_PRIVATE_KEY`, calls `CreatePositionRequest`, verifies the draft can be fetched with `GetPositionRequest`, then attempts to cancel the draft with `CancelPositionRequest`.

It does not sign the draft message and does not call `SubmitPositionRequest`, so it should not open a real position. It may still create server-side draft state, so it has a separate opt-in switch and cleanup.

It is protected by two opt-in switches:

```bash
WORM_INTEGRATION=1 \
WORM_POSITION_REQUEST_DRAFT_INTEGRATION=1 \
WORM_PRIVATE_KEY='<your-solana-private-key>' \
WORM_POSITION_REQUEST_MARKET_CONDITION_ID='4YnAc9NUg1beqUafykRLP7hKVVpHZmJGGDhW7Ei81cYW' \
WORM_POSITION_REQUEST_TYPE='MARKET' \
WORM_POSITION_REQUEST_IS_YES='true' \
WORM_POSITION_REQUEST_LEVERAGE='2' \
WORM_POSITION_REQUEST_FUNDS='6' \
go test -v ./util/worm -run '^TestIntegrationCreatePositionRequestFromEnv$'
```

Required environment variables:

- `WORM_PRIVATE_KEY`: Solana base58 keypair, 128-character hex keypair, Solana CLI JSON byte array, or 32-byte seed in one of those encodings.
- `WORM_POSITION_REQUEST_MARKET_CONDITION_ID`: target market condition id.
- `WORM_POSITION_REQUEST_TYPE`: `MARKET` or `LIMIT`.
- `WORM_POSITION_REQUEST_IS_YES`: `true` or `false`.
- `WORM_POSITION_REQUEST_LEVERAGE`: leverage multiplier, such as `2` or `2.5`.

For `MARKET` drafts, also set:

- `WORM_POSITION_REQUEST_FUNDS`: funds amount, for example `6`.

For `LIMIT` drafts, set these instead:

- `WORM_POSITION_REQUEST_PRICE`: limit entry price.
- `WORM_POSITION_REQUEST_SHARES`: share size.

Optional take-profit and stop-loss settings:

- `WORM_POSITION_REQUEST_TAKE_PROFIT_PRICE`
- `WORM_POSITION_REQUEST_STOP_LOSS_PRICE`

The test logs only safe status information, such as whether `pubkey` and `message` are present, message length, state, funds, and whether cleanup cancellation returned tx ids. It does not log the private key, API secret, draft message text, signatures, or full request payload.

## Run SubmitPositionRequest Integration Test

This authenticated integration test creates a real Worm margin position request draft, signs the draft `message` with `WORM_PRIVATE_KEY`, and calls `SubmitPositionRequest`. It may open or fund a real leveraged position. Run it only with a rotated test wallet and parameters you intend to submit.

It does not cancel the position request after submit because the request may already be funding, processing, completed, or otherwise finalized. Cleanup only revokes the temporary Worm API key created for the test.

It is protected by two opt-in switches:

```bash
WORM_INTEGRATION=1 \
WORM_POSITION_REQUEST_SUBMIT_INTEGRATION=1 \
WORM_PRIVATE_KEY='<your-solana-private-key>' \
WORM_POSITION_REQUEST_MARKET_CONDITION_ID='4YnAc9NUg1beqUafykRLP7hKVVpHZmJGGDhW7Ei81cYW' \
WORM_POSITION_REQUEST_TYPE='MARKET' \
WORM_POSITION_REQUEST_IS_YES='true' \
WORM_POSITION_REQUEST_LEVERAGE='2' \
WORM_POSITION_REQUEST_FUNDS='6' \
go test -v ./util/worm -run '^TestIntegrationSubmitPositionRequestFromEnv$'
```

The test uses the same `WORM_POSITION_REQUEST_*` parameters as the draft test. For `MARKET` requests it first calls `EstimateMarginPosition` and requires `is_fully_filled=true` before creating credentials or drafts.

On submit failure, the test logs the structured Worm API error and fetches the created position request to log safe lifecycle fields such as `state`, tx id presence, message length, funds, and market id. It does not log the private key, API secret, draft message text, signature, or full request payload.

## Run CreateOrderDraft Integration Test

This authenticated integration test bootstraps Worm API credentials from `WORM_PRIVATE_KEY`, selects one open market, then calls `CreateOrderDraft` with a market buy draft for YES using `funds="1.00"`. It may create a server-side order draft, but it does not submit the order and does not call `SubmitOrder`, `SubmitOrderCancel`, cancel, redeem, position, or other finalization endpoints.

It is protected by two opt-in switches:

```bash
WORM_INTEGRATION=1 \
WORM_ORDER_DRAFT_INTEGRATION=1 \
WORM_PRIVATE_KEY='<your-solana-private-key>' \
go test -v ./util/worm -run TestIntegrationCreateOrderDraft
```

The test fails if the private key is missing, no open market is returned, or the draft response does not include a pubkey or message. It does not log the private key, API key, API secret, signatures, or request payloads. The test creates a real Worm API key and the secret is returned only once; store generated credentials securely, or revoke the generated key if the test was only for validation.



## NOTES

vpn is required to run the integration test.

## Refferences

- [Worm API Reference](https://docs.worm.wtf/api-reference)
