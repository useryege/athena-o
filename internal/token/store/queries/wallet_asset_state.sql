-- name: UpsertWalletAssetState :one
INSERT INTO wallet_asset_state (
  chain_id,
  wallet,
  weth_balance,
  usdt_balance,
  native_balance,
  usdt_value,
  fetched_at
) VALUES (
  @chain_id,
  @wallet,
  @weth_balance,
  @usdt_balance,
  @native_balance,
  @usdt_value,
  @fetched_at
)
ON CONFLICT (chain_id, wallet) DO UPDATE
SET weth_balance = EXCLUDED.weth_balance,
  usdt_balance = EXCLUDED.usdt_balance,
  native_balance = EXCLUDED.native_balance,
  usdt_value = EXCLUDED.usdt_value,
  fetched_at = EXCLUDED.fetched_at
RETURNING *;

-- name: GetWalletAssetState :one
SELECT *
FROM wallet_asset_state
WHERE chain_id = @chain_id
  AND wallet = @wallet;

-- name: ListWalletAssetStatesByWallets :many
SELECT *
FROM wallet_asset_state
WHERE chain_id = @chain_id
  AND wallet = ANY(@wallets::bytea[])
ORDER BY usdt_value DESC, wallet;

-- name: DeleteWalletAssetState :execrows
DELETE FROM wallet_asset_state
WHERE chain_id = @chain_id
  AND wallet = @wallet;
