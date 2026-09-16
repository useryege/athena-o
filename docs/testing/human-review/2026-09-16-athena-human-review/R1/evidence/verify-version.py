#!/usr/bin/env python3
"""Read-only check of this handoff's source version; never changes the checkout."""

import hashlib
import json
from pathlib import Path
import subprocess
import sys


def main():
    evidence = Path(__file__).resolve().parent
    root = evidence.parents[5]
    manifest = json.loads((evidence / "source-manifest.json").read_text())
    actual_head = subprocess.check_output(
        ["git", "rev-parse", "HEAD"], cwd=root, text=True
    ).strip()
    differences = []
    if actual_head != manifest["base_head"]:
        differences.append(f"HEAD: expected {manifest['base_head']}, found {actual_head}")
    for relative, expected in manifest["source_sha256"].items():
        path = root / relative
        if not path.is_file():
            differences.append(f"Missing: {relative}")
        elif hashlib.sha256(path.read_bytes()).hexdigest() != expected:
            differences.append(f"Changed: {relative}")
    if differences:
        print("Review version differs; preserve current work and confirm the target:")
        print("\n".join(differences))
        return 1
    print(
        f"PASS: R1 source version matches ({len(manifest['source_sha256'])} files; "
        f"base {actual_head[:12]})."
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
