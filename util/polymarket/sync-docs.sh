#!/usr/bin/env bash

set -euo pipefail

readonly DOCS_HOST="https://docs.polymarket.com"
readonly INDEX_URL="${DOCS_HOST}/llms.txt"

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
target_dir="${script_dir}/polymarket-docs"

tmp_urls="$(mktemp)"
cleanup() {
  rm -f "${tmp_urls}"
}
trap cleanup EXIT

echo "Fetching index: ${INDEX_URL}"
index_content="$(curl -fsSL "${INDEX_URL}")"
printf '%s\n' "${index_content}" \
  | grep -oE 'https://docs\.polymarket\.com/[^)[:space:]]+\.md' \
  | sort -u > "${tmp_urls}" || true

url_count="$(wc -l < "${tmp_urls}" | tr -d ' ')"
if [[ "${url_count}" == "0" ]]; then
  echo "No markdown URLs were found in ${INDEX_URL}" >&2
  exit 1
fi

echo "Found ${url_count} markdown pages"
echo "Rebuilding target directory: ${target_dir}"
rm -rf "${target_dir}"
mkdir -p "${target_dir}"

while IFS= read -r url; do
  rel_path="${url#${DOCS_HOST}/}"
  mkdir -p "${target_dir}/$(dirname "${rel_path}")"
  curl -fsSL "${url}" -o "${target_dir}/${rel_path}"
done < "${tmp_urls}"

downloaded_count="$(find "${target_dir}" -type f -name '*.md' | wc -l | tr -d ' ')"
echo "Download completed: ${downloaded_count} files"
