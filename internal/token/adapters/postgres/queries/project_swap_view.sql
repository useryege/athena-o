-- Read-only per-project Swap activity and event-detail queries.

-- name: ListProjectSwapActivityPairs :many
SELECT project_swap_pair.*
FROM project_swap_pair
WHERE project_id = @project_id
ORDER BY
  CASE pair_kind
    WHEN 'weth' THEN 1
    WHEN 'usdt' THEN 2
    ELSE 3
  END,
  id;

-- name: GetProjectSwapActivityPairTotals :one
WITH mapped_events AS (
  SELECT
    event.transaction_hash,
    event.tx_from,
    CASE
      WHEN sqlc.arg('base_is_token0')::boolean THEN event.amount0_in
      ELSE event.amount1_in
    END AS base_in,
    CASE
      WHEN sqlc.arg('base_is_token0')::boolean THEN event.amount0_out
      ELSE event.amount1_out
    END AS base_out,
    CASE
      WHEN sqlc.arg('base_is_token0')::boolean THEN event.amount1_in
      ELSE event.amount0_in
    END AS quote_in,
    CASE
      WHEN sqlc.arg('base_is_token0')::boolean THEN event.amount1_out
      ELSE event.amount0_out
    END AS quote_out
  FROM project_swap_event AS event
  WHERE event.project_swap_pair_id = @project_swap_pair_id
),
classified_events AS (
  SELECT
    mapped_events.*,
    CASE
      WHEN quote_in > 0 AND base_out > 0 AND base_in = 0 AND quote_out = 0 THEN 'buy'
      WHEN base_in > 0 AND quote_out > 0 AND quote_in = 0 AND base_out = 0 THEN 'sell'
      ELSE 'complex'
    END::text AS direction
  FROM mapped_events
)
SELECT
  COUNT(*)::bigint AS event_count,
  COUNT(DISTINCT transaction_hash)::bigint AS transaction_count,
  COUNT(DISTINCT tx_from)::bigint AS transaction_origin_count,
  COUNT(*) FILTER (WHERE direction = 'buy')::bigint AS buy_event_count,
  COUNT(*) FILTER (WHERE direction = 'sell')::bigint AS sell_event_count,
  COUNT(*) FILTER (WHERE direction = 'complex')::bigint AS complex_event_count,
  COALESCE(SUM(base_in), 0)::text AS base_in,
  COALESCE(SUM(base_out), 0)::text AS base_out,
  COALESCE(SUM(quote_in), 0)::text AS quote_in,
  COALESCE(SUM(quote_out), 0)::text AS quote_out,
  COALESCE(SUM(quote_in) FILTER (WHERE direction = 'buy'), 0)::text AS buy_quote_volume,
  COALESCE(SUM(quote_out) FILTER (WHERE direction = 'sell'), 0)::text AS sell_quote_volume
FROM classified_events;

-- name: ListProjectSwapActivityBlocks :many
WITH mapped_events AS (
  SELECT
    event.id AS event_id,
    event.project_swap_block_id,
    event.transaction_hash,
    event.transaction_index,
    event.log_index,
    event.tx_from,
    CASE
      WHEN sqlc.arg('base_is_token0')::boolean THEN event.amount0_in
      ELSE event.amount1_in
    END AS base_in,
    CASE
      WHEN sqlc.arg('base_is_token0')::boolean THEN event.amount0_out
      ELSE event.amount1_out
    END AS base_out,
    CASE
      WHEN sqlc.arg('base_is_token0')::boolean THEN event.amount1_in
      ELSE event.amount0_in
    END AS quote_in,
    CASE
      WHEN sqlc.arg('base_is_token0')::boolean THEN event.amount1_out
      ELSE event.amount0_out
    END AS quote_out
  FROM project_swap_event AS event
  WHERE event.project_swap_pair_id = @project_swap_pair_id
),
classified_events AS (
  SELECT
    mapped_events.*,
    CASE
      WHEN quote_in > 0 AND base_out > 0 AND base_in = 0 AND quote_out = 0 THEN 'buy'
      WHEN base_in > 0 AND quote_out > 0 AND quote_in = 0 AND base_out = 0 THEN 'sell'
      ELSE 'complex'
    END::text AS direction
  FROM mapped_events
),
priced_events AS (
  SELECT
    classified_events.*,
    CASE direction
      WHEN 'buy' THEN
        round(
          (quote_in * power(10::numeric, sqlc.arg('base_decimals')::int))::numeric(1000, 400)
          / NULLIF(
            (base_out * power(10::numeric, sqlc.arg('quote_decimals')::int))::numeric(1000, 400),
            0
          ),
          100
        )
      WHEN 'sell' THEN
        round(
          (quote_out * power(10::numeric, sqlc.arg('base_decimals')::int))::numeric(1000, 400)
          / NULLIF(
            (base_in * power(10::numeric, sqlc.arg('quote_decimals')::int))::numeric(1000, 400),
            0
          ),
          100
        )
      ELSE NULL
    END AS effective_price
  FROM classified_events
),
block_aggregates AS (
  SELECT
    block.sample_index,
    block.block_number,
    block.block_time,
    COUNT(event.event_id)::bigint AS event_count,
    COUNT(DISTINCT event.transaction_hash)::bigint AS transaction_count,
    COUNT(DISTINCT event.tx_from)::bigint AS transaction_origin_count,
    COUNT(event.event_id) FILTER (WHERE event.direction = 'buy')::bigint AS buy_event_count,
    COUNT(event.event_id) FILTER (WHERE event.direction = 'sell')::bigint AS sell_event_count,
    COUNT(event.event_id) FILTER (WHERE event.direction = 'complex')::bigint AS complex_event_count,
    COALESCE(SUM(event.base_in), 0)::text AS base_in,
    COALESCE(SUM(event.base_out), 0)::text AS base_out,
    COALESCE(SUM(event.quote_in), 0)::text AS quote_in,
    COALESCE(SUM(event.quote_out), 0)::text AS quote_out,
    COALESCE(SUM(event.quote_in) FILTER (WHERE event.direction = 'buy'), 0)::text AS buy_quote_volume,
    COALESCE(SUM(event.quote_out) FILTER (WHERE event.direction = 'sell'), 0)::text AS sell_quote_volume,
    (
      array_agg(event.effective_price ORDER BY event.transaction_index, event.log_index, event.event_id)
      FILTER (WHERE event.effective_price IS NOT NULL)
    )[1] AS open_price,
    MAX(event.effective_price) AS high_price,
    MIN(event.effective_price) AS low_price,
    (
      array_agg(event.effective_price ORDER BY event.transaction_index DESC, event.log_index DESC, event.event_id DESC)
      FILTER (WHERE event.effective_price IS NOT NULL)
    )[1] AS close_price,
    (
      (
        SUM(
          CASE event.direction
            WHEN 'buy' THEN event.quote_in
            WHEN 'sell' THEN event.quote_out
            ELSE 0
          END
        ) * power(10::numeric, sqlc.arg('base_decimals')::int)
      )::numeric(1000, 400)
      / NULLIF(
        (
          SUM(
            CASE event.direction
              WHEN 'buy' THEN event.base_out
              WHEN 'sell' THEN event.base_in
              ELSE 0
            END
          ) * power(10::numeric, sqlc.arg('quote_decimals')::int)
        )::numeric(1000, 400),
        0
      )
    ) AS vwap_price
  FROM project_swap_block AS block
  LEFT JOIN priced_events AS event
    ON event.project_swap_block_id = block.id
  WHERE block.project_swap_pair_id = @project_swap_pair_id
  GROUP BY block.id, block.sample_index, block.block_number, block.block_time
),
blocks_with_gaps AS (
  SELECT
    block_aggregates.*,
    COALESCE(
      block_number - LAG(block_number) OVER (ORDER BY sample_index),
      0
    )::bigint AS previous_block_gap,
    COALESCE(
      block_time - LAG(block_time) OVER (ORDER BY sample_index),
      0
    )::bigint AS previous_time_gap_seconds
  FROM block_aggregates
)
SELECT
  sample_index,
  block_number,
  block_time,
  previous_block_gap,
  previous_time_gap_seconds,
  event_count,
  transaction_count,
  transaction_origin_count,
  buy_event_count,
  sell_event_count,
  complex_event_count,
  base_in,
  base_out,
  quote_in,
  quote_out,
  buy_quote_volume,
  sell_quote_volume,
  COALESCE(
    trim(trailing '.' FROM trim(trailing '0' FROM round(open_price, 100)::text)),
    ''
  )::text AS open_price,
  COALESCE(
    trim(trailing '.' FROM trim(trailing '0' FROM round(high_price, 100)::text)),
    ''
  )::text AS high_price,
  COALESCE(
    trim(trailing '.' FROM trim(trailing '0' FROM round(low_price, 100)::text)),
    ''
  )::text AS low_price,
  COALESCE(
    trim(trailing '.' FROM trim(trailing '0' FROM round(close_price, 100)::text)),
    ''
  )::text AS close_price,
  COALESCE(
    trim(trailing '.' FROM trim(trailing '0' FROM round(vwap_price, 100)::text)),
    ''
  )::text AS vwap_price
FROM blocks_with_gaps
ORDER BY sample_index;

-- name: CountProjectSwapEvents :one
SELECT COUNT(*)::bigint
FROM project_swap_event AS event
JOIN project_swap_block AS block
  ON block.id = event.project_swap_block_id
  AND block.project_swap_pair_id = event.project_swap_pair_id
JOIN project_swap_pair AS pair
  ON pair.id = event.project_swap_pair_id
WHERE pair.project_id = @project_id
  AND pair.pair_kind = @pair_kind
  AND block.block_number = @block_number;

-- name: ListProjectSwapEvents :many
SELECT
  event.id,
  event.transaction_hash,
  event.transaction_index,
  event.log_index,
  event.tx_from,
  event.sender,
  event.to_address,
  event.amount0_in::text AS amount0_in,
  event.amount1_in::text AS amount1_in,
  event.amount0_out::text AS amount0_out,
  event.amount1_out::text AS amount1_out
FROM project_swap_event AS event
JOIN project_swap_block AS block
  ON block.id = event.project_swap_block_id
  AND block.project_swap_pair_id = event.project_swap_pair_id
JOIN project_swap_pair AS pair
  ON pair.id = event.project_swap_pair_id
WHERE pair.project_id = @project_id
  AND pair.pair_kind = @pair_kind
  AND block.block_number = @block_number
ORDER BY event.transaction_index, event.log_index, event.id
LIMIT @page_limit
OFFSET @page_offset;
