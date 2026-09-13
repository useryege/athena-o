#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
temporary="$(mktemp -d)"
trap 'rm -rf "$temporary"' EXIT

fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }

check_abi() {
  local repo="$temporary/repo [x]* ?' quote" bin="$temporary/abi-bin" result status scenario
  mkdir -p "$repo/hack" "$repo/pkg/abi/Demo" "$bin"
  cp "$root/hack/generate-abi.sh" "$repo/hack/"
  printf 'contract Demo {}\n' > "$repo/pkg/abi/Demo/Demo.sol"
  cat > "$bin/solcjs" <<'STUB'
#!/usr/bin/env bash
set -eu
while (($#)); do
  if [[ $1 == -o ]]; then output=$2; shift; fi
  shift
done
printf '{}' > "$output/Demo.abi"
if [[ $ABI_SCENARIO == empty ]]; then : > "$output/Demo.bin"; else printf 00 > "$output/Demo.bin"; fi
if [[ $ABI_SCENARIO == multiple ]]; then printf 11 > "$output/Extra.bin"; fi
STUB
  cat > "$bin/abigen" <<'STUB'
#!/usr/bin/env bash
set -eu
while (($#)); do
  if [[ $1 == --out ]]; then printf '// fixture\n' > "$2"; exit 0; fi
  shift
done
exit 9
STUB
  chmod +x "$bin/solcjs" "$bin/abigen"
  for scenario in normal empty multiple multi-sol; do
    if [[ $scenario == multi-sol ]]; then cp "$repo/pkg/abi/Demo/Demo.sol" "$repo/pkg/abi/Demo/Other.sol"; fi
    status=0
    result="$(PATH="$bin:$PATH" ABI_SCENARIO="$scenario" bash "$repo/hack/generate-abi.sh" 2>&1)" || status=$?
    if [[ $scenario == normal ]]; then
      [[ $status == 0 && -f "$repo/pkg/abi/Demo/Demo.go" ]] || fail "ABI success: $result"
      [[ $result == *'compile pkg/abi/Demo/Demo.sol'* && $result == *'generate pkg/abi/Demo/Demo.go'* ]] || fail "ABI paths must be relative and literal: $result"
    else
      [[ $status == 1 ]] || fail "ABI $scenario exit $status"
      case $scenario in
        empty) [[ $result == *'no deployable contract bytecode generated from pkg/abi/Demo/Demo.sol'* ]] || fail "$result" ;;
        multiple) [[ $result == *'multiple deployable contract bytecode files generated from pkg/abi/Demo/Demo.sol'* ]] || fail "$result" ;;
        multi-sol) [[ $result == *'multiple Solidity files found in pkg/abi/Demo; keep exactly one *.sol file'* && $result == *'  - pkg/abi/Demo/Other.sol'* ]] || fail "$result" ;;
      esac
    fi
  done
  printf 'PASS ABI: four scenarios under a literal special-character repository path\n'
}

check_runtime() {
  local repo="$temporary/runtime repo [x]" action expected actual command
  mkdir -p "$repo/hack" "$repo/blocked-bin"
  cp "$root/hack/local-runtime.sh" "$repo/hack/"
  for command in docker ss lsof goreman kill; do
    # shellcheck disable=SC2016 # Expand $0 only in the isolated command substitute.
    printf '#!/usr/bin/env bash\nprintf "unexpected external operation: %%s\\n" "$0" >&2\nexit 99\n' > "$repo/blocked-bin/$command"
    chmod +x "$repo/blocked-bin/$command"
  done
  # An allowlist PATH has no fallback to host Docker/process tools.
  for command in bash dirname cat grep; do
    ln -s "$(command -v "$command")" "$repo/blocked-bin/$command"
  done
  local PATH="$repo/blocked-bin"
  cat > "$repo/hack/run-local-runtime.sh" <<'STUB'
#!/usr/bin/env bash
set -eu
printf '%s\n' "$PWD" "$@" "${INSTANCE-}" "${ENV_FILE-}" > "$EVENTS"
exit "${RESULT:-0}"
STUB
  for action in start stop reset; do
    case "$action" in start) expected=make-run-full-stack ;; stop) expected=make-stop-full-stack ;; reset) expected=make-reset-full-stack ;; esac
    EVENTS="$temporary/events" INSTANCE='literal-instance' ENV_FILE='literal $(touch sentinel).env' bash "$repo/hack/local-runtime.sh" "$action"
    [[ $(cat "$temporary/events") == "$repo"$'\n'"$expected"$'\n'"literal-instance"$'\n'"literal \$(touch sentinel).env" ]] || fail "runtime adapter $action lost literal parameters"
  done
  actual=0
  EVENTS="$temporary/events" RESULT=23 bash "$repo/hack/local-runtime.sh" start || actual=$?
  [[ $actual == 23 ]] || fail 'runtime adapter lost engine failure'
  actual=0
  bash "$repo/hack/local-runtime.sh" invalid > "$temporary/invalid.out" 2>&1 || actual=$?
  [[ $actual == 2 ]] || fail 'runtime adapter accepted invalid action'
  [[ ! -e "$repo/sentinel" ]] || fail 'runtime evaluated configuration as code'
  if grep -Eq '^[[:space:]]*[^#[:space:]].*:' "$root/Procfile"; then fail 'Procfile has a second executable ownership path'; fi
  printf 'PASS runtime: shared engine routing, literal environment, errors, declarative Procfile\n'
}

check_temporal() {
  local bin="$temporary/temporal-bin" scenario status expected_attempts expected_sleeps
  mkdir -p "$bin"
  cat > "$bin/psql" <<'STUB'
#!/usr/bin/env bash
set -eu
if [[ $* == *'SELECT 1' && $* != *pg_database* ]]; then
  count=0
  [[ ! -f $COUNTER ]] || read -r count < "$COUNTER"
  count=$((count + 1)); echo "$count" > "$COUNTER"
  ((count >= READY_AT))
else
  echo 1
fi
STUB
  cat > "$bin/sleep" <<'STUB'
#!/usr/bin/env bash
echo "$1" >> "$SLEEPS"
STUB
  cat > "$bin/docker" <<'STUB'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$DOCKER_EVENTS"
case $1 in
  ps) exit 0 ;;
  run) exit "$DOCKER_STATUS" ;;
  *) exit 97 ;;
esac
STUB
  chmod +x "$bin/psql" "$bin/sleep" "$bin/docker"
  for scenario in ready delayed timeout docker-failure; do
    local ready_at=1 docker_status=0 expected_status=0
    expected_attempts=1; expected_sleeps=0
    case $scenario in
      delayed) ready_at=3; expected_attempts=3; expected_sleeps=2 ;;
      timeout) ready_at=121; expected_attempts=120; expected_sleeps=120; expected_status=1 ;;
      docker-failure) docker_status=19; expected_status=19 ;;
    esac
    : > "$temporary/sleeps"; : > "$temporary/docker-events"; echo 0 > "$temporary/counter"
    status=0
    PATH="$bin:$PATH" COUNTER="$temporary/counter" SLEEPS="$temporary/sleeps" DOCKER_EVENTS="$temporary/docker-events" \
      READY_AT="$ready_at" DOCKER_STATUS="$docker_status" bash "$root/hack/start-temporal.sh" > "$temporary/temporal.out" 2>&1 || status=$?
    [[ $status == "$expected_status" ]] || fail "Temporal $scenario exit $status"
    [[ $(cat "$temporary/counter") == "$expected_attempts" ]] || fail "Temporal $scenario attempt count"
    [[ $(wc -l < "$temporary/sleeps") == "$expected_sleeps" ]] || fail "Temporal $scenario retry intervals"
    if grep -qvx '1' "$temporary/sleeps"; then fail "Temporal $scenario must retain one-second sleeps"; fi
    if [[ $scenario == timeout ]]; then
      [[ ! -s $temporary/docker-events ]] || fail 'Temporal started after timeout'
    else
      grep -q '^run ' "$temporary/docker-events" || fail 'Temporal did not invoke Docker'
    fi
  done
  printf 'PASS Temporal: immediate/delayed readiness, 120-attempt timeout, Docker failure\n'
}

case ${1:-all} in
  abi) check_abi ;;
  runtime) check_runtime ;;
  temporal) check_temporal ;;
  all) check_abi; check_runtime; check_temporal ;;
  *) fail 'expected all|abi|runtime|temporal' ;;
esac
