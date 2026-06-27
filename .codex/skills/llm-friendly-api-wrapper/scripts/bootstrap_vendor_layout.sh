#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  bootstrap_vendor_layout.sh --vendor <name> [--module-root util] [--target-root .] [--docs-host https://docs.example.com] [--docs-index /llms.txt] [--force]

Example:
  bootstrap_vendor_layout.sh --vendor demoapi --docs-host https://docs.demoapi.com
USAGE
}

vendor=""
module_root="util"
target_root="."
docs_host="https://docs.example.com"
docs_index="/llms.txt"
force="0"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --vendor)
      vendor="${2:-}"
      shift 2
      ;;
    --module-root)
      module_root="${2:-}"
      shift 2
      ;;
    --target-root)
      target_root="${2:-}"
      shift 2
      ;;
    --docs-host)
      docs_host="${2:-}"
      shift 2
      ;;
    --docs-index)
      docs_index="${2:-}"
      shift 2
      ;;
    --force)
      force="1"
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
  usage >&2
  exit 1
fi

vendor_norm="$(echo "$vendor" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9-' '-')"
vendor_norm="${vendor_norm#-}"
vendor_norm="${vendor_norm%-}"
if [[ -z "$vendor_norm" ]]; then
  echo "invalid vendor after normalization" >&2
  exit 1
fi

module_dir="${target_root%/}/${module_root}/${vendor_norm}"
cmd_dir="${module_dir}/cmd-${vendor_norm}-public-read-snapshot"
docs_dir="${module_dir}/${vendor_norm}-docs"
rr_dir="${module_dir}/request-response/latest"

if [[ -d "$module_dir" && "$force" != "1" ]]; then
  echo "target exists: $module_dir (use --force to continue)" >&2
  exit 1
fi

mkdir -p "$cmd_dir" "$docs_dir" "$rr_dir"

cat > "${module_dir}/sync-docs.sh" <<SYNC
#!/usr/bin/env bash
set -euo pipefail

readonly DOCS_HOST="${docs_host}"
readonly INDEX_URL="\${DOCS_HOST}${docs_index}"

script_dir="\$(cd "\$(dirname "\${BASH_SOURCE[0]}")" && pwd)"
target_dir="\${script_dir}/${vendor_norm}-docs"
tmp_urls="\$(mktemp)"
cleanup() { rm -f "\${tmp_urls}"; }
trap cleanup EXIT

echo "Fetching index: \${INDEX_URL}"
index_content="\$(curl -fsSL "\${INDEX_URL}")"
printf '%s\n' "\${index_content}" \
  | grep -oE 'https?://[^)[:space:]]+\\.md' \
  | sort -u > "\${tmp_urls}" || true

url_count="\$(wc -l < "\${tmp_urls}" | tr -d ' ')"
if [[ "\${url_count}" == "0" ]]; then
  echo "No markdown URLs were found in \${INDEX_URL}" >&2
  exit 1
fi

echo "Found \${url_count} markdown pages"
rm -rf "\${target_dir}"
mkdir -p "\${target_dir}"

while IFS= read -r url; do
  rel_path="\${url#\${DOCS_HOST}/}"
  mkdir -p "\${target_dir}/\$(dirname "\${rel_path}")"
  curl -fsSL "\${url}" -o "\${target_dir}/\${rel_path}"
done < "\${tmp_urls}"

echo "Download completed"
SYNC
chmod +x "${module_dir}/sync-docs.sh"

cat > "${module_dir}/Makefile" <<MAKEFILE
.DEFAULT_GOAL := sync-docs

.PHONY: sync-docs snapshot-public-read snapshot-public-read-dry test-unit
sync-docs:
	./sync-docs.sh

snapshot-public-read:
	go run ./${module_root}/${vendor_norm}/cmd-${vendor_norm}-public-read-snapshot

snapshot-public-read-dry:
	go run ./${module_root}/${vendor_norm}/cmd-${vendor_norm}-public-read-snapshot --dry-run

test-unit:
	go test ./${module_root}/${vendor_norm}
MAKEFILE

cat > "${module_dir}/README.md" <<README
# ${vendor_norm} API Wrapper

## Sync docs

\`\`\`bash
make -C ${module_root}/${vendor_norm} sync-docs
\`\`\`

## Snapshot public read endpoints

\`\`\`bash
make -C ${module_root}/${vendor_norm} snapshot-public-read-dry
make -C ${module_root}/${vendor_norm} snapshot-public-read
\`\`\`

## Run tests

\`\`\`bash
make -C ${module_root}/${vendor_norm} test-unit
\`\`\`
README

cat > "${cmd_dir}/main.go" <<GO
package main

import "fmt"

func main() {
	fmt.Println("TODO: implement public read snapshot command for ${vendor_norm}")
}
GO

touch "${docs_dir}/.gitkeep"
touch "${rr_dir}/.gitkeep"

echo "created module layout: ${module_dir}"
echo "next steps:"
echo "  1) implement ${cmd_dir}/main.go"
echo "  2) update ${module_dir}/README.md"
echo "  3) run make -C ${module_root}/${vendor_norm} sync-docs"
