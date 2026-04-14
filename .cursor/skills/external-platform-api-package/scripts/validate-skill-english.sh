#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="${1:-$(cd "${SCRIPT_DIR}/.." && pwd)}"

if [[ ! -d "${SKILL_DIR}" ]]; then
  echo "Skill directory does not exist: ${SKILL_DIR}" >&2
  exit 1
fi

python3 - "${SKILL_DIR}" <<'PY'
from __future__ import annotations

import pathlib
import re
import sys


skill_dir = pathlib.Path(sys.argv[1]).resolve()

candidate_paths = [skill_dir / "SKILL.md"]
for name in ("references", "scripts", "assets"):
    base = skill_dir / name
    if base.exists():
        candidate_paths.extend(p for p in base.rglob("*") if p.is_file())


def should_check(path: pathlib.Path) -> bool:
    if path.name == ".gitkeep":
        return False
    if path.suffix.lower() in {
        ".png",
        ".jpg",
        ".jpeg",
        ".gif",
        ".webp",
        ".pdf",
        ".zip",
        ".tar",
        ".gz",
        ".tgz",
        ".bz2",
        ".xz",
        ".7z",
        ".ico",
    }:
        return False
    return True


non_english_pattern = re.compile(
    r"[\u3000-\u303F\u3040-\u30FF\u3400-\u4DBF\u4E00-\u9FFF\uAC00-\uD7AF\uFF00-\uFFEF]"
)

violations: list[tuple[pathlib.Path, int, str]] = []

for path in sorted(set(candidate_paths)):
    if not should_check(path):
        continue
    try:
        content = path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        continue

    for line_no, line in enumerate(content.splitlines(), start=1):
        if non_english_pattern.search(line):
            violations.append((path, line_no, line.strip()))

if violations:
    print("English-only validation failed.", file=sys.stderr)
    print("The following lines contain CJK or full-width characters:", file=sys.stderr)
    for path, line_no, line in violations:
        rel_path = path.relative_to(skill_dir)
        print(f"- {rel_path}:{line_no}: {line}", file=sys.stderr)
    sys.exit(1)

print(f"English-only validation passed for {skill_dir}")
PY
