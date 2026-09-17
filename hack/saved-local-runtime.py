#!/usr/bin/env python3
"""Locate a verified per-instance immutable runner without compiling the checkout.

File validity deliberately does not depend on a process boot ID. A reboot removes
process ownership, but recorded container ownership and the executable survive.
"""
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import sys


def locate(arguments):
    if not arguments:
        return None
    action = arguments[0]
    saved_actions = {"status", "stop", "make-status", "make-stop", "make-stop-full-stack"}
    start_actions = {"run", "run-full-stack", "make-run-full-stack", "make-run-service", "make-run-services"}
    if action not in saved_actions | start_actions:
        return None
    checkout = Path.cwd().resolve()
    instance = os.environ.get("INSTANCE", "")
    if action.endswith("full-stack") and not instance:
        instance = "full-stack"
    if action == "make-run-service" and not instance:
        instance = os.environ.get("SERVICE", "")
    for index, argument in enumerate(arguments[1:], 1):
        for flag in ("--checkout", "--instance"):
            value = None
            if argument == flag and index + 1 < len(arguments):
                value = arguments[index + 1]
            elif argument.startswith(flag + "="):
                value = argument[len(flag) + 1:]
            if value is not None:
                if flag == "--checkout":
                    checkout = Path(value).resolve(strict=True)
                else:
                    instance = value
    if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9_-]{0,62}", instance):
        return None
    directory = checkout / ".run" / "instances" / instance
    for path in (checkout / ".run", directory.parent, directory):
        try:
            mode = path.lstat().st_mode
        except FileNotFoundError:
            return None
        if not stat.S_ISDIR(mode):
            raise ValueError("unsafe instance directory")
    state_path = directory / "state.json"
    try:
        fd = os.open(state_path, os.O_RDONLY | os.O_NOFOLLOW | os.O_CLOEXEC)
    except FileNotFoundError:
        return None
    with os.fdopen(fd) as stream:
        if not stat.S_ISREG(os.fstat(stream.fileno()).st_mode):
            raise ValueError("unsafe state file")
        state = json.load(stream)
    namespace = hashlib.sha256((str(checkout) + "\0" + instance).encode()).hexdigest()[:32]
    if state.get("Version") != 1 or state.get("Key") != {"Checkout": str(checkout), "Name": instance, "Namespace": namespace}:
        raise ValueError("state checkout or instance identity mismatch")
    supervisor = state.get("Supervisor", {})
    executable = supervisor.get("Exe", "")
    if not executable:
        return None
    path = Path(executable)
    if path.parent != directory / "executables" or not re.fullmatch(r"[0-9a-f]{64}", path.name):
        raise ValueError("saved runner is outside immutable instance directory")
    if not stat.S_ISDIR(path.parent.lstat().st_mode):
        raise ValueError("unsafe executables directory")
    try:
        fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_CLOEXEC)
    except FileNotFoundError:
        return None
    with os.fdopen(fd, "rb") as stream:
        metadata = os.fstat(stream.fileno())
        if not stat.S_ISREG(metadata.st_mode) or metadata.st_mode & 0o111 == 0:
            raise ValueError("saved runner is not executable")
        digest = hashlib.file_digest(stream, "sha256").hexdigest()
        if digest != path.name:
            raise ValueError("saved runner digest mismatch")
    if action in start_actions:
        # Only reuse the saved runner to report an already-owned active run.
        # A stopped instance always builds the requested new checkout version.
        pid = supervisor.get("PID", 0)
        try:
            boot = Path("/proc/sys/kernel/random/boot_id").read_text().strip()
            proc = Path("/proc") / str(pid)
            fields = (proc / "stat").read_text().rsplit(")", 1)[1].split()
            marker = ("ATHENA_LOCAL_RUNTIME_RUN_ID=" + supervisor.get("RunID", "")).encode()
            valid = (boot == supervisor.get("BootID") and fields[0] not in ("Z", "X")
                     and int(fields[2]) == supervisor.get("PGID")
                     and int(fields[19]) == supervisor.get("StartTicks")
                     and os.readlink(proc / "exe") == executable
                     and marker in (proc / "environ").read_bytes().split(b"\0"))
        except (OSError, ValueError, IndexError):
            valid = False
        if not valid:
            return None
    return executable


if __name__ == "__main__":
    try:
        result = locate(sys.argv[1:])
        if result:
            print(result)
    except (OSError, ValueError, KeyError, TypeError) as error:
        print(f"saved runtime validation failed: {error}", file=sys.stderr)
        sys.exit(2)
