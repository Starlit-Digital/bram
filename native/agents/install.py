#!/usr/bin/env python3
"""Install a real bram-agent executable and retain provenance and prior copies."""
import datetime,hashlib,json,os,pathlib,shutil
source=pathlib.Path(__file__).with_name('bram_agent.py').resolve()
home=pathlib.Path.home();dest=home/'.local/bin/bram-agent';data=home/'.local/share/bram-agent'
stamp=datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ')
receipt=data/'install-info.json';history=data/'installs'/stamp
history.mkdir(parents=True,exist_ok=False);dest.parent.mkdir(parents=True,exist_ok=True)
for old in [dest,receipt]:
 if old.exists():shutil.copy2(old,history/old.name)
staged=dest.with_name('.bram-agent.stage.'+stamp);shutil.copy2(source,staged);staged.chmod(0o755);os.replace(staged,dest)
info={'installed':stamp,'source':str(source),'executable':str(dest),'sha256':hashlib.sha256(dest.read_bytes()).hexdigest(),'runtime':'Python 3 standard library','mode':'pull mailboxes, no automatic AI execution'}
receipt.write_text(json.dumps(info,indent=2)+'\n');print(json.dumps(info,indent=2))
