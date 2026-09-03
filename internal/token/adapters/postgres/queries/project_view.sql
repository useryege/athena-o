-- Project-centered collection and profile read model.

-- name: CountProjectListItems :one
WITH collection_summary AS (
  SELECT
    project_id,
    COUNT(*)::bigint AS task_count,
    COUNT(*) FILTER (WHERE status IN ('succeeded', 'failed'))::bigint AS terminal_count,
    COUNT(*) FILTER (WHERE status = 'succeeded')::bigint AS succeeded_count,
    COUNT(*) FILTER (WHERE status = 'failed')::bigint AS failed_count,
    COUNT(*) FILTER (WHERE status = 'running')::bigint AS running_count
  FROM project_data_collection_task
  GROUP BY project_id
), project_state AS (
  SELECT
    project.*,
    profile.project_id AS profile_project_id,
    profile.completeness_status,
    profile.weth_pair_token_balance_exceeds_total_supply,
    profile.weth_pair_lp_minimum_supply_only,
    profile.weth_pair_fixed_fee_address_lp_share_gte_90_percent,
    profile.weth_pair_quote_usdt_value_int,
    profile.usdt_pair_token_balance_exceeds_total_supply,
    profile.usdt_pair_lp_minimum_supply_only,
    profile.usdt_pair_fixed_fee_address_lp_share_gte_90_percent,
    profile.usdt_pair_quote_usdt_value_int,
    CASE
      WHEN COALESCE(collection.failed_count, 0) > 0 THEN 'needs_attention'
      WHEN COALESCE(collection.succeeded_count, 0) = 6 THEN 'complete'
      WHEN COALESCE(collection.running_count, 0) > 0 OR COALESCE(collection.succeeded_count, 0) > 0 THEN 'collecting'
      ELSE 'queued'
    END::text AS collection_status,
    CASE
      WHEN profile.project_id IS NOT NULL THEN profile.completeness_status
      WHEN profile_build.status = 'failed' THEN 'failed'
      ELSE 'pending'
    END::text AS profile_state
  FROM project
  LEFT JOIN collection_summary AS collection ON collection.project_id = project.id
  LEFT JOIN project_profile_build_task AS profile_build ON profile_build.project_id = project.id
  LEFT JOIN project_profile AS profile ON profile.project_id = project.id
)
SELECT COUNT(*)::bigint
FROM project_state
WHERE (sqlc.arg('chain_id')::bigint = 0 OR chain_id = sqlc.arg('chain_id')::bigint)
  AND (sqlc.arg('project_id')::bigint = 0 OR id = sqlc.arg('project_id')::bigint)
  AND (sqlc.narg('code_hash')::bytea IS NULL OR code_hash = sqlc.narg('code_hash')::bytea)
  AND (sqlc.narg('contract')::bytea IS NULL OR contract = sqlc.narg('contract')::bytea)
  AND (sqlc.arg('collection_status')::text = '' OR collection_status = sqlc.arg('collection_status')::text)
  AND (sqlc.arg('profile_state')::text = '' OR profile_state = sqlc.arg('profile_state')::text)
  AND (
    COALESCE(cardinality(sqlc.arg('weth_pair_token_balance_states')::text[]), 0) = 0
    OR ('detected' = ANY(sqlc.arg('weth_pair_token_balance_states')::text[]) AND weth_pair_token_balance_exceeds_total_supply IS TRUE)
    OR ('clear' = ANY(sqlc.arg('weth_pair_token_balance_states')::text[]) AND weth_pair_token_balance_exceeds_total_supply IS FALSE)
    OR ('no_profile' = ANY(sqlc.arg('weth_pair_token_balance_states')::text[]) AND profile_project_id IS NULL)
    OR ('signal_unavailable' = ANY(sqlc.arg('weth_pair_token_balance_states')::text[]) AND profile_project_id IS NOT NULL AND weth_pair_token_balance_exceeds_total_supply IS NULL)
  )
  AND (
    COALESCE(cardinality(sqlc.arg('weth_pair_lp_minimum_supply_states')::text[]), 0) = 0
    OR ('detected' = ANY(sqlc.arg('weth_pair_lp_minimum_supply_states')::text[]) AND weth_pair_lp_minimum_supply_only IS TRUE)
    OR ('clear' = ANY(sqlc.arg('weth_pair_lp_minimum_supply_states')::text[]) AND weth_pair_lp_minimum_supply_only IS FALSE)
    OR ('no_profile' = ANY(sqlc.arg('weth_pair_lp_minimum_supply_states')::text[]) AND profile_project_id IS NULL)
    OR ('signal_unavailable' = ANY(sqlc.arg('weth_pair_lp_minimum_supply_states')::text[]) AND profile_project_id IS NOT NULL AND weth_pair_lp_minimum_supply_only IS NULL)
  )
  AND (
    COALESCE(cardinality(sqlc.arg('weth_pair_fixed_fee_lp_share_states')::text[]), 0) = 0
    OR ('detected' = ANY(sqlc.arg('weth_pair_fixed_fee_lp_share_states')::text[]) AND weth_pair_fixed_fee_address_lp_share_gte_90_percent IS TRUE)
    OR ('clear' = ANY(sqlc.arg('weth_pair_fixed_fee_lp_share_states')::text[]) AND weth_pair_fixed_fee_address_lp_share_gte_90_percent IS FALSE)
    OR ('no_profile' = ANY(sqlc.arg('weth_pair_fixed_fee_lp_share_states')::text[]) AND profile_project_id IS NULL)
    OR ('signal_unavailable' = ANY(sqlc.arg('weth_pair_fixed_fee_lp_share_states')::text[]) AND profile_project_id IS NOT NULL AND weth_pair_fixed_fee_address_lp_share_gte_90_percent IS NULL)
  )
  AND (
    (
      sqlc.narg('weth_pair_quote_usdt_min')::numeric IS NULL
      AND sqlc.narg('weth_pair_quote_usdt_max')::numeric IS NULL
      AND COALESCE(cardinality(sqlc.arg('weth_pair_quote_missing_states')::text[]), 0) = 0
    )
    OR (
      (
        (sqlc.narg('weth_pair_quote_usdt_min')::numeric IS NOT NULL OR sqlc.narg('weth_pair_quote_usdt_max')::numeric IS NOT NULL)
        AND weth_pair_quote_usdt_value_int IS NOT NULL
        AND (sqlc.narg('weth_pair_quote_usdt_min')::numeric IS NULL OR weth_pair_quote_usdt_value_int >= sqlc.narg('weth_pair_quote_usdt_min')::numeric)
        AND (sqlc.narg('weth_pair_quote_usdt_max')::numeric IS NULL OR weth_pair_quote_usdt_value_int <= sqlc.narg('weth_pair_quote_usdt_max')::numeric)
      )
      OR ('no_profile' = ANY(sqlc.arg('weth_pair_quote_missing_states')::text[]) AND profile_project_id IS NULL)
      OR ('value_unavailable' = ANY(sqlc.arg('weth_pair_quote_missing_states')::text[]) AND profile_project_id IS NOT NULL AND weth_pair_quote_usdt_value_int IS NULL)
    )
  )
  AND (
    COALESCE(cardinality(sqlc.arg('usdt_pair_token_balance_states')::text[]), 0) = 0
    OR ('detected' = ANY(sqlc.arg('usdt_pair_token_balance_states')::text[]) AND usdt_pair_token_balance_exceeds_total_supply IS TRUE)
    OR ('clear' = ANY(sqlc.arg('usdt_pair_token_balance_states')::text[]) AND usdt_pair_token_balance_exceeds_total_supply IS FALSE)
    OR ('no_profile' = ANY(sqlc.arg('usdt_pair_token_balance_states')::text[]) AND profile_project_id IS NULL)
    OR ('signal_unavailable' = ANY(sqlc.arg('usdt_pair_token_balance_states')::text[]) AND profile_project_id IS NOT NULL AND usdt_pair_token_balance_exceeds_total_supply IS NULL)
  )
  AND (
    COALESCE(cardinality(sqlc.arg('usdt_pair_lp_minimum_supply_states')::text[]), 0) = 0
    OR ('detected' = ANY(sqlc.arg('usdt_pair_lp_minimum_supply_states')::text[]) AND usdt_pair_lp_minimum_supply_only IS TRUE)
    OR ('clear' = ANY(sqlc.arg('usdt_pair_lp_minimum_supply_states')::text[]) AND usdt_pair_lp_minimum_supply_only IS FALSE)
    OR ('no_profile' = ANY(sqlc.arg('usdt_pair_lp_minimum_supply_states')::text[]) AND profile_project_id IS NULL)
    OR ('signal_unavailable' = ANY(sqlc.arg('usdt_pair_lp_minimum_supply_states')::text[]) AND profile_project_id IS NOT NULL AND usdt_pair_lp_minimum_supply_only IS NULL)
  )
  AND (
    COALESCE(cardinality(sqlc.arg('usdt_pair_fixed_fee_lp_share_states')::text[]), 0) = 0
    OR ('detected' = ANY(sqlc.arg('usdt_pair_fixed_fee_lp_share_states')::text[]) AND usdt_pair_fixed_fee_address_lp_share_gte_90_percent IS TRUE)
    OR ('clear' = ANY(sqlc.arg('usdt_pair_fixed_fee_lp_share_states')::text[]) AND usdt_pair_fixed_fee_address_lp_share_gte_90_percent IS FALSE)
    OR ('no_profile' = ANY(sqlc.arg('usdt_pair_fixed_fee_lp_share_states')::text[]) AND profile_project_id IS NULL)
    OR ('signal_unavailable' = ANY(sqlc.arg('usdt_pair_fixed_fee_lp_share_states')::text[]) AND profile_project_id IS NOT NULL AND usdt_pair_fixed_fee_address_lp_share_gte_90_percent IS NULL)
  )
  AND (
    (
      sqlc.narg('usdt_pair_quote_usdt_min')::numeric IS NULL
      AND sqlc.narg('usdt_pair_quote_usdt_max')::numeric IS NULL
      AND COALESCE(cardinality(sqlc.arg('usdt_pair_quote_missing_states')::text[]), 0) = 0
    )
    OR (
      (
        (sqlc.narg('usdt_pair_quote_usdt_min')::numeric IS NOT NULL OR sqlc.narg('usdt_pair_quote_usdt_max')::numeric IS NOT NULL)
        AND usdt_pair_quote_usdt_value_int IS NOT NULL
        AND (sqlc.narg('usdt_pair_quote_usdt_min')::numeric IS NULL OR usdt_pair_quote_usdt_value_int >= sqlc.narg('usdt_pair_quote_usdt_min')::numeric)
        AND (sqlc.narg('usdt_pair_quote_usdt_max')::numeric IS NULL OR usdt_pair_quote_usdt_value_int <= sqlc.narg('usdt_pair_quote_usdt_max')::numeric)
      )
      OR ('no_profile' = ANY(sqlc.arg('usdt_pair_quote_missing_states')::text[]) AND profile_project_id IS NULL)
      OR ('value_unavailable' = ANY(sqlc.arg('usdt_pair_quote_missing_states')::text[]) AND profile_project_id IS NOT NULL AND usdt_pair_quote_usdt_value_int IS NULL)
    )
  );

-- name: ListProjectListItems :many
WITH collection_summary AS (
  SELECT
    project_id,
    COUNT(*)::bigint AS task_count,
    COUNT(*) FILTER (WHERE status IN ('succeeded', 'failed'))::bigint AS terminal_count,
    COUNT(*) FILTER (WHERE status = 'succeeded')::bigint AS succeeded_count,
    COUNT(*) FILTER (WHERE status = 'failed')::bigint AS failed_count,
    COUNT(*) FILTER (WHERE status = 'running')::bigint AS running_count
  FROM project_data_collection_task
  GROUP BY project_id
), project_state AS (
  SELECT
    project.id,
    project.chain_id,
    project.contract,
	project.code_hash,
	project.tx_hash,
    project.block_number,
    project.block_time,
    project.name,
    project.symbol,
	project.weth_pair,
	project.usdt_pair,
    project.created_at,
    COALESCE(collection.task_count, 0)::bigint AS collection_task_count,
    COALESCE(collection.terminal_count, 0)::bigint AS collection_terminal_count,
    COALESCE(collection.succeeded_count, 0)::bigint AS collection_succeeded_count,
    COALESCE(collection.failed_count, 0)::bigint AS collection_failed_count,
    CASE
      WHEN COALESCE(collection.failed_count, 0) > 0 THEN 'needs_attention'
      WHEN COALESCE(collection.succeeded_count, 0) = 6 THEN 'complete'
      WHEN COALESCE(collection.running_count, 0) > 0 OR COALESCE(collection.succeeded_count, 0) > 0 THEN 'collecting'
      ELSE 'queued'
    END::text AS collection_status,
    CASE
      WHEN profile.project_id IS NOT NULL THEN profile.completeness_status
      WHEN profile_build.status = 'failed' THEN 'failed'
      ELSE 'pending'
    END::text AS profile_state,
    COALESCE(profile_build.status, '')::text AS profile_build_status,
    COALESCE(profile_build.failure_count, 0)::int AS profile_build_failure_count,
    COALESCE(profile_build.last_error, '')::text AS profile_build_last_error,
    profile_build.updated_at AS profile_build_updated_at,
    profile.project_id AS profile_project_id,
    COALESCE(profile.logo_url, '')::text AS logo_url,
    profile.completeness_status,
    profile.current_price_usd,
    profile.market_cap_usd,
    profile.fdv_usd,
    profile.tvl_usd,
    profile.holders,
    profile.contract_source_status,
    profile.weth_pair_is_created,
    profile.weth_pair_token_balance_exceeds_total_supply,
    profile.weth_pair_lp_minimum_supply_only,
    profile.weth_pair_fixed_fee_address_lp_share_gte_90_percent,
    profile.weth_pair_quote_usdt_value_int,
    profile.weth_pair_reserve_updated_at,
    profile.usdt_pair_is_created,
    profile.usdt_pair_token_balance_exceeds_total_supply,
    profile.usdt_pair_lp_minimum_supply_only,
    profile.usdt_pair_fixed_fee_address_lp_share_gte_90_percent,
    profile.usdt_pair_quote_usdt_value_int,
    profile.usdt_pair_reserve_updated_at,
    profile.built_at AS profile_built_at
  FROM project
  LEFT JOIN collection_summary AS collection ON collection.project_id = project.id
  LEFT JOIN project_profile_build_task AS profile_build ON profile_build.project_id = project.id
  LEFT JOIN project_profile AS profile ON profile.project_id = project.id
)
SELECT *
FROM project_state
WHERE (sqlc.arg('chain_id')::bigint = 0 OR chain_id = sqlc.arg('chain_id')::bigint)
  AND (sqlc.arg('project_id')::bigint = 0 OR id = sqlc.arg('project_id')::bigint)
  AND (sqlc.narg('code_hash')::bytea IS NULL OR EXISTS (
    SELECT 1 FROM project WHERE project.id = project_state.id AND project.code_hash = sqlc.narg('code_hash')::bytea
  ))
  AND (sqlc.narg('contract')::bytea IS NULL OR contract = sqlc.narg('contract')::bytea)
  AND (sqlc.arg('collection_status')::text = '' OR collection_status = sqlc.arg('collection_status')::text)
  AND (sqlc.arg('profile_state')::text = '' OR profile_state = sqlc.arg('profile_state')::text)
  AND (
    COALESCE(cardinality(sqlc.arg('weth_pair_token_balance_states')::text[]), 0) = 0
    OR ('detected' = ANY(sqlc.arg('weth_pair_token_balance_states')::text[]) AND weth_pair_token_balance_exceeds_total_supply IS TRUE)
    OR ('clear' = ANY(sqlc.arg('weth_pair_token_balance_states')::text[]) AND weth_pair_token_balance_exceeds_total_supply IS FALSE)
    OR ('no_profile' = ANY(sqlc.arg('weth_pair_token_balance_states')::text[]) AND profile_project_id IS NULL)
    OR ('signal_unavailable' = ANY(sqlc.arg('weth_pair_token_balance_states')::text[]) AND profile_project_id IS NOT NULL AND weth_pair_token_balance_exceeds_total_supply IS NULL)
  )
  AND (
    COALESCE(cardinality(sqlc.arg('weth_pair_lp_minimum_supply_states')::text[]), 0) = 0
    OR ('detected' = ANY(sqlc.arg('weth_pair_lp_minimum_supply_states')::text[]) AND weth_pair_lp_minimum_supply_only IS TRUE)
    OR ('clear' = ANY(sqlc.arg('weth_pair_lp_minimum_supply_states')::text[]) AND weth_pair_lp_minimum_supply_only IS FALSE)
    OR ('no_profile' = ANY(sqlc.arg('weth_pair_lp_minimum_supply_states')::text[]) AND profile_project_id IS NULL)
    OR ('signal_unavailable' = ANY(sqlc.arg('weth_pair_lp_minimum_supply_states')::text[]) AND profile_project_id IS NOT NULL AND weth_pair_lp_minimum_supply_only IS NULL)
  )
  AND (
    COALESCE(cardinality(sqlc.arg('weth_pair_fixed_fee_lp_share_states')::text[]), 0) = 0
    OR ('detected' = ANY(sqlc.arg('weth_pair_fixed_fee_lp_share_states')::text[]) AND weth_pair_fixed_fee_address_lp_share_gte_90_percent IS TRUE)
    OR ('clear' = ANY(sqlc.arg('weth_pair_fixed_fee_lp_share_states')::text[]) AND weth_pair_fixed_fee_address_lp_share_gte_90_percent IS FALSE)
    OR ('no_profile' = ANY(sqlc.arg('weth_pair_fixed_fee_lp_share_states')::text[]) AND profile_project_id IS NULL)
    OR ('signal_unavailable' = ANY(sqlc.arg('weth_pair_fixed_fee_lp_share_states')::text[]) AND profile_project_id IS NOT NULL AND weth_pair_fixed_fee_address_lp_share_gte_90_percent IS NULL)
  )
  AND (
    (
      sqlc.narg('weth_pair_quote_usdt_min')::numeric IS NULL
      AND sqlc.narg('weth_pair_quote_usdt_max')::numeric IS NULL
      AND COALESCE(cardinality(sqlc.arg('weth_pair_quote_missing_states')::text[]), 0) = 0
    )
    OR (
      (
        (sqlc.narg('weth_pair_quote_usdt_min')::numeric IS NOT NULL OR sqlc.narg('weth_pair_quote_usdt_max')::numeric IS NOT NULL)
        AND weth_pair_quote_usdt_value_int IS NOT NULL
        AND (sqlc.narg('weth_pair_quote_usdt_min')::numeric IS NULL OR weth_pair_quote_usdt_value_int >= sqlc.narg('weth_pair_quote_usdt_min')::numeric)
        AND (sqlc.narg('weth_pair_quote_usdt_max')::numeric IS NULL OR weth_pair_quote_usdt_value_int <= sqlc.narg('weth_pair_quote_usdt_max')::numeric)
      )
      OR ('no_profile' = ANY(sqlc.arg('weth_pair_quote_missing_states')::text[]) AND profile_project_id IS NULL)
      OR ('value_unavailable' = ANY(sqlc.arg('weth_pair_quote_missing_states')::text[]) AND profile_project_id IS NOT NULL AND weth_pair_quote_usdt_value_int IS NULL)
    )
  )
  AND (
    COALESCE(cardinality(sqlc.arg('usdt_pair_token_balance_states')::text[]), 0) = 0
    OR ('detected' = ANY(sqlc.arg('usdt_pair_token_balance_states')::text[]) AND usdt_pair_token_balance_exceeds_total_supply IS TRUE)
    OR ('clear' = ANY(sqlc.arg('usdt_pair_token_balance_states')::text[]) AND usdt_pair_token_balance_exceeds_total_supply IS FALSE)
    OR ('no_profile' = ANY(sqlc.arg('usdt_pair_token_balance_states')::text[]) AND profile_project_id IS NULL)
    OR ('signal_unavailable' = ANY(sqlc.arg('usdt_pair_token_balance_states')::text[]) AND profile_project_id IS NOT NULL AND usdt_pair_token_balance_exceeds_total_supply IS NULL)
  )
  AND (
    COALESCE(cardinality(sqlc.arg('usdt_pair_lp_minimum_supply_states')::text[]), 0) = 0
    OR ('detected' = ANY(sqlc.arg('usdt_pair_lp_minimum_supply_states')::text[]) AND usdt_pair_lp_minimum_supply_only IS TRUE)
    OR ('clear' = ANY(sqlc.arg('usdt_pair_lp_minimum_supply_states')::text[]) AND usdt_pair_lp_minimum_supply_only IS FALSE)
    OR ('no_profile' = ANY(sqlc.arg('usdt_pair_lp_minimum_supply_states')::text[]) AND profile_project_id IS NULL)
    OR ('signal_unavailable' = ANY(sqlc.arg('usdt_pair_lp_minimum_supply_states')::text[]) AND profile_project_id IS NOT NULL AND usdt_pair_lp_minimum_supply_only IS NULL)
  )
  AND (
    COALESCE(cardinality(sqlc.arg('usdt_pair_fixed_fee_lp_share_states')::text[]), 0) = 0
    OR ('detected' = ANY(sqlc.arg('usdt_pair_fixed_fee_lp_share_states')::text[]) AND usdt_pair_fixed_fee_address_lp_share_gte_90_percent IS TRUE)
    OR ('clear' = ANY(sqlc.arg('usdt_pair_fixed_fee_lp_share_states')::text[]) AND usdt_pair_fixed_fee_address_lp_share_gte_90_percent IS FALSE)
    OR ('no_profile' = ANY(sqlc.arg('usdt_pair_fixed_fee_lp_share_states')::text[]) AND profile_project_id IS NULL)
    OR ('signal_unavailable' = ANY(sqlc.arg('usdt_pair_fixed_fee_lp_share_states')::text[]) AND profile_project_id IS NOT NULL AND usdt_pair_fixed_fee_address_lp_share_gte_90_percent IS NULL)
  )
  AND (
    (
      sqlc.narg('usdt_pair_quote_usdt_min')::numeric IS NULL
      AND sqlc.narg('usdt_pair_quote_usdt_max')::numeric IS NULL
      AND COALESCE(cardinality(sqlc.arg('usdt_pair_quote_missing_states')::text[]), 0) = 0
    )
    OR (
      (
        (sqlc.narg('usdt_pair_quote_usdt_min')::numeric IS NOT NULL OR sqlc.narg('usdt_pair_quote_usdt_max')::numeric IS NOT NULL)
        AND usdt_pair_quote_usdt_value_int IS NOT NULL
        AND (sqlc.narg('usdt_pair_quote_usdt_min')::numeric IS NULL OR usdt_pair_quote_usdt_value_int >= sqlc.narg('usdt_pair_quote_usdt_min')::numeric)
        AND (sqlc.narg('usdt_pair_quote_usdt_max')::numeric IS NULL OR usdt_pair_quote_usdt_value_int <= sqlc.narg('usdt_pair_quote_usdt_max')::numeric)
      )
      OR ('no_profile' = ANY(sqlc.arg('usdt_pair_quote_missing_states')::text[]) AND profile_project_id IS NULL)
      OR ('value_unavailable' = ANY(sqlc.arg('usdt_pair_quote_missing_states')::text[]) AND profile_project_id IS NOT NULL AND usdt_pair_quote_usdt_value_int IS NULL)
    )
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
