package store

import (
	"github.com/useryege/athena/internal/serviceschema"
	"github.com/useryege/athena/util/db/postgres"
)

// Schema declares the existing migration and columns consumed by this service.
func Schema() serviceschema.Spec {
	return serviceschema.Spec{Module: "wallet", DSN: func() string { return postgres.DSN("ATHENA_WALLET_POSTGRES_DSN", "wallet") }, Migrations: migrations, Relations: map[string][]string{
		"public.wallets": {"id", "owner_account_id", "wallet_type", "address", "address_key", "remark", "source", "private_key_ciphertext", "avatar_preset_id", "avatar_object_key", "avatar_content_type", "avatar_etag", "avatar_size_bytes", "revision", "created_at", "updated_at"},
	}}
}
