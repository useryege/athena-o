#!/usr/bin/env bash
# Real container acceptance. Uses only resources bearing a unique test label.
set -euo pipefail
image="${TRADER_SYNC_IMAGE:-athena-trader-sync:local}"
postgres_image="${TEST_POSTGRES_IMAGE:-postgres:16}"
run="athena-ts-image-test-$(date +%s)-$$"
tmp="$(mktemp -d)"
network="${run}-net"
postgres="${run}-postgres"
service="${run}-service"
cleanup() {
  status=$?
  if ((status)); then docker logs "$service" 2>&1 || true; fi
  for name in "$service" "$postgres"; do
    if [[ "$(docker inspect --format '{{index .Config.Labels "athena.test.run"}}' "$name" 2>/dev/null || true)" == "$run" ]]; then docker rm -f "$name" >/dev/null; fi
  done
  if [[ "$(docker network inspect --format '{{index .Labels "athena.test.run"}}' "$network" 2>/dev/null || true)" == "$run" ]]; then docker network rm "$network" >/dev/null; fi
  rm -rf "$tmp"
  exit "$status"
}
trap cleanup EXIT
printf 'test resources: %s; image: %s\n' "$run" "$image"
docker image inspect "$image" >/dev/null
[[ "$(docker image inspect --format '{{.Config.User}}' "$image")" == '999:999' ]]
[[ "$(docker image inspect --format '{{json .Config.Cmd}}' "$image")" == '["athena-trader-sync"]' ]]
# Runtime layer contains two executables, no UI, seed binaries or application secrets.
docker run --rm --entrypoint sh "$image" -ec '
  test "$(ls /usr/local/bin | wc -l)" -eq 2
  test -x /usr/local/bin/athena-trader-sync
  test -x /usr/local/bin/athena-account-state-migrate
  test ! -e /src && test ! -e /app && test ! -e /ui && test ! -e /.env
  ! command -v node
  ! command -v athena
'
mkdir "$tmp/certs"
openssl req -x509 -newkey rsa:2048 -nodes -keyout "$tmp/ca.key" -out "$tmp/certs/ca.crt" -days 1 -subj '/CN=Athena image test CA' >/dev/null 2>&1
openssl req -newkey rsa:2048 -nodes -keyout "$tmp/certs/server.key" -out "$tmp/server.csr" -subj '/CN=athena-trader-sync' >/dev/null 2>&1
printf 'subjectAltName=DNS:athena-trader-sync\nextendedKeyUsage=serverAuth\n' >"$tmp/extensions"
openssl x509 -req -in "$tmp/server.csr" -CA "$tmp/certs/ca.crt" -CAkey "$tmp/ca.key" -CAcreateserial -out "$tmp/certs/server.crt" -days 1 -extfile "$tmp/extensions" >/dev/null 2>&1
openssl req -x509 -newkey rsa:2048 -nodes -keyout "$tmp/wrong.key" -out "$tmp/certs/wrong-ca.crt" -days 1 -subj '/CN=Untrusted test CA' >/dev/null 2>&1
printf 'image-test-token-0123456789abcdef0123456789abcdef' >"$tmp/certs/token"
printf 'image-test-cursor-abcdef0123456789abcdef0123456789' >"$tmp/certs/cursor"
chmod 755 "$tmp/certs"
chmod 644 "$tmp/certs/"*
# This internal network has no host port and cannot reach external Telegram/provider endpoints.
docker network create --internal --label "athena.test.run=$run" "$network" >/dev/null
docker run -d --name "$postgres" --label "athena.test.run=$run" --network "$network" --network-alias postgres -e POSTGRES_HOST_AUTH_METHOD=trust "$postgres_image" >/dev/null
ready=false
for ((attempt=0; attempt<120; attempt++)); do
  if docker exec "$postgres" pg_isready -U postgres >/dev/null 2>&1; then ready=true; break; fi
  sleep 1
done
[[ "$ready" == true ]]
printf 'ATHENA_ACCOUNT_STATE_POSTGRES_DSN=postgres://postgres@postgres:5432/postgres?sslmode=disable\n' >"$tmp/schema.env"
chmod 600 "$tmp/schema.env"
for action in up verify; do
  docker run --rm --network "$network" --env-file "$tmp/schema.env" --entrypoint athena-account-state-migrate "$image" "$action"
done
cp "$tmp/schema.env" "$tmp/service.env"
cat >>"$tmp/service.env" <<'ENV'
ATHENA_URL=https://athena.test
ATHENA_TRADER_SYNC_HTTP_URL=http://127.0.0.1:1
ATHENA_TRADER_SYNC_WSS_URL=ws://127.0.0.1:1
ATHENA_TRADER_SYNC_LISTEN_ADDRESS=0.0.0.0:8122
ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN_FILE=/certs/token
ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY_FILE=/certs/cursor
ATHENA_TRADER_SYNC_TLS_CERT_FILE=/certs/server.crt
ATHENA_TRADER_SYNC_TLS_KEY_FILE=/certs/server.key
ENV
docker run -d --name "$service" --label "athena.test.run=$run" --network "$network" --network-alias athena-trader-sync --env-file "$tmp/service.env" -v "$tmp/certs:/certs:ro" "$image" >/dev/null
health() {
  # A separate process with an empty environment proves no DB/token/provider config is needed.
  docker run --rm --network "container:$service" -v "$tmp/certs:/certs:ro" --entrypoint /usr/bin/env "$image" -i /usr/local/bin/athena-trader-sync health --target 127.0.0.1:8122 --tls-ca-file "/certs/$1" --tls-server-name "$2" --timeout 2s
}
ready=false
for ((attempt=0; attempt<30; attempt++)); do
  if health ca.crt athena-trader-sync >"$tmp/health.log" 2>&1; then ready=true; break; fi
  sleep 1
done
[[ "$ready" == true ]]
if [[ "${TEST_HEALTH_ONLY:-false}" != true ]]; then
if health wrong-ca.crt athena-trader-sync >"$tmp/wrong-ca.log" 2>&1; then echo 'wrong CA accepted' >&2; exit 1; fi
if health ca.crt wrong-name >"$tmp/wrong-name.log" 2>&1; then echo 'wrong certificate name accepted' >&2; exit 1; fi
grep -qi 'certificate' "$tmp/wrong-ca.log"
grep -qi 'certificate' "$tmp/wrong-name.log"
fi
# Docker sends TERM to tini; the process must drain within its 30-second budget.
docker stop -t 40 "$service" >/dev/null
[[ "$(docker inspect --format '{{.State.ExitCode}}' "$service")" == 0 ]]
printf 'real image: schema up/verify, TLS health without business credentials, non-root minimal filesystem, graceful stop passed (health-only=%s)\n' "${TEST_HEALTH_ONLY:-false}"
