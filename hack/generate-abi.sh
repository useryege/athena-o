#!/bin/bash
set -euo pipefail

ROOT="$(CDPATH='' cd -- "$(dirname "$0")/.." && pwd -P)"
ABI_ROOT="${ROOT}/pkg/abi"

if ! command -v abigen >/dev/null 2>&1; then
  echo "abigen is required. Install it with:" >&2
  echo "  go install github.com/ethereum/go-ethereum/cmd/abigen@v1.17.2" >&2
  exit 1
fi

if [[ ! -d "${ABI_ROOT}" ]]; then
  echo "ABI directory not found: ${ABI_ROOT}" >&2
  exit 1
fi

generated=0
skipped=0

for dir in "${ABI_ROOT}"/*; do
  [[ -d "${dir}" ]] || continue

  name="$(basename "${dir}")"
  abi_file="${dir}/a.abi"
  bin_file="${dir}/b.bin"
  out_file="${dir}/${name}.go"

  if [[ ! -f "${abi_file}" || ! -f "${bin_file}" ]]; then
    continue
  fi

  if [[ -f "${out_file}" ]]; then
    echo "skip ${out_file#${ROOT}/}: already exists"
    skipped=$((skipped + 1))
    continue
  fi

  echo "generate ${out_file#${ROOT}/}"
  abigen \
    --abi "${abi_file}" \
    --bin "${bin_file}" \
    --pkg "${name}" \
    --type "${name}" \
    --out "${out_file}"
  generated=$((generated + 1))
done

echo "abi generation complete: generated=${generated} skipped=${skipped}"
