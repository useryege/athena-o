#!/usr/bin/env python3
"""Exercise merged launch entrypoints without starting services or infrastructure."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent.parent


class SolanaIntegrationTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="athena-solana-integration-")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name) / "repo [literal]"
        (self.root / "hack").mkdir(parents=True)
        self.bin = self.root / "bin"
        self.bin.mkdir()
        # Only command recorders and necessary shell tools are visible. A failure
        # cannot fall through to host Go, Docker or a real process controller.
        for name in ("bash", "sh", "dirname", "python3"):
            (self.bin / name).symlink_to(shutil.which(name))
        self.events = self.root / "events.jsonl"
        self.env = {
            "PATH": str(self.bin),
            "EVENTS": str(self.events),
            "ATHENA_SOLANA_PREVIEW_POSTGRES_DSN": "postgres://fixture/isolated_preview",
            "ATHENA_SERVER_POSTGRES_DSN": "postgres://fixture/legacy_must_not_be_used",
            "ATHENA_SOLANA_PREVIEW_API_PORT": "38080",
            "ATHENA_SOLANA_PREVIEW_DISCOVERY_PORT": "38112",
        }
        recorder = '''#!/usr/bin/env bash
exec python3 - "$0" "$@" <<'PY'
import json, os, sys
from pathlib import Path
with open(os.environ["EVENTS"], "a") as output:
    output.write(json.dumps({"program": Path(sys.argv[1]).name, "argv": sys.argv[2:], "cwd": os.getcwd(), "dsn": os.environ.get("ATHENA_ACCOUNT_STATE_POSTGRES_DSN"), "solana": os.environ.get("ATHENA_SOLANA_DISCOVERY_SERVER_ADDRESS")}) + "\\n")
sys.exit(int(os.environ.get("RESULT", "0")) if not os.environ.get("FAIL_ACTION") or os.environ["FAIL_ACTION"] in sys.argv else 0)
PY
'''
        for name in ("hack/run-local-runtime.sh", "hack/solana-local.sh", "bin/go"):
            path = self.root / name
            path.write_text(recorder)
            path.chmod(0o700)
        shutil.copy(ROOT / "hack/local-runtime.sh", self.root / "hack/local-runtime.sh")

    def records(self):
        if not self.events.exists():
            return []
        return [json.loads(line) for line in self.events.read_text().splitlines()]

    def run_adapter(self, profile, action):
        return subprocess.run(
            [str(self.bin / "bash"), "hack/local-runtime.sh", action],
            cwd=self.root, env={**self.env, "ATHENA_RUN_PROFILE": profile},
            capture_output=True, text=True, timeout=5,
        )

    def test_explicit_profiles_route_only_to_their_owner(self):
        for profile in ("solana-discovery", "solana-preview"):
            for action in ("start", "stop"):
                with self.subTest(profile=profile, action=action):
                    result = self.run_adapter(profile, action)
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertEqual(self.records()[-1]["program"], "solana-local.sh")
                    self.assertEqual(self.records()[-1]["argv"], [action, profile])

    def test_default_routes_to_full_stack(self):
        for action in ("start", "stop", "reset"):
            result = self.run_adapter("", action)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(self.records()[-1]["program"], "run-local-runtime.sh")
            self.assertEqual(self.records()[-1]["argv"], ["make-" + {"start": "run", "stop": "stop", "reset": "reset"}[action] + "-full-stack"])

    def test_profiles_reject_reset_and_unknown_values_before_dispatch(self):
        for profile, action in (("solana-preview", "reset"), ("solana-discovery", "reset"), ("unknown", "start")):
            with self.subTest(profile=profile, action=action):
                result = self.run_adapter(profile, action)
                self.assertEqual(result.returncode, 2, result.stderr)
                self.assertEqual(self.records(), [])

    def preview_api_command(self):
        line = next(line for line in (ROOT / "Procfile.solana-preview").read_text().splitlines() if line.startswith("api-server:"))
        return line.split(":", 1)[1].strip()

    def run_preview_api(self, **extra):
        return subprocess.run([str(self.bin / "sh"), "-c", self.preview_api_command()],
                              cwd=self.root, env={**self.env, **extra},
                              capture_output=True, text=True, timeout=5)

    def test_preview_prepares_shared_schema_before_direct_api(self):
        result = self.run_preview_api()
        self.assertEqual(result.returncode, 0, result.stderr)
        records = self.records()
        self.assertEqual([r["argv"][:3] for r in records], [
            ["run", "./cmd/athena-account-state-migrate", "up"],
            ["run", "./cmd/athena-account-state-migrate", "verify"],
            ["run", "./cmd/athena-server", "--redisdb"],
        ])
        self.assertEqual([r["dsn"] for r in records], ["postgres://fixture/isolated_preview"] * 3)
        self.assertEqual(records[-1]["solana"], "127.0.0.1:38112")
        self.assertIn("38080", records[-1]["argv"])

    def test_preview_schema_failure_does_not_start_api(self):
        for action, count in (("up", 1), ("verify", 2)):
            with self.subTest(action=action):
                self.events.unlink(missing_ok=True)
                result = self.run_preview_api(FAIL_ACTION=action, RESULT="19")
                self.assertEqual(result.returncode, 19, result.stderr)
                self.assertEqual(len(self.records()), count)
                self.assertTrue(all(r["argv"][1] == "./cmd/athena-account-state-migrate" for r in self.records()))


if __name__ == "__main__":
    unittest.main()
