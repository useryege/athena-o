#!/usr/bin/env bash

set -u

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
api_url="${SERVICE_CORE_ACCEPTANCE_API_URL:-http://127.0.0.1:6776/base/health}"
admin_url="${SERVICE_CORE_ACCEPTANCE_ADMIN_URL:-http://127.0.0.1:5173/admin/login}"
team_url="${SERVICE_CORE_ACCEPTANCE_TEAM_URL:-http://127.0.0.1:5174/}"
targets=",${SERVICE_CORE_ACCEPTANCE_TARGETS:-api,admin,team,mysql,redis,minio},"
playwright_package="${SERVICE_CORE_ACCEPTANCE_PLAYWRIGHT_PACKAGE:-$repo_root/e2e/node_modules/playwright/package.json}"
chrome_bin="${SERVICE_CORE_ACCEPTANCE_CHROME_BIN:-/usr/bin/google-chrome}"
goreman_bin="$(command -v goreman 2>/dev/null || true)"
if [[ -z "$goreman_bin" && -x "$HOME/go/bin/goreman" ]]; then
  goreman_bin="$HOME/go/bin/goreman"
fi

http_status() {
  curl -sS -o /dev/null -w '%{http_code}' --max-time 3 "$1" 2>/dev/null || printf '000'
}

target_enabled() {
  [[ "$targets" == *",$1,"* ]]
}

port_ready() {
  ss -ltn "( sport = :$1 )" 2>/dev/null | grep -q LISTEN
}

api_status="$(http_status "$api_url")"
admin_status="$(http_status "$admin_url")"
team_status="$(http_status "$team_url")"
goreman_ready=false
mysql_ready=false
redis_ready=false
minio_ready=false
playwright_ready=false
chrome_ready=false

if [[ -n "$goreman_bin" ]] && ss -ltn '( sport = :8555 )' 2>/dev/null | grep -q LISTEN; then goreman_ready=true; fi
if port_ready 23306; then mysql_ready=true; fi
if port_ready 26379; then redis_ready=true; fi
if port_ready 19000; then minio_ready=true; fi
if [[ -f "$playwright_package" ]]; then playwright_ready=true; fi
if [[ -x "$chrome_bin" ]]; then chrome_ready=true; fi

ready=true
if target_enabled api && [[ "$api_status" != "200" ]]; then ready=false; fi
if target_enabled admin && [[ "$admin_status" != "200" ]]; then ready=false; fi
if target_enabled team && [[ "$team_status" != "200" ]]; then ready=false; fi
if target_enabled mysql && [[ "$mysql_ready" != true ]]; then ready=false; fi
if target_enabled redis && [[ "$redis_ready" != true ]]; then ready=false; fi
if target_enabled minio && [[ "$minio_ready" != true ]]; then ready=false; fi
if [[ "$goreman_ready" != true || "$playwright_ready" != true || "$chrome_ready" != true ]]; then ready=false; fi

jq -n \
  --arg repo_root "$repo_root" \
  --arg api_status "$api_status" \
  --arg admin_status "$admin_status" \
  --arg team_status "$team_status" \
  --arg targets "${targets#,}" \
  --arg goreman_bin "$goreman_bin" \
  --arg playwright_package "$playwright_package" \
  --arg chrome_bin "$chrome_bin" \
  --argjson ready "$ready" \
  --argjson goreman_ready "$goreman_ready" \
  --argjson mysql_ready "$mysql_ready" \
  --argjson redis_ready "$redis_ready" \
  --argjson minio_ready "$minio_ready" \
  --argjson playwright_ready "$playwright_ready" \
  --argjson chrome_ready "$chrome_ready" \
  '{ready:$ready,repo_root:$repo_root,targets:($targets | rtrimstr(",") | split(",")),services:{api:$api_status,admin:$admin_status,team:$team_status,goreman:$goreman_ready,mysql:$mysql_ready,redis:$redis_ready,minio:$minio_ready},dependencies:{goreman_bin:$goreman_bin,playwright_package:$playwright_package,playwright:$playwright_ready,chrome_bin:$chrome_bin,chrome:$chrome_ready}}'

if [[ "$ready" != true ]]; then exit 2; fi
