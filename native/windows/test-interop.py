#!/usr/bin/env python3
"""Run the authorized Mac/WIN11 TestNoop qualification from the Mac."""
import json, os, pathlib, socket, subprocess, tempfile, time, uuid
HERE=pathlib.Path(__file__).resolve().parent
MAC=pathlib.Path.home()/'BramMac/BramMac'
MAC_HUB=pathlib.Path.home()/'Downloads/grok-muse'
WINCTL=pathlib.Path.home()/'.local/bin/win11ctl'
ROOT=pathlib.Path(tempfile.mkdtemp(prefix='bram-win11-qualification-'))
RESULT={}
def win(script):
    r=subprocess.run([str(WINCTL),'ps',script],text=True,capture_output=True,timeout=40)
    if r.returncode: raise RuntimeError(r.stderr or r.stdout)
    return r.stdout.strip()
def ctl(*args):
    command=' '.join("'"+arg.replace("'","''")+"'" for arg in args)
    result=json.loads(win('$r=Get-Content $env:LOCALAPPDATA/Bram/install-info.json -Raw|ConvertFrom-Json; & $r.exe '+command))
    if not result.get('ok'): raise RuntimeError(result)
    return result

def until(check, seconds=35):
    deadline=time.monotonic()+seconds
    while time.monotonic()<deadline:
        value=check()
        if value:return value
        time.sleep(1)
    raise AssertionError('Verification deadline exceeded')

def discovery():
    local=subprocess.check_output([str(MAC),'discover'],text=True).splitlines()
    remote=ctl('discover')['discovered']
    return 'win11' in local and 'chriss-mac-mini' in remote

def messages(label):
    output=subprocess.check_output([str(MAC),'post','TestNoop@win11','TestNoop',label+' Mac to WIN11'],text=True)
    mac_id=output.strip().split()[-1]
    win_id=ctl('post','TestNoop@chriss-mac-mini','TestNoop',label+' WIN11 to Mac')['id']
    until(lambda:(MAC_HUB/'bram/delivered'/f'{win_id}.json').exists())
    until(lambda:(MAC_HUB/'bram/delivered'/f'{mac_id}.json').exists())
    remote=json.loads(win(f'$p=Join-Path $env:USERPROFILE "Downloads/grok-muse/bram/delivered/{mac_id}.json";if(Test-Path $p){{Get-Content $p -Raw}}else{{"null"}}'))
    assert remote and remote['to']=='TestNoop' and remote['delivered'].endswith('Z')
    returned=json.loads((MAC_HUB/'bram/delivered'/f'{win_id}.json').read_text())
    assert returned['to']=='TestNoop' and returned['delivered'].endswith('Z')
    remote_log=win('Get-Content $env:USERPROFILE/Downloads/grok-muse/bram.log -Tail 50 | Out-String')
    local_log=(MAC_HUB/'bram.log').read_text()
    assert mac_id in remote_log and win_id in remote_log and mac_id in local_log and win_id in local_log
    return {'mac_to_win11':mac_id,'win11_to_mac':win_id}

try:
    until(discovery)
    RESULT['bonjour_discovery_both_machines']=True
    RESULT['messages_before_restart']=messages('Bram qualification')
    # A wrong code is tried while an actual code window is open.
    ctl('pair-code')
    wrong=subprocess.run([str(MAC),'pair','10.0.0.244','47831','0'*32],capture_output=True,text=True)
    assert wrong.returncode!=0
    RESULT['wrong_pairing_code_rejected']=True
    wire=subprocess.check_output(['swift',str(HERE/'test-tamper.swift')])
    with socket.create_connection(('10.0.0.244',47831),timeout=10) as connection:
        connection.sendall(wire)
        assert json.loads(connection.recv(65536)).get('error')=='rejected'
    ctl('tamper','chriss-mac-mini')
    RESULT['tampering_rejected_both_receivers']=True
    # Exercise the reverse pairing initiator using a separate Mac identity.
    hub=ROOT/'isolated-mac';(hub/'bram').mkdir(parents=True)
    test_id='bram-interop-'+uuid.uuid4().hex[:8]
    (hub/'bram/machine.json').write_text(json.dumps({'id':test_id,'name':'Bram interop test Mac'}))
    env=dict(os.environ,BRAM_HUB=str(hub))
    process=subprocess.Popen([str(MAC),'relay-only'],env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)
    try:
        code=process.stdout.readline().strip().split(': ')[-1]
        until(lambda:(hub/'bram/network.json').exists(),5)
        port=str(json.loads((hub/'bram/network.json').read_text())['port'])
        ip=os.environ.get('BRAM_TEST_MAC_IP') or subprocess.check_output(['ipconfig','getifaddr','en1'],text=True).strip()
        ctl('pair-host',ip,port,code,test_id)
        assert 'win11' in json.loads((hub/'bram/peers.json').read_text())
        assert test_id in ctl('peers')['peers']
        RESULT['windows_initiated_pairing']=True
    finally:
        process.terminate();process.wait(timeout=5)
        # Back up trust, then remove only this run's verification peer.
        win('$p=Join-Path $env:USERPROFILE "Downloads/grok-muse/bram/peers.json";Copy-Item $p ($p+".backup."+(Get-Date -Format yyyyMMddHHmmss));$o=Get-Content $p -Raw|ConvertFrom-Json;$o.PSObject.Properties.Remove("'+test_id+'");$o|ConvertTo-Json -Depth 5|Set-Content -Encoding UTF8 $p')
    subprocess.run(['launchctl','kickstart','-k','gui/'+str(os.getuid())+'/ca.starlitdigital.brammac'],check=True)
    win('Stop-ScheduledTask -TaskName "Bram Windows";Start-Sleep -Seconds 1;Start-ScheduledTask -TaskName "Bram Windows"')
    until(lambda:'chriss-mac-mini' in ctl('peers')['peers'])
    assert 'win11' in subprocess.check_output([str(MAC),'peers'],text=True)
    until(discovery)
    RESULT['pairing_and_discovery_survive_both_app_restarts']=True
    RESULT['messages_after_restart']=messages('Bram qualification after restart')
    RESULT['private_firewall_enabled']=json.loads(win('(Get-NetFirewallProfile -Name Private).Enabled|ConvertTo-Json'))
    assert RESULT['private_firewall_enabled']
    RESULT['result']='PASS'
except Exception as error:
    RESULT['result']='FAIL';RESULT['error']=str(error)
finally:
    (ROOT/'results.json').write_text(json.dumps(RESULT,indent=2)+'\n')
    print(json.dumps(RESULT,indent=2));print('Evidence:',ROOT)
if RESULT['result']!='PASS':raise SystemExit(1)
