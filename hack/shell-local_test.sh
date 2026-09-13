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
  local library="$temporary/local-runtime-library.sh" child="$temporary/cleanup-child.sh" scenario expected actual
  # Load the real function definitions without invoking the production action dispatcher.
  grep -q '^action=' "$root/hack/local-runtime.sh" || fail 'runtime dispatcher marker missing'
  awk '/^action=/{exit} {print}' "$root/hack/local-runtime.sh" > "$library"
  cat > "$child" <<'CHILD'
#!/usr/bin/env bash
set -euo pipefail
source "$LIBRARY"
stop_goreman() { echo stop >> "$EVENTS"; return "$STOP_STATUS"; }
cleanup_athena_ports() { echo ports >> "$EVENTS"; return "$PORT_STATUS"; }
cleanup_local_containers() { echo containers >> "$EVENTS"; return "$CONTAINER_STATUS"; }
remove_runtime_state() { echo state >> "$EVENTS"; }
trap cleanup_foreground_run EXIT
if [[ $SIGNAL != none ]]; then
  trap 'exit 130' INT
  trap 'exit 143' TERM
  kill -s "$SIGNAL" "$BASHPID"
fi
exit "$ORIGINAL_STATUS"
CHILD
  for scenario in success original stop ports containers both term int skip; do
    local stop=0 ports=0 containers=0 original=0 signal=none port_cleanup=true
    expected=0
    case $scenario in
      original) original=23; stop=9; ports=7; containers=8; expected=23 ;;
      stop) stop=9; expected=9 ;;
      ports) ports=7; expected=1 ;;
      containers) containers=8; expected=1 ;;
      both) ports=7; containers=8; expected=1 ;;
      term) signal=TERM; ports=7; expected=143 ;;
      int) signal=INT; expected=130 ;;
      skip) port_cleanup=false ;;
    esac
    : > "$temporary/events"
    actual=0
    LIBRARY="$library" EVENTS="$temporary/events" STOP_STATUS="$stop" PORT_STATUS="$ports" CONTAINER_STATUS="$containers" \
      ORIGINAL_STATUS="$original" SIGNAL="$signal" ATHENA_RUN_PORT_CLEANUP="$port_cleanup" bash "$child" || actual=$?
    [[ $actual == "$expected" ]] || fail "runtime $scenario exit $actual, expected $expected"
    if [[ $scenario == skip ]]; then
      [[ $(cat "$temporary/events") == $'stop\ncontainers\nstate' ]] || fail 'skip cleanup ordering'
    else
      [[ $(cat "$temporary/events") == $'stop\nports\ncontainers\nstate' ]] || fail "$scenario cleanup ordering"
    fi
  done
  # Exercise the actual ownership checks with a Docker function substitute.
  cat > "$temporary/ownership-child.sh" <<'CHILD'
#!/usr/bin/env bash
set -euo pipefail
source "$LIBRARY"
docker() {
  if [[ $1 == info ]]; then return 0; fi
  if [[ $2 == rm ]]; then printf '%s\n' "${@: -1}" >> "$EVENTS"; return 0; fi
  if [[ $3 != --format ]]; then return 0; fi
  case $4 in
    *io.athena.local-runtime*) [[ ${@: -1} == athena-postgres ]] && echo another-project || echo athena ;;
    *io.athena.component*)
      case ${@: -1} in athena-postgres) echo postgres ;; athena-redis) echo wrong-component ;; athena-minio) echo minio ;; esac ;;
    *) return 98 ;;
  esac
}
cleanup_local_containers
CHILD
  : > "$temporary/events"
  actual=0
  LIBRARY="$library" EVENTS="$temporary/events" bash "$temporary/ownership-child.sh" > "$temporary/ownership.out" 2>&1 || actual=$?
  [[ $actual == 1 && $(cat "$temporary/events") == athena-minio ]] || fail 'ownership boundary or continuing cleanup changed'
  printf 'PASS runtime: nine exit/signal scenarios and owned-container cleanup boundary\n'
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
