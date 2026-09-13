#!/usr/bin/env python3
"""Exercise terminal-generated SIGINT through the public Make entrypoint."""
import json
import os
import pathlib
import pty
import select
import shlex
import socket
import subprocess
import tempfile
import time

root = pathlib.Path(__file__).resolve().parents[1]
name = 'task9-pty-' + str(os.getpid())
folder = pathlib.Path(tempfile.mkdtemp(prefix='athena-task9-pty-'))
with socket.socket() as listener:
    listener.bind(('127.0.0.1', 0))
    port = listener.getsockname()[1]
envfile = folder / 'ui.env'
envfile.write_text(f'ATHENA_UI_PORT={port}\nATHENA_API_URL=http://127.0.0.1:1\n')
statefile = root / '.run/instances' / name / 'state.json'
returned = folder / 'shell-returned.json'
log = bytearray()
pid, fd = pty.fork()
if pid == 0:
    os.chdir(root)
    os.environ.pop('ATHENA_UI_PORT', None)
    os.execv('/bin/bash', ['bash', '--noprofile', '--norc', '-i'])


def collect(seconds):
    end = time.monotonic() + seconds
    while time.monotonic() < end:
        readable, _, _ = select.select([fd], [], [], min(.1, max(0, end-time.monotonic())))
        if readable:
            try:
                log.extend(os.read(fd, 65536))
            except OSError:
                return


try:
    collect(.2)
    argv = ['make', '--no-print-directory', 'run-service', 'SERVICE=ui', 'INSTANCE='+name, 'ENV_FILE='+str(envfile)]
    os.write(fd, (shlex.join(argv)+'\n').encode())
    deadline = time.monotonic()+120
    while time.monotonic() < deadline:
        collect(.1)
        if statefile.exists():
            state = json.loads(statefile.read_text())
            if state['Phase'] == 'running':
                break
    else:
        raise AssertionError('UI did not become ready: '+log.decode(errors='replace')[-2000:])
    supervisor = state['Supervisor']['PID']
    node = state['Processes']['ui']['PID']
    assert os.getpgid(supervisor) == supervisor
    assert supervisor != os.tcgetpgrp(fd), 'supervisor must keep its independently verified PGID'
    # This queued command runs only after Make returns. Capture that exact point,
    # rather than relying on terminal echo or a later sleep hiding early return.
    probe = ('import json,pathlib; '
             f's=json.loads(pathlib.Path({str(statefile)!r}).read_text()); '
             f'pathlib.Path({str(returned)!r}).write_text(json.dumps(s))')
    os.write(fd, b'\x03')
    os.write(fd, ('python3 -c '+shlex.quote(probe)+'\n').encode())
    deadline = time.monotonic()+90
    while not returned.exists() and time.monotonic() < deadline:
        collect(.1)
    assert returned.exists(), 'Make did not return after the Stop protocol'
    after = json.loads(returned.read_text())
    assert after['Phase'] == 'stopped', 'Make returned before Stop completed: '+after['Phase']
    for label, process in [('supervisor', supervisor), ('ui', node)]:
        assert not pathlib.Path(f'/proc/{process}').exists(), label+' survived terminal Ctrl+C'
    assert not after.get('Failures'), after.get('Failures')
    print(json.dumps({'instance': name, 'namespace': state['Key']['Namespace'], 'supervisor': supervisor,
                      'ui': node, 'port': port, 'phase_when_make_returned': after['Phase'], 'log': str(folder/'pty.log')}), flush=True)
finally:
    for action in ['stop-instance', 'reset-instance']:
        cleanup = subprocess.run(['make', '--no-print-directory', action, 'INSTANCE='+name],
                                 cwd=root, capture_output=True, text=True, timeout=120)
        if cleanup.returncode:
            raise AssertionError(action+' failed: '+cleanup.stderr)
    (folder/'pty.log').write_bytes(log)
    try:
        os.write(fd, b'exit\n')
        collect(.2)
    except OSError:
        pass
    os.close(fd)
    os.waitpid(pid, 0)
