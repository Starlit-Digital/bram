#!/usr/bin/env python3
"""Package the already published Windows app and its source, retaining notices."""
from pathlib import Path
import json,zipfile
repo=Path(__file__).resolve().parents[2]
base=repo/'.build/windows'
assets=json.loads((repo/'native/windows/obj/project.assets.json').read_text())
packages=Path(next(iter(assets['packageFolders'])))
with zipfile.ZipFile(repo/'.build/BramWindows.zip','w',zipfile.ZIP_DEFLATED) as z:
    for f in base.rglob('*'):
        if f.is_file():z.write(f,str(f.relative_to(base)))
    z.write(repo/'LICENSE','LICENSE-Bram.txt')
    z.write(repo/'native/windows/README.md','README.md')
    for name,info in assets['libraries'].items():
        if info['type']!='package':continue
        pkg=packages/info['path']
        for f in pkg.iterdir():
            if f.is_file() and ('license' in f.name.lower() or f.suffix=='.nuspec'):
                z.write(f,'licenses/'+name+'/'+f.name)
with zipfile.ZipFile(repo/'.build/BramWindows-source.zip','w',zipfile.ZIP_DEFLATED) as z:
    for f in (repo/'native/windows').iterdir():
        if f.is_file() and '.backup.' not in f.name:z.write(f,f.name)
print('Packaged Windows app and source')
