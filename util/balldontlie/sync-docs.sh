#!/usr/bin/env bash

set -euo pipefail

readonly OPENAPI_URL="https://www.balldontlie.io/openapi/atp.yml"
readonly HTML_URL="https://atp.balldontlie.io/"

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
target_dir="${script_dir}/balldontlie-docs"
openapi_dir="${target_dir}/openapi"
html_dir="${target_dir}/html"
metadata_dir="${target_dir}/metadata"
openapi_file="${openapi_dir}/atp.yml"
html_file="${html_dir}/atp-api.html"
openapi_headers_file="${metadata_dir}/atp-openapi.headers.txt"
html_headers_file="${metadata_dir}/atp-html.headers.txt"

require_text() {
  local file="$1"
  local text="$2"

  if ! grep -Fq "$text" "$file"; then
    echo "Missing required text in ${file}: ${text}" >&2
    exit 1
  fi
}

echo "Rebuilding target directory: ${target_dir}"
rm -rf "${target_dir}"
mkdir -p "${openapi_dir}" "${html_dir}" "${metadata_dir}"

echo "Fetching OpenAPI headers: ${OPENAPI_URL}"
curl -fsSIL "${OPENAPI_URL}" -o "${openapi_headers_file}"

echo "Fetching OpenAPI document: ${OPENAPI_URL}"
curl -fsSL "${OPENAPI_URL}" -o "${openapi_file}"

echo "Fetching HTML headers: ${HTML_URL}"
curl -fsSIL "${HTML_URL}" -o "${html_headers_file}"

echo "Fetching HTML document: ${HTML_URL}"
curl -fsSL "${HTML_URL}" -o "${html_file}"

echo "Validating OpenAPI document"
require_text "${openapi_file}" "openapi: 3.1.0"
require_text "${openapi_file}" "BALLDONTLIE - ATP Tennis API"
require_text "${openapi_file}" "/atp/v1/players"
require_text "${openapi_file}" "/atp/v1/matches"
require_text "${openapi_file}" "/atp/v1/odds"

echo "Download completed"
