# PredMart API Client

`util/pred` is the ATHENA Go client for the core PredMart Swagger API.

## Source and scope

- API base URL: `https://api.predmart.com`
- Swagger UI: `https://api.predmart.com/docs`
- Local source of truth: `openapi.json`
- Implemented operations: 67

The client covers Authentication, Oracle, Lending, Leverage, Take-Profit /
Stop-Loss, Users, Portfolio, Markets, Orders, and Leagues. It intentionally
does not expose Admin, Diagnostic, SEO, Share, Discord, root, health, or debug
operations.

## Construct a client

Public and authentication endpoints do not require a token:

```go
client, err := pred.NewClient(pred.Config{})
if err != nil {
	return err
}

nonce, err := client.GetNonce(ctx, walletAddress)
if err != nil {
	return err
}
```

After signing the nonce message, verify the signature and create a new client
with the returned JWT:

```go
auth, err := client.VerifySignature(ctx, pred.SignatureVerifyRequest{
	Address:   walletAddress,
	Signature: signature,
	Nonce:     nonce.Nonce,
	Timestamp: nonce.Timestamp,
})
if err != nil {
	return err
}

authenticatedClient, err := pred.NewClient(pred.Config{
	BearerToken: auth.AccessToken,
})
if err != nil {
	return err
}
```

`BearerToken` contains only the raw JWT. The client adds the `Bearer` prefix.
Calling a protected operation without it returns
`pred.ErrBearerTokenRequired` before an HTTP request is sent.

## Response modeling

Responses with a schema in `openapi.json` use exported Go structs. Nullable
fields use pointers, and documented string/number representations are
preserved.

PredMart leaves 46 core success responses with an empty schema. The client
returns 45 of these as `json.RawMessage`, so callers retain the complete JSON
without this package inventing an undocumented model. The liquidation
justification download returns `[]byte`.

Non-2xx responses are returned as `*pred.APIError`, including the HTTP status,
decoded message when available, and a size-limited raw body.

## Endpoint coverage

| Group | Client method | Swagger operation |
| --- | --- | --- |
| Authentication | `GetNonce` | `GET /auth/nonce` |
| Authentication | `VerifySignature` | `POST /auth/verify` |
| Oracle | `GetBorrowSigningData` | `GET /oracle/borrow-intent/{borrower}/{recipient}/{token_id}/{amount}` |
| Oracle | `SubmitBorrowRelay` | `POST /oracle/borrow-relay` |
| Oracle | `GetWithdrawSigningData` | `GET /oracle/withdraw-intent/{borrower}/{to}/{token_id}/{amount}` |
| Oracle | `SubmitWithdrawRelay` | `POST /oracle/withdraw-relay` |
| Lending | `GetPoolStats` | `GET /lending/pool-stats` |
| Lending | `GetInterestCorrection` | `GET /lending/interest-correction/{address}` |
| Lending | `GetLendingPosition` | `GET /lending/position/{address}/{token_id}` |
| Lending | `ListLendingPositions` | `GET /lending/positions/{address}` |
| Lending | `ListLendingEvents` | `GET /lending/events/{address}` |
| Lending | `ListLiquidations` | `GET /lending/liquidations/{address}` |
| Lending | `GetRateHistory` | `GET /lending/rate-history` |
| Lending | `GetRealAPY` | `GET /lending/real-apy` |
| Lending | `GetDepthStatus` | `GET /lending/depth-status` |
| Lending | `GetOraclePrice` | `GET /lending/oracle-price/{token_id}` |
| Lending | `GetOraclePrices` | `GET /lending/oracle-prices` |
| Lending | `GetOrderbookAsks` | `GET /lending/orderbook-asks/{token_id}` |
| Lending | `GetContractInfo` | `GET /lending/constants` |
| Leverage | `GetLeverageAuthData` | `GET /leverage/auth-data` |
| Leverage | `GetLeverageStatus` | `GET /leverage/status/{leverage_id}` |
| Leverage | `OpenLeverage` | `POST /leverage/open` |
| Leverage | `GetFlashCloseAuthData` | `GET /leverage/flash-close-auth-data` |
| Leverage | `OpenFlashClose` | `POST /leverage/flash-close` |
| Leverage | `GetFlashCloseStatus` | `GET /leverage/flash-close-status/{close_id}` |
| TP/SL | `UpsertTPSL` | `POST /tp-sl` |
| TP/SL | `GetTPSL` | `GET /tp-sl` |
| TP/SL | `CancelTPSL` | `DELETE /tp-sl` |
| TP/SL | `GetTPSLAuthData` | `GET /tp-sl/auth-data` |
| TP/SL | `TriggerTPSLTest` | `POST /tp-sl/test-trigger` |
| Users | `GetUser` | `GET /users/{identifier}` |
| Users | `EnsureUser` | `POST /users/ensure` |
| Users | `UpdateUserProfile` | `PUT /users/{target_wallet}/profile` |
| Users | `ListUserActivity` | `GET /users/{identifier}/activity` |
| Users | `DownloadLiquidationJustification` | `GET /users/liquidation/{liquidation_id}/justification.xlsx` |
| Portfolio | `GetPortfolioSummary` | `GET /portfolio/summary` |
| Markets | `ListMarkets` | `GET /markets/` |
| Markets | `ListGroups` | `GET /groups/` |
| Markets | `ListCategories` | `GET /categories/` |
| Markets | `ListTrendingTags` | `GET /trending-tags` |
| Markets | `SearchMarkets` | `GET /search` |
| Markets | `LookupSlug` | `GET /lookup/{slug}` |
| Markets | `GetMarket` | `GET /markets/{slug}` |
| Markets | `ListSeriesSiblings` | `GET /markets/series/{series_slug}/siblings` |
| Markets | `GetGroup` | `GET /groups/{slug}` |
| Markets | `GetPriceHistory` | `GET /price-history/{market_id}` |
| Markets | `GetAggregatedOrderbook` | `GET /orders/book-aggregated/{market_id}/{side}` |
| Markets | `GetMarketPosition` | `GET /portfolio/position/{market_id}` |
| Orders | `GetCLOBAuthData` | `GET /orders/clob-auth-data` |
| Orders | `GetSafeDeployData` | `GET /orders/safe-deploy-data` |
| Orders | `DeploySafe` | `POST /orders/deploy-safe` |
| Orders | `ConfirmSafeDeployment` | `POST /orders/confirm-deploy` |
| Orders | `GetRelayStatus` | `GET /orders/relay-status` |
| Orders | `GetTradingSetupStatus` | `GET /orders/trading-setup-status` |
| Orders | `GetApproveTokensData` | `GET /orders/approve-tokens-data` |
| Orders | `ApproveTokens` | `POST /orders/approve-tokens` |
| Orders | `GetLendingApproveData` | `GET /orders/lending-approve-data` |
| Orders | `GetLendingApproveCalldata` | `POST /orders/lending-approve-calldata` |
| Orders | `GetWithdrawRelayerData` | `GET /orders/withdraw-relayer-data` |
| Orders | `WithdrawViaRelayer` | `POST /orders/withdraw-relayer` |
| Orders | `GetRepayRelayerData` | `GET /orders/repay-relayer-data` |
| Orders | `RepayViaRelayer` | `POST /orders/repay-relayer` |
| Orders | `GetDepositCollateralRelayerData` | `GET /orders/deposit-collateral-relayer-data` |
| Orders | `DepositCollateralViaRelayer` | `POST /orders/deposit-collateral-relayer` |
| Orders | `GetCLOBInfo` | `GET /orders/clob-info/{market_id}` |
| Leagues | `GetLeagueCatalog` | `GET /leagues/catalog` |
| Leagues | `RefreshLeagueCatalog` | `POST /leagues/catalog/refresh` |
