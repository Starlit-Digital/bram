#!/usr/bin/env python3
"""Copy the standalone 0BSD integration snapshot into an explicitly chosen repo."""
import argparse
import hashlib
import json
from pathlib import Path

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--repo', type=Path, required=True)
args = parser.parse_args()
source = Path(__file__).resolve().parents[1] / 'internal/toolbridge'
repo = args.repo.resolve()
if not (repo / 'go.mod').is_file():
    parser.error('--repo must be a Go repository')
dest = repo / 'internal/toolbridge'
if source == dest:
    parser.error('the authoritative snapshot cannot overwrite itself')
existing = dest / 'UPSTREAM.json'
if existing.exists():
    old = json.loads(existing.read_text())
    for name, digest in old.get('files', {}).items():
        candidate = dest / name
        if Path(name).name != name or not candidate.is_file() or hashlib.sha256(candidate.read_bytes()).hexdigest() != digest:
            parser.error('destination snapshot has edits; preserve or reconcile them before syncing')
elif dest.exists() and any(dest.iterdir()):
    parser.error('nonempty destination has no snapshot receipt; preserve it before syncing')
dest.mkdir(parents=True, exist_ok=True)
files = {}
for name in ['bridge.go', 'bridge_test.go', 'LICENSE']:
    data = (source / name).read_bytes()
    (dest / name).write_bytes(data)
    files[name] = hashlib.sha256(data).hexdigest()
(dest / 'UPSTREAM.json').write_text(json.dumps({
    'schemaVersion': 1,
    'source': 'Starlit-Digital/bram/internal/toolbridge',
    'version': '1.0.0',
    'license': '0BSD',
    'files': files,
}, indent=2) + '\n')
print('Updated snapshot:', dest)
