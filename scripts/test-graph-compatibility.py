#!/usr/bin/env python3
"""Exercise installed CLI graph compatibility using owned temporary local fixtures."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--output', type=Path, help='new JSON result file')
a = parser.parse_args()
tools = ['vigil', 'loom', 'clyde', 'bram', 'nora', 'goshi']
binaries = {name: shutil.which(name) for name in tools}
if any(path is None for path in binaries.values()):
    raise SystemExit('Missing tool: ' + ', '.join(n for n, p in binaries.items() if p is None))
checks = []

def invoke(name, args, cwd, expected=0):
    result = subprocess.run([binaries[name], *args], cwd=cwd, capture_output=True,
                            text=True, timeout=120)
    if result.returncode != expected:
        raise AssertionError(f'{name} {args}: {result.returncode}\n{result.stdout}\n{result.stderr}')
    return result.stdout

with tempfile.TemporaryDirectory(prefix='starlit-graph-compat-') as temp:
    base = Path(temp).resolve()
    repo = base / 'repo'
    repo.mkdir()
    (repo / 'ui').mkdir()
    (repo / 'ui/Main.qml').write_text('import QtQuick\nItem { Badge {} }\n')
    (repo / 'ui/Badge.qml').write_text('import QtQuick\nRectangle { width: 40; height: 20 }\n')
    (repo / 'owner.md').write_text('Owned fixture: Badge is a component dependency.\n')
    (repo / 'acceptance.txt').write_text('accepted\n')
    for args in [['init', '-q'], ['config', 'user.name', 'Graph Fixture'],
                 ['config', 'user.email', 'graph@example.invalid'], ['add', '.'],
                 ['commit', '-qm', 'owned fixture']]:
        subprocess.run(['git', '-C', str(repo), *args], check=True, capture_output=True)
    for name in tools:
        cap = json.loads(invoke(name, ['tools', 'graph', 'capabilities'], repo))
        assert cap['protocol_version'] == '1.1.0' and cap['tool'] == name
    checks.append('six CLI capability contracts')
    loom_report = base / 'loom.json'
    loom_report.write_text(invoke('loom', ['graph:components', 'ui', '--format', 'json'], repo))
    assert json.loads(loom_report.read_text())['source_files']
    bundle = base / 'bundle'
    invoke('clyde', ['bundle', '.', '--include', 'ui/*.qml', '--out', str(bundle)], repo)
    invoke('clyde', ['bundle', 'verify', str(bundle)], repo)
    checks.append('native Loom source hashes and native Clyde bundle verification')
    scope = {'root': 'ui', 'suffixes': ['.qml']}
    workflow = {
        'schema': 'starlit.workflow.v1', 'name': 'compatibility', 'budget_seconds': 120,
        'context': [{'path': 'owner.md', 'stages': ['investigate']}],
        'nodes': [
            {'id': 'components', 'kind': 'analysis',
             'argv': ['loom', 'graph:components', 'ui', '--format', 'json'],
             'scopes': [scope], 'timeout_seconds': 30, 'max_attempts': 1, 'replay_safe': True},
            {'id': 'source', 'kind': 'analysis',
             'argv': ['clyde', 'scan-report', '.', '--include', 'ui/*.qml', '--json', '--top', '2'],
             'scopes': [scope], 'timeout_seconds': 30, 'max_attempts': 1, 'replay_safe': True},
            {'id': 'contract', 'kind': 'check', 'argv': [sys.executable, '-c',
             "from pathlib import Path; assert Path('acceptance.txt').read_text().strip() == 'accepted'; print('{\\\"value\\\":9223372036854775807}')"],
             'inputs': ['acceptance.txt'], 'needs': ['components'],
             'timeout_seconds': 30, 'max_attempts': 1, 'replay_safe': True}
        ]}
    workflow_path = repo / 'workflow.json'
    workflow_path.write_text(json.dumps(workflow))
    plan_args = ['tools', 'graph', 'plan', '--repo', str(repo), '--workflow', str(workflow_path),
                 '--loom', str(loom_report), '--clyde-manifest', str(bundle / 'manifest.json')]
    plan = json.loads(invoke('vigil', plan_args, repo))
    for name in tools:
        other = json.loads(invoke(name, plan_args, repo))
        assert other['digest'] == plan['digest'], (name, other['digest'], plan['digest'])
    checks.append('identical connected plan digest across all six CLIs')
    plan_path = base / 'plan.json'
    plan_path.write_text(json.dumps(plan))
    packet = json.loads(invoke('clyde', ['tools', 'graph', 'context', '--plan', str(plan_path), '--stage', 'investigate'], repo))
    assert packet['schema'] == 'starlit.context.v1' and 'Badge' in packet['items'][0]['text']
    run = base / 'run'
    execute = ['--plan', str(plan_path), '--run-dir', str(run), '--approve-digest', plan['digest']]
    first = json.loads(invoke('bram', ['tools', 'graph', 'run', *execute], repo))
    assert first['status'] == 'completed' and first['qualification'] == 'declared_checks_passed'
    resumed = json.loads(invoke('loom', ['tools', 'graph', 'resume', *execute], repo))
    assert all(n['reused'] for n in resumed['nodes'].values())
    verified = json.loads(invoke('clyde', ['tools', 'graph', 'verify', '--run-dir', str(run)], repo))
    assert verified['fresh'] and verified['consistent']
    invoke('nora', ['tools', 'graph', 'explain', '--run-dir', str(run)], repo)
    checks.append('Bram execution, Loom resume, Clyde verify, Nora explanation')
    denied = base / 'goshi-denied'
    invoke('goshi', ['tools', 'graph', 'run', '--plan', str(plan_path), '--run-dir', str(denied), '--approve-digest', plan['digest']], repo, expected=1)
    assert not denied.exists()
    checks.append('Goshi cannot gain general execution authority')
    (repo / 'acceptance.txt').write_text('accepted\n\n')
    stale = json.loads(invoke('vigil', ['tools', 'graph', 'verify', '--run-dir', str(run)], repo, expected=1))
    assert stale['status'] == 'stale' and stale['consistent']
    updated = json.loads(invoke('vigil', plan_args, repo))
    assert updated['node_fingerprints']['components'] == plan['node_fingerprints']['components']
    assert updated['node_fingerprints']['source'] == plan['node_fingerprints']['source']
    assert updated['node_fingerprints']['contract'] != plan['node_fingerprints']['contract']
    plan_path.write_text(json.dumps(updated))
    changed = json.loads(invoke('vigil', ['tools', 'graph', 'resume', '--plan', str(plan_path), '--run-dir', str(run), '--approve-digest', updated['digest']], repo))
    assert changed['nodes']['components']['reused'] and changed['nodes']['source']['reused']
    assert not changed['nodes']['contract']['reused']
    checks.append('stale rejection and selective reuse after changed acceptance input')
    attempt = json.loads((run / changed['nodes']['contract']['artifacts'][-1]).read_text())
    raw = run / attempt['stdout']
    assert '9223372036854775807' in raw.read_text()
    raw.write_text('tampered')
    corrupt = json.loads(invoke('clyde', ['tools', 'graph', 'verify', '--run-dir', str(run)], repo, expected=1))
    assert not corrupt['consistent']
    checks.append('exact numeric stdout retention and tampered evidence rejection')

result = {'schema': 'starlit.graph.compatibility.v1', 'status': 'passed',
          'checks': checks, 'platform': sys.platform,
          'binaries': {name: {'path': path, 'sha256': hashlib.sha256(Path(path).read_bytes()).hexdigest()}
                       for name, path in binaries.items()},
          'external_ai_or_upload': False}
text = json.dumps(result, indent=2) + '\n'
if a.output:
    with a.output.open('x') as f:
        f.write(text)
print(text, end='')
