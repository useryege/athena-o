package store

import (
	"github.com/useryege/athena/internal/serviceschema"
	"github.com/useryege/athena/util/db/postgres"
)

// Schema declares the existing migration and columns consumed by this service.
func Schema() serviceschema.Spec {
	return serviceschema.Spec{Module: "managed-oo", DSN: func() string { return postgres.DSN("ATHENA_MANAGED_OO_POSTGRES_DSN", "managed_oo") }, Migrations: migrations, Relations: map[string][]string{
		"public.managed_oo_chain_log_cursor":          {"sync_name", "contract_address", "topic", "last_block_number", "last_polled_at", "created_at", "updated_at"},
		"public.managed_oo_propose_price_log":         {"tx_hash", "log_index", "block_number", "block_hash", "tx_index", "contract_address", "topic", "requester", "proposer", "identifier", "request_timestamp", "ancillary_data_hex", "ancillary_data_text", "market_id", "proposed_price", "expiration_timestamp", "currency", "raw_topics", "raw_data", "fetched_at", "created_at", "updated_at"},
		"public.managed_oo_dispute_price_log":         {"tx_hash", "log_index", "block_number", "block_hash", "tx_index", "contract_address", "topic", "requester", "proposer", "disputer", "identifier", "request_timestamp", "ancillary_data_hex", "ancillary_data_text", "market_id", "proposed_price", "raw_topics", "raw_data", "fetched_at", "created_at", "updated_at"},
		"public.managed_oo_market":                    {"market_id", "condition_id", "slug", "event_slug", "question", "description", "resolution_source", "question_id", "sports_market_type", "group_item_title", "image", "icon", "outcomes", "outcome_prices", "clob_token_ids", "active", "closed", "archived", "restricted", "enable_order_book", "accepting_orders", "volume", "start_date", "end_date", "created_at_gamma", "updated_at_gamma", "tags", "raw", "fetch_status", "last_error", "last_error_at", "fetched_at", "created_at", "updated_at"},
		"public.managed_oo_market_label":              {"market_id", "label", "tag_id", "slug", "position", "fetched_at", "created_at", "updated_at"},
		"public.managed_oo_propose_price_alert_state": {"tx_hash", "log_index", "notification_id", "notified_at", "created_at", "updated_at"},
		"public.managed_oo_dispute_price_alert_state": {"tx_hash", "log_index", "notification_id", "notified_at", "created_at", "updated_at"},
	}}
}
