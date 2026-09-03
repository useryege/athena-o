-- Immutable one-per-project profile persistence.

-- name: InsertProjectProfile :one
INSERT INTO project_profile (
  project_id,
  schema_version,
  completeness_status,
  failed_data_types,
  profile,
  content_hash,
  logo_url,
  current_price_usd,
  market_cap_usd,
  fdv_usd,
  tvl_usd,
  holders,
  contract_source_status,
  weth_pair_is_created,
  weth_pair_token_balance_exceeds_total_supply,
  weth_pair_lp_minimum_supply_only,
  weth_pair_fixed_fee_address_lp_share_gte_90_percent,
  weth_pair_quote_usdt_value_int,
  weth_pair_reserve_updated_at,
  usdt_pair_is_created,
  usdt_pair_token_balance_exceeds_total_supply,
  usdt_pair_lp_minimum_supply_only,
  usdt_pair_fixed_fee_address_lp_share_gte_90_percent,
  usdt_pair_quote_usdt_value_int,
  usdt_pair_reserve_updated_at,
  built_at
) VALUES (
  @project_id,
  @schema_version,
  @completeness_status,
  @failed_data_types,
  @profile::jsonb,
  @content_hash,
  @logo_url,
  sqlc.narg('current_price_usd')::numeric,
  sqlc.narg('market_cap_usd')::numeric,
  sqlc.narg('fdv_usd')::numeric,
  sqlc.narg('tvl_usd')::numeric,
  sqlc.narg('holders')::bigint,
  sqlc.narg('contract_source_status')::text,
  sqlc.narg('weth_pair_is_created')::boolean,
  sqlc.narg('weth_pair_token_balance_exceeds_total_supply')::boolean,
  sqlc.narg('weth_pair_lp_minimum_supply_only')::boolean,
  sqlc.narg('weth_pair_fixed_fee_address_lp_share_gte_90_percent')::boolean,
  sqlc.narg('weth_pair_quote_usdt_value_int')::numeric,
  sqlc.narg('weth_pair_reserve_updated_at')::bigint,
  sqlc.narg('usdt_pair_is_created')::boolean,
  sqlc.narg('usdt_pair_token_balance_exceeds_total_supply')::boolean,
  sqlc.narg('usdt_pair_lp_minimum_supply_only')::boolean,
  sqlc.narg('usdt_pair_fixed_fee_address_lp_share_gte_90_percent')::boolean,
  sqlc.narg('usdt_pair_quote_usdt_value_int')::numeric,
  sqlc.narg('usdt_pair_reserve_updated_at')::bigint,
  @built_at
)
ON CONFLICT (project_id) DO NOTHING
RETURNING *;

-- name: GetProjectProfile :one
SELECT *
FROM project_profile
WHERE project_id = @project_id;
