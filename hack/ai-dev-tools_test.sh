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

cat >"$temporary/bin/govulncheck" <<'STUB'
#!/usr/bin/env bash
echo 'package load failed' >&2
exit 3
STUB
chmod +x "$temporary/bin/govulncheck"
set +e
AI_DEV_TOOLS_BIN_DIR="$temporary/bin" AI_DEV_TOOLS_ARTIFACT_DIR="$temporary/artifacts" \
  "$root/hack/ai-dev-tools.sh" vuln-check >"$temporary/load-failure.out" 2>&1
status=$?
set -e
[[ "$status" -eq 3 ]] || fail "vuln-check lost package load failure status 3 (got $status)"
grep -q 'package load failed' "$temporary/load-failure.out" || fail 'package load error was lost'
artifact=$(rg -l 'package load failed' "$temporary/artifacts"/govulncheck-*.log | head -1 || true)
[[ -n "$artifact" ]] || fail 'vuln-check did not retain the package load error'
grep -q 'package load failed' "$artifact" || fail 'artifact lacks package load error'

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
  govulncheck) echo "Scanner: govulncheck@v${TEST_GOVULNCHECK_VERSION:-1.7.0}" ;;
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

required_go=$(awk '$1 == "go" { print "go" $2; exit }' "$root/go.mod")
cat >"$temporary/bin/go" <<'STUB'
#!/usr/bin/env bash
if [[ ${1:-} == version && ${2:-} == -m ]]; then
  if [[ ${TEST_GO_METADATA_FAILURE:-} == 1 ]]; then
    echo 'cannot read Go build metadata' >&2
    exit 1
  fi
  printf '%s: %s\n' "$3" "$TEST_COMPILED_GO_VERSION"
  exit 0
fi
if [[ ${1:-} == env && ${2:-} == GOVERSION ]]; then
  echo "${TEST_HOST_GO_VERSION:-go1.27.1}"
  exit 0
fi
echo "unexpected go invocation: $*" >&2
exit 1
STUB
chmod +x "$temporary/bin/go"

for version in 24.14.1 24.15.0; do
  REAL_NODE="$real_node" TEST_NODE_VERSION="$version" TEST_COMPILED_GO_VERSION="$required_go" PATH="$temporary/bin:$PATH" \
    AI_DEV_TOOLS_BIN_DIR="$temporary/bin" "$root/hack/ai-dev-tools.sh" check >"$temporary/node-check.out" 2>&1 ||
    fail "stable Node $version was rejected"
  grep -q '@axe-core/playwright 4.13.0: ready' "$temporary/node-check.out" ||
    fail "stable Node $version did not load the UI tool"
done
for version in 22.9.0 23.0.0 24.14.0 25.0.0 24.15.0-rc.0; do
  if REAL_NODE="$real_node" TEST_NODE_VERSION="$version" TEST_COMPILED_GO_VERSION="$required_go" PATH="$temporary/bin:$PATH" \
    AI_DEV_TOOLS_BIN_DIR="$temporary/bin" "$root/hack/ai-dev-tools.sh" check >"$temporary/node-check.out" 2>&1; then
    fail "unsupported Node $version was accepted"
  fi
  grep -q '@axe-core/playwright 4.13.0: missing' "$temporary/node-check.out" ||
    fail "unsupported Node $version did not report the UI tool as unavailable"
done

# A matching tool release built with the wrong Go compiler is not ready.
if REAL_NODE="$real_node" TEST_NODE_VERSION=24.14.1 TEST_COMPILED_GO_VERSION=go1.24.0 \
  PATH="$temporary/bin:$PATH" AI_DEV_TOOLS_BIN_DIR="$temporary/bin" \
  "$root/hack/ai-dev-tools.sh" check >"$temporary/go-check.out" 2>&1; then
  fail 'check accepted govulncheck built with an older Go compiler'
fi
grep -q "compiled with go1.24.0; require $required_go" "$temporary/go-check.out" ||
  fail 'check did not identify the stale govulncheck compiler'
if REAL_NODE="$real_node" TEST_NODE_VERSION=24.14.1 TEST_GO_METADATA_FAILURE=1 \
  TEST_COMPILED_GO_VERSION="$required_go" PATH="$temporary/bin:$PATH" \
  AI_DEV_TOOLS_BIN_DIR="$temporary/bin" "$root/hack/ai-dev-tools.sh" check >"$temporary/go-check.out" 2>&1; then
  fail 'check accepted govulncheck without readable Go build metadata'
fi
grep -q 'cannot inspect Go build metadata' "$temporary/go-check.out" ||
  fail 'check did not explain the govulncheck metadata failure'
if REAL_NODE="$real_node" TEST_NODE_VERSION=24.14.1 TEST_COMPILED_GO_VERSION="$required_go" \
  TEST_GOVULNCHECK_VERSION=1.7.0-extra PATH="$temporary/bin:$PATH" \
  AI_DEV_TOOLS_BIN_DIR="$temporary/bin" "$root/hack/ai-dev-tools.sh" check >"$temporary/go-check.out" 2>&1; then
  fail 'check accepted a govulncheck release suffix as version 1.7.0'
fi
if REAL_NODE="$real_node" TEST_NODE_VERSION=24.14.1 TEST_COMPILED_GO_VERSION="$required_go" \
  TEST_HOST_GO_VERSION=go1.25.5 PATH="$temporary/bin:$PATH" \
  AI_DEV_TOOLS_BIN_DIR="$temporary/bin" "$root/hack/ai-dev-tools.sh" check >"$temporary/go-check.out" 2>&1; then
  fail 'check accepted an old host Go toolchain'
fi
grep -q "host Go go1.25.5; require $required_go" "$temporary/go-check.out" ||
  fail 'check did not identify the old host Go toolchain'

# Exercise the installer in an isolated project, replacing only external commands.
fixture="$temporary/install-root"
mkdir -p "$fixture/hack/installers" "$fixture/dist" "$fixture/ui/node_modules/@axe-core/playwright"
cp "$root/hack/installers/install-ai-dev-tools.sh" "$fixture/hack/installers/"
cp "$root/hack/ai-dev-tools.sh" "$root/hack/tool-versions.sh" "$fixture/hack/"
cp "$root/ui/package.json" "$fixture/ui/"
printf 'module fixture\n\ngo 1.27.1\n' >"$fixture/go.mod"
printf '{"version":"4.13.0"}\n' >"$fixture/ui/node_modules/@axe-core/playwright/package.json"
printf 'module.exports = {}\n' >"$fixture/ui/node_modules/@axe-core/playwright/index.js"
for tool in shellcheck grpcurl govulncheck; do
  cp "$temporary/bin/version-stub" "$fixture/dist/$tool"
done
cat >"$temporary/bin/dpkg-query" <<'STUB'
#!/usr/bin/env bash
printf 'install ok installed'
STUB
cat >"$temporary/bin/go" <<'STUB'
#!/usr/bin/env bash
case "${1:-} ${2:-}" in
  'env GOVERSION') echo 'go1.27.1' ;;
  'version -m')
    if [[ -f ${TEST_GO_METADATA_FAILURE_FILE:-/dev/null} ]]; then
      rm "$TEST_GO_METADATA_FAILURE_FILE"
      exit 1
    fi
    printf '%s: %s\n' "$3" "$(cat "$TEST_GO_COMPILER_FILE")"
    ;;
  'install golang.org/x/vuln/cmd/govulncheck@v1.7.0')
    [[ ${GOTOOLCHAIN:-} == local && ${GOBIN:-} == "$TEST_INSTALL_DIST" ]] || exit 3
    printf '#!/usr/bin/env bash\necho "Scanner: govulncheck@v1.7.0"\n' >"$GOBIN/govulncheck"
    chmod +x "$GOBIN/govulncheck"
    echo go1.27.1 >"$TEST_GO_COMPILER_FILE"
    echo installed >>"$TEST_GO_INSTALL_LOG"
    ;;
  *) echo "unexpected go invocation: $*" >&2; exit 2 ;;
esac
STUB
chmod +x "$temporary/bin/dpkg-query" "$temporary/bin/go"
echo go1.25.5 >"$temporary/compiled-go"
installer_env=(
  PATH="$temporary/bin:$PATH"
  REAL_NODE="$real_node"
  TEST_NODE_VERSION=24.14.1
  TEST_GO_COMPILER_FILE="$temporary/compiled-go"
  TEST_GO_METADATA_FAILURE_FILE="$temporary/metadata-failure-once"
  TEST_GO_INSTALL_LOG="$temporary/install.log"
  TEST_INSTALL_DIST="$fixture/dist"
)
env "${installer_env[@]}" "$fixture/hack/installers/install-ai-dev-tools.sh" >"$temporary/install.out" 2>&1 ||
  fail "installer did not rebuild the stale govulncheck: $(cat "$temporary/install.out")"
[[ $(wc -l <"$temporary/install.log") == 1 ]] || fail 'installer did not rebuild stale govulncheck once'
grep -q 'go1.27.1' "$temporary/compiled-go" || fail 'installer did not produce a binary built with the required Go'
env "${installer_env[@]}" "$fixture/hack/installers/install-ai-dev-tools.sh" >"$temporary/install.out" 2>&1 ||
  fail "installer rejected an already-correct govulncheck: $(cat "$temporary/install.out")"
[[ $(wc -l <"$temporary/install.log") == 1 ]] || fail 'installer reinstalled an already-correct govulncheck'
printf '#!/usr/bin/env bash\necho "Scanner: govulncheck@v1.7.0-extra"\n' >"$fixture/dist/govulncheck"
chmod +x "$fixture/dist/govulncheck"
env "${installer_env[@]}" "$fixture/hack/installers/install-ai-dev-tools.sh" >"$temporary/install.out" 2>&1 ||
  fail "installer did not replace a govulncheck release suffix: $(cat "$temporary/install.out")"
[[ $(wc -l <"$temporary/install.log") == 2 ]] || fail 'installer accepted a govulncheck release suffix as version 1.7.0'
touch "$temporary/metadata-failure-once"
env "${installer_env[@]}" "$fixture/hack/installers/install-ai-dev-tools.sh" >"$temporary/install.out" 2>&1 ||
  fail "installer did not rebuild govulncheck after a metadata read failure: $(cat "$temporary/install.out")"
[[ $(wc -l <"$temporary/install.log") == 3 ]] || fail 'installer skipped govulncheck after a metadata read failure'
rm "$fixture/dist/govulncheck"
env "${installer_env[@]}" "$fixture/hack/installers/install-ai-dev-tools.sh" >"$temporary/install.out" 2>&1 ||
  fail "installer did not restore missing govulncheck: $(cat "$temporary/install.out")"
[[ $(wc -l <"$temporary/install.log") == 4 ]] || fail 'installer did not rebuild missing govulncheck'

echo 'PASS: missing-tool guidance, scanner exit/artifact behavior, Node boundaries, and govulncheck compiler rebuild/readiness'
