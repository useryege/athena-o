#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  render_make_targets.sh --vendor <name> [--module-path util/<vendor>] [--write <Makefile>] [--dry-run]

Examples:
  render_make_targets.sh --vendor demoapi --dry-run
  render_make_targets.sh --vendor demoapi --module-path util/demoapi --write Makefile
USAGE
}

vendor=""
module_path=""
write_file=""
dry_run="0"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --vendor)
      vendor="${2:-}"
      shift 2
      ;;
    --module-path)
      module_path="${2:-}"
      shift 2
      ;;
    --write)
      write_file="${2:-}"
      shift 2
      ;;
    --dry-run)
      dry_run="1"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown arg: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ -z "$vendor" ]]; then
  echo "--vendor is required" >&2
  exit 1
fi

vendor_norm="$(echo "$vendor" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9-' '-')"
vendor_norm="${vendor_norm#-}"
vendor_norm="${vendor_norm%-}"
if [[ -z "$vendor_norm" ]]; then
  echo "invalid vendor" >&2
  exit 1
fi

if [[ -z "$module_path" ]]; then
  module_path="util/${vendor_norm}"
fi

block=$(cat <<MAKE
# BEGIN AUTO ${vendor_norm}
.PHONY: ${vendor_norm}-sync-docs ${vendor_norm}-snapshot ${vendor_norm}-snapshot-dry ${vendor_norm}-test-unit ${vendor_norm}-test-integration

${vendor_norm}-sync-docs:
	make -C ${module_path} sync-docs

${vendor_norm}-snapshot:
	go run ./${module_path}/cmd-${vendor_norm}-public-read-snapshot

${vendor_norm}-snapshot-dry:
	go run ./${module_path}/cmd-${vendor_norm}-public-read-snapshot --dry-run

${vendor_norm}-test-unit:
	go test ./${module_path}

${vendor_norm}-test-integration:
	go test -count=1 -v ./${module_path} -run '^TestIntegration'
# END AUTO ${vendor_norm}
MAKE
)

if [[ "$dry_run" == "1" || -z "$write_file" ]]; then
  echo "$block"
  exit 0
fi

if [[ ! -f "$write_file" ]]; then
  echo "$block" > "$write_file"
  echo "created: $write_file"
  exit 0
fi

start="# BEGIN AUTO ${vendor_norm}"
end="# END AUTO ${vendor_norm}"
if grep -qF "$start" "$write_file"; then
  awk -v start="$start" -v end="$end" -v repl="$block" '
    BEGIN { in_block=0; replaced=0 }
    {
      if (index($0, start) == 1) {
        if (!replaced) {
          print repl
          replaced=1
        }
        in_block=1
        next
      }
      if (in_block && index($0, end) == 1) {
        in_block=0
        next
      }
      if (!in_block) {
        print $0
      }
    }
    END {
      if (!replaced) {
        print repl
      }
    }
  ' "$write_file" > "${write_file}.tmp"
  mv "${write_file}.tmp" "$write_file"
else
  {
    echo ""
    echo "$block"
  } >> "$write_file"
fi

echo "updated: $write_file"
