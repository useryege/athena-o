#!/usr/bin/env bash
# Existing profiles retain their original owner; no new profile can be launched.
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
temporary="$(mktemp -d)"
owned=''
foreign=''
cleanup() {
  [[ -z "$owned" ]] || kill "$owned" 2>/dev/null || true
  [[ -z "$foreign" ]] || kill "$foreign" 2>/dev/null || true
  wait 2>/dev/null || true
  rm -rf -- "$temporary"
}
trap cleanup EXIT
mkdir -p "$temporary/repo/hack" "$temporary/bin" "$temporary/repo/.run/solana-discovery"
cp "$root/hack/solana-local.sh" "$temporary/repo/hack/"
printf 'keep-data\n' > "$temporary/repo/.run/solana-discovery/data"
cat > "$temporary/bin/goreman" <<'FIXTURE'
#!/usr/bin/env bash
trap 'exit 0' TERM INT
while true; do sleep 0.1; done
FIXTURE
chmod +x "$temporary/bin/goreman"
# Construct an actual pre-existing legacy session, independently of the retired
# start code. Its exact process owner/session/checkout must pass the old stopper.
(
  cd "$temporary/repo"
  exec setsid "$temporary/bin/goreman" -f "$temporary/repo/Procfile.solana-discovery" start
) &
owned=$!
for ((i=0;i<50;i++)); do
  [[ "$(ps -o sid= -p "$owned" | tr -d ' ')" == "$owned" ]] && break
  sleep 0.02
done
read -r ticks < <(awk '{print $22}' "/proc/$owned/stat")
printf '%s %s\n' "$owned" "$ticks" > "$temporary/repo/.run/solana-discovery/process"
sleep 60 &
foreign=$!
if PATH="$temporary/bin:$PATH" bash "$temporary/repo/hack/solana-local.sh" start solana-discovery; then
  echo 'retired owner accepted new startup' >&2; exit 1
fi
bash "$temporary/repo/hack/solana-local.sh" stop solana-discovery
wait "$owned"
owned=''
kill -0 "$foreign"
[[ -s "$temporary/repo/.run/solana-discovery/process" ]]
[[ "$(cat "$temporary/repo/.run/solana-discovery/data")" == keep-data ]]
# Forged identity must not affect an unrelated live process.
printf '%s %s\n' "$foreign" "$(awk '{print $22}' "/proc/$foreign/stat")" > "$temporary/repo/.run/solana-discovery/process"
if bash "$temporary/repo/hack/solana-local.sh" stop solana-discovery; then
  echo 'forged owner accepted' >&2; exit 1
fi
kill -0 "$foreign"
echo 'PASS: retired startup rejected, legacy owned stop, data/state retained, foreign PID protected'
