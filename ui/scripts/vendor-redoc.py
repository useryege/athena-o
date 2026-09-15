#!/usr/bin/env python3
"""Reproduce the pinned local documentation client; no application dependency tree."""
import base64
import hashlib
import io
from pathlib import Path
import tarfile
import urllib.request

URL = 'https://registry.yarnpkg.com/redoc/-/redoc-2.4.0.tgz'
INTEGRITY = 'rFlfzFVWS9XJ6aYAs/bHnLhHP5FQEhwAHDBVgwb9L2FqDQ8Hu8rQ1G84iwaWXxZfPP9UWn7JdWkxI6MXr2ZDjw=='
FILES = {
    'package/bundles/redoc.standalone.js': ('redoc.standalone.js', '2c0d3cb1a32e2417c5b7200da812fdf48e48d2a6ed9c49a39926c7fd3181d14c'),
    'package/LICENSE': ('redoc-LICENSE.txt', 'd3026d549cf68ab7355bcfa85877bf8f845b3334a7efbfdc63936432fb34ff0e'),
}

def main():
    data = urllib.request.urlopen(URL, timeout=60).read()
    if base64.b64encode(hashlib.sha512(data).digest()).decode() != INTEGRITY:
        raise ValueError('ReDoc package integrity mismatch')
    destination = Path(__file__).resolve().parents[1] / 'src/assets/assets/scripts'
    destination.mkdir(parents=True, exist_ok=True)
    with tarfile.open(fileobj=io.BytesIO(data), mode='r:gz') as archive:
        for member, (name, digest) in FILES.items():
            content = archive.extractfile(member).read()
            if hashlib.sha256(content).hexdigest() != digest:
                raise ValueError(f'ReDoc file integrity mismatch: {member}')
            (destination / name).write_bytes(content)
            print(f'{name}: {digest}')

if __name__ == '__main__':
    main()
