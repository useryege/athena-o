# Declarative full-stack inventory. Start with make run; stop with make stop.
# Procfile has no executable entries: devruntime owns every process and resource.
# Business processes use immutable, per-run built binaries under .run/instances/:
# trader-sync: athena-trader-sync (account-state PostgreSQL; default 8122)
# profit-sharing: athena-profit-sharing (profit_sharing PostgreSQL; default 8108)
# notification: athena-notification (account-state PostgreSQL; default 8086)
# wallet: athena-wallet (wallet PostgreSQL; default 8088)
# ui: node ui/node_modules/vite/bin/vite.js (default 4000)
# api-server: athena-server (account-state PostgreSQL, Redis, MinIO; default 8080)
# Infrastructure: owned PostgreSQL 16, Redis 7.2, MinIO; dynamic loopback ports.
# Other business modules are not enabled by this full-stack selection.
# Solana discovery remains paused in the default full stack.
# Explicit Solana profiles use Procfile.solana-discovery or Procfile.solana-preview.
