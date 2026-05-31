#!/usr/bin/env bash

set -euo pipefail

readonly DOCS_HOST="https://docs.polymarket.com"
readonly INDEX_URL="${DOCS_HOST}/llms.txt"
readonly URL_PREFIX="${DOCS_HOST}/api-reference/"

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
target_dir="${script_dir}/polymarket-api-docs"

tmp_urls="$(mktemp)"
cleanup() {
  rm -f "${tmp_urls}"
}
trap cleanup EXIT

echo "Fetching index: ${INDEX_URL}"
curl -fsSL "${INDEX_URL}" \
  | grep -oE 'https://docs\.polymarket\.com/api-reference/[^)]*\.md' \
  | sort -u > "${tmp_urls}"

url_count="$(wc -l < "${tmp_urls}" | tr -d ' ')"
if [[ "${url_count}" == "0" ]]; then
  echo "No api-reference markdown URLs were found in ${INDEX_URL}" >&2
  exit 1
fi

echo "Found ${url_count} markdown pages under /api-reference/"
echo "Rebuilding target directory: ${target_dir}"
rm -rf "${target_dir}"
mkdir -p "${target_dir}"

while IFS= read -r url; do
  path="${url#${URL_PREFIX}}"
  mkdir -p "${target_dir}/$(dirname "${path}")"
  curl -fsSL "${url}" -o "${target_dir}/${path}"
done < "${tmp_urls}"

downloaded_count="$(find "${target_dir}" -type f -name '*.md' | wc -l | tr -d ' ')"
echo "Download completed: ${downloaded_count} files"
