#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
temporary="$(mktemp -d)"
trap 'rm -rf -- "$temporary"' EXIT
mkdir -p "$temporary/repo/hack" "$temporary/bin"
cp "$root/hack/solana-local.sh" "$temporary/repo/hack/"
touch "$temporary/repo/Procfile.solana-discovery"
cat > "$temporary/bin/goreman" <<'STUB'
#!/usr/bin/env bash
trap 'exit 0' TERM INT
while true; do sleep 0.1; done
STUB
cat > "$temporary/bin/docker" <<'STUB'
#!/usr/bin/env bash
echo 'Unexpected Docker resource operation' >&2
exit 99
STUB
chmod +x "$temporary/bin/"*
export PATH="$temporary/bin:$PATH"
bash "$temporary/repo/hack/solana-local.sh" start solana-discovery > "$temporary/log" 2>&1 &
controller=$!
for ((i=0;i<50;i++));do [[ -s "$temporary/repo/.run/solana-discovery/process" ]] && break;sleep 0.1;done
[[ -s "$temporary/repo/.run/solana-discovery/process" ]]
if bash "$temporary/repo/hack/solana-local.sh" start solana-discovery >> "$temporary/log" 2>&1;then echo 'duplicate start accepted';exit 1;fi
sleep 60 &
unrelated=$!
bash "$temporary/repo/hack/solana-local.sh" stop solana-discovery
wait "$controller"
kill -0 "$unrelated"
kill "$unrelated"
wait "$unrelated" 2>/dev/null || true
[[ ! -d "$temporary/repo/.run/solana-discovery" ]]
# A forged state referring to a live unrelated process must never signal it.
mkdir "$temporary/repo/.run/solana-discovery"
sleep 60 &
unrelated=$!
printf '%s %s\n' "$unrelated" "$(awk '{print $22}' "/proc/$unrelated/stat")" > "$temporary/repo/.run/solana-discovery/process"
if bash "$temporary/repo/hack/solana-local.sh" stop solana-discovery;then echo 'forged owner accepted';kill "$unrelated";exit 1;fi
kill -0 "$unrelated"
kill "$unrelated"
wait "$unrelated" 2>/dev/null || true
echo 'PASS: isolated start, duplicate prevention, bounded stop and unrelated process protection'
# Use real Goreman: its children have separate process groups within one session.
PATH="${PATH#*:}" python3 - "$root" <<'PY'
import os, pathlib, shutil, signal, subprocess, sys, tempfile, time
source = pathlib.Path(sys.argv[1]) / 'hack/solana-local.sh'
def live(pid):
    try:
        return pathlib.Path(f'/proc/{pid}/stat').read_text().rsplit(') ', 1)[1].split()[0] != 'Z'
    except FileNotFoundError:
        return False
for scenario in ['supervisor_crash', 'orphan_session', 'stubborn_child']:
    with tempfile.TemporaryDirectory(prefix='solana-runtime-test-') as temp:
        root = pathlib.Path(temp)
        (root/'hack').mkdir()
        shutil.copy(source, root/'hack/solana-local.sh')
        (root/'child.py').write_text('import os,time,pathlib,signal\nsignal.signal(signal.SIGTERM, signal.SIG_IGN)\nsignal.signal(signal.SIGINT, signal.SIG_IGN)\npathlib.Path("child.pid").write_text(str(os.getpid()))\ntime.sleep(120)\n')
        (root/'Procfile.solana-discovery').write_text('child: python3 child.py\n')
        log = open(root/'log', 'w')
        controller = subprocess.Popen(['bash',str(root/'hack/solana-local.sh'),'start','solana-discovery'],stdout=log,stderr=log)
        child = supervisor = None
        try:
            for _ in range(100):
                if (root/'child.pid').exists(): break
                time.sleep(.05)
            child = int((root/'child.pid').read_text())
            supervisor = int((root/'.run/solana-discovery/process').read_text().split()[0])
            assert os.getpgid(child) != supervisor, 'fixture must have distinct process groups'
            if scenario == 'orphan_session':
                controller.kill(); controller.wait(timeout=2)
            if scenario != 'stubborn_child':
                os.kill(supervisor, signal.SIGKILL)
            if scenario == 'supervisor_crash':
                controller.wait(timeout=18)
            else:
                stopped = subprocess.run(['bash',str(root/'hack/solana-local.sh'),'stop','solana-discovery'],capture_output=True,text=True,timeout=18)
                assert stopped.returncode == 0, stopped.stderr
                if scenario == 'stubborn_child': controller.wait(timeout=3)
            assert not live(child), f'{scenario}: child remains running after bounded cleanup'
            assert not (root/'.run/solana-discovery').exists(), f'{scenario}: state not cleaned'
            print('PASS:', scenario)
        finally:
            if supervisor and live(supervisor): os.kill(supervisor,signal.SIGKILL)
            if child and live(child): os.killpg(os.getpgid(child),signal.SIGKILL)
            if controller.poll() is None: controller.kill(); controller.wait()
            log.close()
PY
