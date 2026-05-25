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

## Run Public READ Integration Test

The integration test is opt-in because it depends on the external Worm API and network availability:

```bash
WORM_INTEGRATION=1 go test -v ./util/worm -run TestIntegrationPublicReadFlow
```

The test uses the default base URL:

```text
https://api.worm.wtf
```

It only calls public READ endpoints:

- `ListMarkets`
- `GetMarket`
- `GetMarketPrice`
- `GetMarketOrderBook`

It does not require `WORM_API_KEY` or `WORM_API_SECRET`, and it does not call account, order, redeem, position, submit, cancel, or other WRITE/authenticated endpoints.

Example successful output:

```text
=== RUN   TestIntegrationPublicReadFlow
    worm_integration_test.go:63: market="Above 76,772" state=open price=0.987365 rules=3 bid_levels=0 ask_levels=0
--- PASS: TestIntegrationPublicReadFlow (1.91s)
PASS
ok  	github.com/useryege/athena/util/worm	1.908s
```

If the integration test fails, first check network connectivity and whether `https://api.worm.wtf` is reachable. Empty bid or ask levels are valid; the test only requires the order book response to decode successfully.

## Run CreateOrderDraft Integration Test

This authenticated integration test calls `CreateOrderDraft` only. It may create a server-side order draft, but it does not submit the order and does not call `SubmitOrder`, `SubmitOrderCancel`, cancel, redeem, position, or other finalization endpoints.

It is protected by two opt-in switches and requires explicit draft parameters:

```bash
WORM_INTEGRATION=1 \
WORM_ORDER_DRAFT_INTEGRATION=1 \
WORM_API_KEY='...' \
WORM_API_SECRET='...' \
WORM_ORDER_DRAFT_MARKET_CONDITION_ID='...' \
WORM_ORDER_DRAFT_IS_YES=true \
WORM_ORDER_DRAFT_SIDE=BUY \
WORM_ORDER_DRAFT_ORDER_TYPE=MARKET \
WORM_ORDER_DRAFT_FUNDS='1.00' \
go test -v ./util/worm -run TestIntegrationCreateOrderDraft
```

Optional parameters:

```bash
WORM_ORDER_DRAFT_PRICE='0.50'
WORM_ORDER_DRAFT_AMOUNT='10'
```

Use parameters that are valid for the selected market. The test fails before calling the API if required credentials or draft parameters are missing. It does not log API secrets, signatures, or request payloads.



## NOTES

vpn is required to run the integration test.

## Refferences

- [Worm API Reference](https://docs.worm.wtf/api-reference)