#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  check_public_read_coverage.sh --docs-root <dir> --client-file <file> [--client-file <file> ...] --integration-file <file> [--strict]

Example:
  check_public_read_coverage.sh \
    --docs-root util/polymarket/polymarket-docs/api-reference \
    --client-file util/polymarket/gamma.go \
    --integration-file util/polymarket/gamma_integration_test.go
USAGE
}

docs_root=""
integration_file=""
strict="0"
client_files=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --docs-root)
      docs_root="${2:-}"
      shift 2
      ;;
    --client-file)
      client_files+=("${2:-}")
      shift 2
      ;;
    --integration-file)
      integration_file="${2:-}"
      shift 2
      ;;
    --strict)
      strict="1"
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

if [[ -z "$docs_root" || ${#client_files[@]} -eq 0 || -z "$integration_file" ]]; then
  echo "missing required args" >&2
  usage >&2
  exit 1
fi

[[ -d "$docs_root" ]] || { echo "docs-root not found: $docs_root" >&2; exit 1; }
for f in "${client_files[@]}"; do
  [[ -f "$f" ]] || { echo "client file not found: $f" >&2; exit 1; }
done
[[ -f "$integration_file" ]] || { echo "integration file not found: $integration_file" >&2; exit 1; }

tmp_dir="$(mktemp -d)"
cleanup() { rm -rf "$tmp_dir"; }
trap cleanup EXIT

endpoints_file="$tmp_dir/endpoints.txt"
methods_file="$tmp_dir/methods.txt"
used_methods_file="$tmp_dir/used_methods.txt"
missing_client_file="$tmp_dir/missing_client.txt"
missing_test_file="$tmp_dir/missing_test.txt"

rg -n '^````yaml\s+\S+\s+([A-Za-z]+)\s+\S+' "$docs_root" \
  | sed -E 's#^[^:]+:[0-9]+:````yaml\s+\S+\s+([A-Za-z]+)\s+(\S+)#\1 \2#' \
  | awk '{print toupper($1) " " $2}' \
  | rg '^(GET|POST) ' \
  | sort -u > "$endpoints_file" || true

if [[ ! -s "$endpoints_file" ]]; then
  echo "no endpoints parsed from docs" >&2
  exit 1
fi

: > "$methods_file"
for f in "${client_files[@]}"; do
  rg -o 'func \(c \*[^)]*\) ([A-Z][A-Za-z0-9_]*)\(' "$f" \
    | sed -E 's#.*\) ([A-Z][A-Za-z0-9_]*)\(#\1#' >> "$methods_file" || true
done
sort -u -o "$methods_file" "$methods_file"

rg -o '\.([A-Z][A-Za-z0-9_]*)\(' "$integration_file" \
  | sed -E 's#\.([A-Z][A-Za-z0-9_]*)\(#\1#' \
  | sort -u > "$used_methods_file" || true

: > "$missing_client_file"
: > "$missing_test_file"

while read -r http_method path; do
  [[ -n "$http_method" && -n "$path" ]] || continue

  path_norm="$path"
  path_norm="${path_norm//\{*\}/}"
  path_norm="${path_norm//\// }"
  path_norm="$(echo "$path_norm" | tr -s ' ' ' ' | sed -E 's/^ +| +$//g')"

  matched_method=""
  while read -r m; do
    [[ -n "$m" ]] || continue
    lower="$(echo "$m" | tr '[:upper:]' '[:lower:]')"
    hit="1"
    for tok in $path_norm; do
      case "$tok" in
        ""|api|v1|v2|public|private|data|market|markets|events|tags|comments|profiles|series|search|sports|clob)
          ;;
        *)
          tok_l="$(echo "$tok" | tr '[:upper:]' '[:lower:]')"
          if [[ "$lower" != *"$tok_l"* ]]; then
            hit="0"
            break
          fi
          ;;
      esac
    done

    if [[ "$hit" == "1" ]]; then
      matched_method="$m"
      break
    fi
  done < "$methods_file"

  if [[ -z "$matched_method" ]]; then
    echo "$http_method $path" >> "$missing_client_file"
    continue
  fi

  if ! rg -qx "$matched_method" "$used_methods_file"; then
    echo "$matched_method <= $http_method $path" >> "$missing_test_file"
  fi
done < "$endpoints_file"

echo "coverage summary"
echo "  docs endpoints: $(wc -l < "$endpoints_file" | tr -d ' ')"
echo "  client methods: $(wc -l < "$methods_file" | tr -d ' ')"
echo "  integration used methods: $(wc -l < "$used_methods_file" | tr -d ' ')"

if [[ -s "$missing_client_file" ]]; then
  echo ""
  echo "missing in client (heuristic):"
  cat "$missing_client_file"
fi

if [[ -s "$missing_test_file" ]]; then
  echo ""
  echo "missing in integration tests (heuristic):"
  cat "$missing_test_file"
fi

if [[ "$strict" == "1" && ( -s "$missing_client_file" || -s "$missing_test_file" ) ]]; then
  exit 1
fi
