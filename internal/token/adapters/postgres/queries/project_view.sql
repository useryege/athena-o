-- Project-centered current read model.
-- name: CountProjectListItems :one
SELECT COUNT(*)::bigint
FROM project
LEFT JOIN project_research_state AS research
  ON research.project_id = project.id
LEFT JOIN project_report_revision AS report
  ON report.project_id = project.id
  AND report.revision = research.current_report_revision
LEFT JOIN project_selection_evaluation_task AS evaluation
  ON evaluation.project_id = report.project_id
  AND evaluation.report_revision = report.revision
LEFT JOIN project_selection AS selection
  ON selection.id = research.current_selection_id
  AND selection.project_id = project.id
  AND evaluation.status = 'succeeded'
  AND research.last_evaluated_report_revision = report.revision
WHERE (sqlc.arg('chain_id')::bigint = 0 OR project.chain_id = sqlc.arg('chain_id')::bigint)
  AND (sqlc.arg('project_id')::bigint = 0 OR project.id = sqlc.arg('project_id')::bigint)
  AND (sqlc.narg('code_hash')::bytea IS NULL OR project.code_hash = sqlc.narg('code_hash')::bytea)
  AND (sqlc.narg('contract')::bytea IS NULL OR project.contract = sqlc.narg('contract')::bytea)
  AND (sqlc.arg('research_status')::text = '' OR research.status = sqlc.arg('research_status')::text)
  AND (
    sqlc.arg('report_state')::text = ''
    OR (sqlc.arg('report_state')::text = 'none' AND report.id IS NULL)
    OR report.completeness_status = sqlc.arg('report_state')::text
  )
  AND (
    sqlc.arg('evaluation_status')::text = ''
    OR (sqlc.arg('evaluation_status')::text = 'none' AND evaluation.id IS NULL)
    OR evaluation.status = sqlc.arg('evaluation_status')::text
  )
  AND (
    sqlc.arg('selection_outcome')::text = ''
    OR (sqlc.arg('selection_outcome')::text = 'none' AND selection.id IS NULL)
    OR selection.outcome = sqlc.arg('selection_outcome')::text
  )
  AND (
    COALESCE(cardinality(sqlc.arg('report_pair_remove_liquidity_states')::text[]), 0) = 0
    OR (
      (
        'detected' = ANY(sqlc.arg('report_pair_remove_liquidity_states')::text[])
        AND (
          (sqlc.arg('report_pair_kind')::text = 'weth' AND report.weth_pair_is_remove_liquidity IS TRUE)
          OR (sqlc.arg('report_pair_kind')::text = 'usdt' AND report.usdt_pair_is_remove_liquidity IS TRUE)
        )
      )
      OR (
        'clear' = ANY(sqlc.arg('report_pair_remove_liquidity_states')::text[])
        AND (
          (sqlc.arg('report_pair_kind')::text = 'weth' AND report.weth_pair_is_remove_liquidity IS FALSE)
          OR (sqlc.arg('report_pair_kind')::text = 'usdt' AND report.usdt_pair_is_remove_liquidity IS FALSE)
        )
      )
      OR ('no_report' = ANY(sqlc.arg('report_pair_remove_liquidity_states')::text[]) AND report.id IS NULL)
      OR (
        'risk_unavailable' = ANY(sqlc.arg('report_pair_remove_liquidity_states')::text[])
        AND report.id IS NOT NULL
        AND (
          (sqlc.arg('report_pair_kind')::text = 'weth' AND report.weth_pair_is_remove_liquidity IS NULL)
          OR (sqlc.arg('report_pair_kind')::text = 'usdt' AND report.usdt_pair_is_remove_liquidity IS NULL)
        )
      )
    )
  )
  AND (
    COALESCE(cardinality(sqlc.arg('report_pair_mint_states')::text[]), 0) = 0
    OR (
      (
        'detected' = ANY(sqlc.arg('report_pair_mint_states')::text[])
        AND (
          (sqlc.arg('report_pair_kind')::text = 'weth' AND report.weth_pair_is_mint IS TRUE)
          OR (sqlc.arg('report_pair_kind')::text = 'usdt' AND report.usdt_pair_is_mint IS TRUE)
        )
      )
      OR (
        'clear' = ANY(sqlc.arg('report_pair_mint_states')::text[])
        AND (
          (sqlc.arg('report_pair_kind')::text = 'weth' AND report.weth_pair_is_mint IS FALSE)
          OR (sqlc.arg('report_pair_kind')::text = 'usdt' AND report.usdt_pair_is_mint IS FALSE)
        )
      )
      OR ('no_report' = ANY(sqlc.arg('report_pair_mint_states')::text[]) AND report.id IS NULL)
      OR (
        'risk_unavailable' = ANY(sqlc.arg('report_pair_mint_states')::text[])
        AND report.id IS NOT NULL
        AND (
          (sqlc.arg('report_pair_kind')::text = 'weth' AND report.weth_pair_is_mint IS NULL)
          OR (sqlc.arg('report_pair_kind')::text = 'usdt' AND report.usdt_pair_is_mint IS NULL)
        )
      )
    )
  )
  AND (
    (
      sqlc.narg('report_pair_quote_usdt_min')::numeric IS NULL
      AND sqlc.narg('report_pair_quote_usdt_max')::numeric IS NULL
      AND COALESCE(cardinality(sqlc.arg('report_pair_quote_missing_states')::text[]), 0) = 0
    )
    OR (
      (
        (sqlc.narg('report_pair_quote_usdt_min')::numeric IS NOT NULL OR sqlc.narg('report_pair_quote_usdt_max')::numeric IS NOT NULL)
        AND sqlc.arg('report_pair_kind')::text = 'weth'
        AND report.weth_pair_quote_usdt_value_int IS NOT NULL
        AND (
          sqlc.narg('report_pair_quote_usdt_min')::numeric IS NULL
          OR report.weth_pair_quote_usdt_value_int >= sqlc.narg('report_pair_quote_usdt_min')::numeric
        )
        AND (
          sqlc.narg('report_pair_quote_usdt_max')::numeric IS NULL
          OR report.weth_pair_quote_usdt_value_int <= sqlc.narg('report_pair_quote_usdt_max')::numeric
        )
      )
      OR (
        (sqlc.narg('report_pair_quote_usdt_min')::numeric IS NOT NULL OR sqlc.narg('report_pair_quote_usdt_max')::numeric IS NOT NULL)
        AND sqlc.arg('report_pair_kind')::text = 'usdt'
        AND report.usdt_pair_quote_usdt_value_int IS NOT NULL
        AND (
          sqlc.narg('report_pair_quote_usdt_min')::numeric IS NULL
          OR report.usdt_pair_quote_usdt_value_int >= sqlc.narg('report_pair_quote_usdt_min')::numeric
        )
        AND (
          sqlc.narg('report_pair_quote_usdt_max')::numeric IS NULL
          OR report.usdt_pair_quote_usdt_value_int <= sqlc.narg('report_pair_quote_usdt_max')::numeric
        )
      )
      OR ('no_report' = ANY(sqlc.arg('report_pair_quote_missing_states')::text[]) AND report.id IS NULL)
      OR (
        'risk_unavailable' = ANY(sqlc.arg('report_pair_quote_missing_states')::text[])
        AND report.id IS NOT NULL
        AND (
          (sqlc.arg('report_pair_kind')::text = 'weth' AND report.weth_pair_quote_usdt_value_int IS NULL)
          OR (sqlc.arg('report_pair_kind')::text = 'usdt' AND report.usdt_pair_quote_usdt_value_int IS NULL)
        )
      )
    )
  );

-- name: ListProjectListItems :many
SELECT
  project.id,
  project.chain_id,
  project.block_time,
  project.name,
  project.symbol,
  project.created_at,
  COALESCE(
    CASE
      WHEN ave_observation.schema_version = 1
      THEN ave_observation.payload #>> '{token,logoUrl}'
    END,
    ''
  )::text AS logo_url,
  COALESCE(research.status, '')::text AS research_status,
  report.revision AS report_revision,
  COALESCE(report.completeness_status, '')::text AS report_completeness_status,
  report.built_at AS report_built_at,
  report.weth_pair_is_created,
  report.weth_pair_is_remove_liquidity,
  report.weth_pair_is_mint,
  report.weth_pair_quote_usdt_value_int,
  report.weth_pair_last_swap_timestamp,
  report.usdt_pair_is_created,
  report.usdt_pair_is_remove_liquidity,
  report.usdt_pair_is_mint,
  report.usdt_pair_quote_usdt_value_int,
  report.usdt_pair_last_swap_timestamp,
  COALESCE(evaluation.status, '')::text AS evaluation_status,
  COALESCE(evaluation.attempts, 0)::int AS evaluation_failed_attempts,
  COALESCE(evaluation.last_error, '')::text AS evaluation_last_error,
  evaluation.updated_at AS evaluation_updated_at,
  COALESCE(selection.outcome, '')::text AS selection_outcome,
  CASE
    WHEN selection.id IS NOT NULL
    THEN research.last_evaluated_at
    ELSE NULL::timestamptz
  END AS evaluated_at
FROM project
LEFT JOIN project_research_state AS research
  ON research.project_id = project.id
LEFT JOIN project_report_revision AS report
  ON report.project_id = project.id
  AND report.revision = research.current_report_revision
LEFT JOIN project_selection_evaluation_task AS evaluation
  ON evaluation.project_id = report.project_id
  AND evaluation.report_revision = report.revision
LEFT JOIN project_selection AS selection
  ON selection.id = research.current_selection_id
  AND selection.project_id = project.id
  AND evaluation.status = 'succeeded'
  AND research.last_evaluated_report_revision = report.revision
LEFT JOIN project_observation_current AS ave_current
  ON ave_current.project_id = project.id
  AND ave_current.data_type = 'ave'
LEFT JOIN project_observation AS ave_observation
  ON ave_observation.id = ave_current.observation_id
  AND ave_observation.project_id = project.id
  AND ave_observation.data_type = 'ave'
WHERE (sqlc.arg('chain_id')::bigint = 0 OR project.chain_id = sqlc.arg('chain_id')::bigint)
  AND (sqlc.arg('project_id')::bigint = 0 OR project.id = sqlc.arg('project_id')::bigint)
  AND (sqlc.narg('code_hash')::bytea IS NULL OR project.code_hash = sqlc.narg('code_hash')::bytea)
  AND (sqlc.narg('contract')::bytea IS NULL OR project.contract = sqlc.narg('contract')::bytea)
  AND (sqlc.arg('research_status')::text = '' OR research.status = sqlc.arg('research_status')::text)
  AND (
    sqlc.arg('report_state')::text = ''
    OR (sqlc.arg('report_state')::text = 'none' AND report.id IS NULL)
    OR report.completeness_status = sqlc.arg('report_state')::text
  )
  AND (
    sqlc.arg('evaluation_status')::text = ''
    OR (sqlc.arg('evaluation_status')::text = 'none' AND evaluation.id IS NULL)
    OR evaluation.status = sqlc.arg('evaluation_status')::text
  )
  AND (
    sqlc.arg('selection_outcome')::text = ''
    OR (sqlc.arg('selection_outcome')::text = 'none' AND selection.id IS NULL)
    OR selection.outcome = sqlc.arg('selection_outcome')::text
  )
  AND (
    COALESCE(cardinality(sqlc.arg('report_pair_remove_liquidity_states')::text[]), 0) = 0
    OR (
      (
        'detected' = ANY(sqlc.arg('report_pair_remove_liquidity_states')::text[])
        AND (
          (sqlc.arg('report_pair_kind')::text = 'weth' AND report.weth_pair_is_remove_liquidity IS TRUE)
          OR (sqlc.arg('report_pair_kind')::text = 'usdt' AND report.usdt_pair_is_remove_liquidity IS TRUE)
        )
      )
      OR (
        'clear' = ANY(sqlc.arg('report_pair_remove_liquidity_states')::text[])
        AND (
          (sqlc.arg('report_pair_kind')::text = 'weth' AND report.weth_pair_is_remove_liquidity IS FALSE)
          OR (sqlc.arg('report_pair_kind')::text = 'usdt' AND report.usdt_pair_is_remove_liquidity IS FALSE)
        )
      )
      OR ('no_report' = ANY(sqlc.arg('report_pair_remove_liquidity_states')::text[]) AND report.id IS NULL)
      OR (
        'risk_unavailable' = ANY(sqlc.arg('report_pair_remove_liquidity_states')::text[])
        AND report.id IS NOT NULL
        AND (
          (sqlc.arg('report_pair_kind')::text = 'weth' AND report.weth_pair_is_remove_liquidity IS NULL)
          OR (sqlc.arg('report_pair_kind')::text = 'usdt' AND report.usdt_pair_is_remove_liquidity IS NULL)
        )
      )
    )
  )
  AND (
    COALESCE(cardinality(sqlc.arg('report_pair_mint_states')::text[]), 0) = 0
    OR (
      (
        'detected' = ANY(sqlc.arg('report_pair_mint_states')::text[])
        AND (
          (sqlc.arg('report_pair_kind')::text = 'weth' AND report.weth_pair_is_mint IS TRUE)
          OR (sqlc.arg('report_pair_kind')::text = 'usdt' AND report.usdt_pair_is_mint IS TRUE)
        )
      )
      OR (
        'clear' = ANY(sqlc.arg('report_pair_mint_states')::text[])
        AND (
          (sqlc.arg('report_pair_kind')::text = 'weth' AND report.weth_pair_is_mint IS FALSE)
          OR (sqlc.arg('report_pair_kind')::text = 'usdt' AND report.usdt_pair_is_mint IS FALSE)
        )
      )
      OR ('no_report' = ANY(sqlc.arg('report_pair_mint_states')::text[]) AND report.id IS NULL)
      OR (
        'risk_unavailable' = ANY(sqlc.arg('report_pair_mint_states')::text[])
        AND report.id IS NOT NULL
        AND (
          (sqlc.arg('report_pair_kind')::text = 'weth' AND report.weth_pair_is_mint IS NULL)
          OR (sqlc.arg('report_pair_kind')::text = 'usdt' AND report.usdt_pair_is_mint IS NULL)
        )
      )
    )
  )
  AND (
    (
      sqlc.narg('report_pair_quote_usdt_min')::numeric IS NULL
      AND sqlc.narg('report_pair_quote_usdt_max')::numeric IS NULL
      AND COALESCE(cardinality(sqlc.arg('report_pair_quote_missing_states')::text[]), 0) = 0
    )
    OR (
      (
        (sqlc.narg('report_pair_quote_usdt_min')::numeric IS NOT NULL OR sqlc.narg('report_pair_quote_usdt_max')::numeric IS NOT NULL)
        AND sqlc.arg('report_pair_kind')::text = 'weth'
        AND report.weth_pair_quote_usdt_value_int IS NOT NULL
        AND (
          sqlc.narg('report_pair_quote_usdt_min')::numeric IS NULL
          OR report.weth_pair_quote_usdt_value_int >= sqlc.narg('report_pair_quote_usdt_min')::numeric
        )
        AND (
          sqlc.narg('report_pair_quote_usdt_max')::numeric IS NULL
          OR report.weth_pair_quote_usdt_value_int <= sqlc.narg('report_pair_quote_usdt_max')::numeric
        )
      )
      OR (
        (sqlc.narg('report_pair_quote_usdt_min')::numeric IS NOT NULL OR sqlc.narg('report_pair_quote_usdt_max')::numeric IS NOT NULL)
        AND sqlc.arg('report_pair_kind')::text = 'usdt'
        AND report.usdt_pair_quote_usdt_value_int IS NOT NULL
        AND (
          sqlc.narg('report_pair_quote_usdt_min')::numeric IS NULL
          OR report.usdt_pair_quote_usdt_value_int >= sqlc.narg('report_pair_quote_usdt_min')::numeric
        )
        AND (
          sqlc.narg('report_pair_quote_usdt_max')::numeric IS NULL
          OR report.usdt_pair_quote_usdt_value_int <= sqlc.narg('report_pair_quote_usdt_max')::numeric
        )
      )
      OR ('no_report' = ANY(sqlc.arg('report_pair_quote_missing_states')::text[]) AND report.id IS NULL)
      OR (
        'risk_unavailable' = ANY(sqlc.arg('report_pair_quote_missing_states')::text[])
        AND report.id IS NOT NULL
        AND (
          (sqlc.arg('report_pair_kind')::text = 'weth' AND report.weth_pair_quote_usdt_value_int IS NULL)
          OR (sqlc.arg('report_pair_kind')::text = 'usdt' AND report.usdt_pair_quote_usdt_value_int IS NULL)
        )
      )
    )
  )
ORDER BY project.created_at DESC, project.id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
