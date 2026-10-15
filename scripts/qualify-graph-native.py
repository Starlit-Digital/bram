#!/usr/bin/env python3
"""Run an owned graph test executable on an explicit SSH host in a new temp directory."""
import argparse
import base64
import hashlib
import json
from pathlib import Path
import re
import shlex
import subprocess
import uuid

p = argparse.ArgumentParser(description=__doc__)
p.add_argument('--host', required=True)
p.add_argument('--os', choices=['windows', 'linux'], required=True)
p.add_argument('--binary', type=Path, required=True)
p.add_argument('--output-dir', type=Path, required=True)
a = p.parse_args()
a.output_dir.mkdir(parents=False, exist_ok=False)
binary = a.binary.resolve(strict=True)
digest = hashlib.sha256(binary.read_bytes()).hexdigest()

def call(args, timeout=150):
    r = subprocess.run(args, capture_output=True, text=True, encoding='utf-8',
                       errors='replace', timeout=timeout)
    if r.returncode:
        raise RuntimeError(f'Command failed ({r.returncode}): {r.stderr}\n{r.stdout}')
    return r.stdout.strip()

def ssh(command):
    return call(['ssh', '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=8', a.host, command])

def powershell(script):
    script = "$ErrorActionPreference='Stop'; $ProgressPreference='SilentlyContinue'; [Console]::OutputEncoding=[System.Text.UTF8Encoding]::new($false); " + script
    code = base64.b64encode(script.encode('utf-16le')).decode('ascii')
    return 'powershell.exe -NoProfile -NonInteractive -EncodedCommand ' + code

label = 'starlit-graph-' + uuid.uuid4().hex
if a.os == 'windows':
    directory = ssh(powershell(f"$d=Join-Path $env:TEMP '{label}'; New-Item -ItemType Directory -Path $d | Out-Null; [Console]::WriteLine($d)"))
    remote = directory.replace('\\', '/') + '/graph.test.exe'
else:
    directory = ssh('mktemp -d /tmp/' + label + '.XXXXXX')
    remote = directory + '/graph.test'
print('Native fixture directory:', a.host, directory, flush=True)
call(['scp', '-q', str(binary), a.host + ':' + remote])
if a.os == 'windows':
    checksum = ssh(powershell(f"(Get-FileHash -Algorithm SHA256 -LiteralPath '{remote}').Hash.ToLowerInvariant()"))
    assert checksum == digest, (checksum, digest)
    command = powershell(f"& '{remote}' '-test.run=^TestGraphNative' '-test.v' '-test.timeout=90s'; exit $LASTEXITCODE")
    platform = ssh(powershell("Get-CimInstance Win32_OperatingSystem | Select-Object Caption,Version,OSArchitecture | ConvertTo-Json -Compress"))
else:
    checksum = ssh('sha256sum ' + shlex.quote(remote)).split()[0]
    assert checksum == digest, (checksum, digest)
    ssh('chmod 700 ' + shlex.quote(remote))
    command = shlex.quote(remote) + ' -test.run=^TestGraphNative -test.v -test.timeout=90s'
    platform = ssh('uname -srm')
result = subprocess.run(['ssh', '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=8', a.host, command],
                        capture_output=True, text=True, encoding='utf-8', errors='replace', timeout=120)
(a.output_dir / 'stdout.txt').write_text(result.stdout, encoding='utf-8')
(a.output_dir / 'stderr.txt').write_text(result.stderr, encoding='utf-8')
expected = {'TestGraphNativeRunResumeFreshnessAndTamper',
            'TestGraphNativeKernelLockReleasedAfterProcessDeath',
            'TestGraphNativeInterruptedCheckpointRecovery',
            'TestGraphNativeCancellationAndOutputBound', 'TestGraphNativeContractSchemas', 'TestGraphNativeConfigurationFreshness', 'TestGraphNativeProtocolContracts'}
passed = set(re.findall(r'--- PASS: (TestGraphNative\w+)', result.stdout))
status = 'passed' if result.returncode == 0 and expected <= passed else 'failed'
report = {'schema': 'starlit.graph.native-qualification.v1', 'status': status,
          'host': a.host, 'platform': platform, 'binary_sha256': digest,
          'remote_directory': directory, 'exit_code': result.returncode,
          'required_tests': sorted(expected), 'passed_tests': sorted(passed),
          'scope': 'Owned standalone public graph test binary only; no CLI install, application launch, AI or upload.',
          'artifacts_retained': True}
(a.output_dir / 'result.json').write_text(json.dumps(report, indent=2) + '\n', encoding='utf-8')
print(json.dumps(report, indent=2), flush=True)
raise SystemExit(0 if status == 'passed' else 1)
