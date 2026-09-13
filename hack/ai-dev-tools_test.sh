#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
temporary=$(mktemp -d)
trap 'rm -rf "$temporary"' EXIT

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

mkdir -p "$temporary/bin" "$temporary/artifacts"

# A missing repository-local binary must fail before any network or install action.
if AI_DEV_TOOLS_BIN_DIR="$temporary/bin" "$root/hack/ai-dev-tools.sh" check >"$temporary/check.out" 2>&1; then
  fail 'check accepted missing development tools'
fi
grep -q 'install-ai-dev-tools' "$temporary/check.out" || fail 'check did not explain installation'

# A security finding or scanner error must survive tee, with its complete output retained.
cat >"$temporary/bin/govulncheck" <<'STUB'
#!/usr/bin/env bash
printf 'finding from govulncheck\n'
printf 'scanner diagnostic\n' >&2
exit 7
STUB
chmod +x "$temporary/bin/govulncheck"
set +e
AI_DEV_TOOLS_BIN_DIR="$temporary/bin" AI_DEV_TOOLS_ARTIFACT_DIR="$temporary/artifacts" \
  "$root/hack/ai-dev-tools.sh" vuln-check >"$temporary/vuln.out" 2>&1
status=$?
set -e
[[ "$status" -eq 7 ]] || fail "vuln-check returned $status instead of scanner status 7"
grep -q 'finding from govulncheck' "$temporary/vuln.out" || fail 'scanner stdout was lost'
grep -q 'scanner diagnostic' "$temporary/vuln.out" || fail 'scanner stderr was lost'
artifact=$(find "$temporary/artifacts" -type f -name 'govulncheck-*.log' -print -quit)
[[ -n "$artifact" ]] || fail 'vuln-check did not retain an artifact'
grep -q 'finding from govulncheck' "$artifact" || fail 'artifact lacks scanner stdout'
grep -q 'scanner diagnostic' "$artifact" || fail 'artifact lacks scanner stderr'

# The readiness gate must follow the declared stable Node engine range.
real_node=$(command -v node)
cat >"$temporary/bin/node" <<'STUB'
#!/usr/bin/env bash
if [[ ${1:-} == -e ]]; then
  case $2 in
    *process.versions.node*)
      exec "$REAL_NODE" -e 'Object.defineProperty(process.versions, "node", {value: process.env.TEST_NODE_VERSION}); eval(process.argv[1])' "$2"
      ;;
    *) exit 0 ;;
  esac
fi
exec "$REAL_NODE" "$@"
STUB
cat >"$temporary/bin/version-stub" <<'STUB'
#!/usr/bin/env bash
case ${0##*/} in
  shellcheck) echo 'version: 0.11.0' ;;
  grpcurl) echo 'grpcurl v1.9.4' ;;
  govulncheck) echo 'govulncheck v1.7.0' ;;
  psql) echo 'psql (PostgreSQL) 16.1' ;;
  pg_isready) echo 'pg_isready (PostgreSQL) 16.1' ;;
  yarn) echo '1.22.22' ;;
esac
STUB
chmod +x "$temporary/bin/node" "$temporary/bin/version-stub"
for tool in shellcheck grpcurl psql pg_isready yarn; do
  ln -s version-stub "$temporary/bin/$tool"
done
rm "$temporary/bin/govulncheck"
ln -s version-stub "$temporary/bin/govulncheck"

for version in 24.14.1 24.15.0; do
  REAL_NODE="$real_node" TEST_NODE_VERSION="$version" PATH="$temporary/bin:$PATH" \
    AI_DEV_TOOLS_BIN_DIR="$temporary/bin" "$root/hack/ai-dev-tools.sh" check >"$temporary/node-check.out" 2>&1 ||
    fail "stable Node $version was rejected"
  grep -q '@axe-core/playwright 4.13.0: ready' "$temporary/node-check.out" ||
    fail "stable Node $version did not load the UI tool"
done
for version in 22.9.0 23.0.0 24.14.0 25.0.0 24.15.0-rc.0; do
  if REAL_NODE="$real_node" TEST_NODE_VERSION="$version" PATH="$temporary/bin:$PATH" \
    AI_DEV_TOOLS_BIN_DIR="$temporary/bin" "$root/hack/ai-dev-tools.sh" check >"$temporary/node-check.out" 2>&1; then
    fail "unsupported Node $version was accepted"
  fi
  grep -q '@axe-core/playwright 4.13.0: missing' "$temporary/node-check.out" ||
    fail "unsupported Node $version did not report the UI tool as unavailable"
done

echo 'PASS: missing-tool guidance, scanner exit/artifact behavior, and Node engine boundaries'
