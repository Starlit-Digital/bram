#!/usr/bin/env python3
"""Exercise fixture mailboxes through a temporary Mac TestNoop relay, with no UI."""
import importlib.util,json,os,pathlib,subprocess,tempfile,time
spec=importlib.util.spec_from_file_location('bram_agent',pathlib.Path(__file__).with_name('bram_agent.py'));a=importlib.util.module_from_spec(spec);spec.loader.exec_module(a)
with tempfile.TemporaryDirectory(prefix='bram-agent-noop-') as folder:
 hub=pathlib.Path(folder);(hub/'bram').mkdir();a.atomic(hub/'bram/machine.json',{'id':'test-noop-mailboxes','name':'TestNoop mailboxes'})
 process=subprocess.Popen([str(pathlib.Path.home()/'BramMac/BramMac'),'relay-only'],env=dict(os.environ,BRAM_HUB=folder),stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
 try:
  m=a.Mailbox(hub)
  with m.db:
   for actor in ['TestNoopA','TestNoopB']:m.db.execute('INSERT INTO registrations VALUES(?,?,?)',(actor,'TestNoop fixture',a.stamp()))
  request=m.send('TestNoopA','TestNoopB@test-noop-mailboxes','Fixture request')['id']
  def wait(actor):
   deadline=time.monotonic()+20
   while time.monotonic()<deadline:
    rows=m.inbox(actor)
    if rows:return rows[0]
    time.sleep(.2)
   raise AssertionError('TestNoop mailbox delivery timed out')
  assert wait('TestNoopB')['id']==request;m.claim('TestNoopB',request)
  sent=m.reply('TestNoopB',request,'Fixture reply');reply=wait('TestNoopA');assert reply['id']==sent['id'] and reply['in_reply_to']==request
  m.claim('TestNoopA',reply['id']);m.finish('TestNoopA',reply['id'],'done')
  assert not m.inbox('TestNoopB') and not m.inbox('TestNoopA')
  print(json.dumps({'result':'PASS','transport':'isolated Mac TestNoop relay','request':request,'reply':reply['id'],'real_app_input':False},indent=2));m.db.close()
 finally:process.terminate();process.wait(timeout=5)
