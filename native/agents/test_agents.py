import importlib.util,json,pathlib,tempfile,unittest
spec=importlib.util.spec_from_file_location('bram_agent',pathlib.Path(__file__).with_name('bram_agent.py'));a=importlib.util.module_from_spec(spec);spec.loader.exec_module(a)
class Tests(unittest.TestCase):
 def setUp(self):
  self.temp=tempfile.TemporaryDirectory();self.hub=pathlib.Path(self.temp.name);root=self.hub/'bram';root.mkdir();(root/'delivered').mkdir();a.atomic(root/'machine.json',{'id':'test-mac'});a.atomic(root/'peers.json',{'test-win':{}});self.m=a.Mailbox(self.hub)
  with self.m.db:
   for agent in ['TestNoopA','TestNoopB']:self.m.db.execute('INSERT INTO registrations VALUES(?,?,?)',(agent,'fixture',a.stamp()))
 def tearDown(self):self.m.db.close();self.temp.cleanup()
 def deliver(self,mid,change=None):
  p=self.m.root/'outbox'/(mid+'.json');o=json.loads(p.read_text());o['to']='TestNoop';o['delivered']=a.stamp()
  if change:change(o)
  a.atomic(self.m.root/'delivered'/p.name,o);p.unlink()
 def test_conversation(self):
  mid=self.m.send('TestNoopA','TestNoopB@test-mac','Question')['id'];self.deliver(mid)
  self.assertEqual(len(self.m.inbox('TestNoopB')),1);self.m.claim('TestNoopB',mid)
  with self.assertRaises(ValueError):self.m.claim('TestNoopB',mid)
  sent=self.m.reply('TestNoopB',mid,'Answer');self.assertEqual(self.m.reply('TestNoopB',mid,'Answer')['id'],sent['id'])
  with self.assertRaises(ValueError):self.m.reply('TestNoopB',mid,'Changed answer')
  self.deliver(sent['id']);reply=self.m.inbox('TestNoopA')[0];self.assertEqual(reply['in_reply_to'],mid);self.assertEqual(reply['thread'],mid)
  self.m.claim('TestNoopA',reply['id'])
  with self.assertRaises(ValueError):self.m.reply('TestNoopA',reply['id'],'Loop')
  self.m.finish('TestNoopA',reply['id'],'done');self.assertEqual(self.m.inbox('TestNoopA'),[])
 def test_auth_binding(self):
  mid=self.m.send('TestNoopA','TestNoopB@test-mac','Question')['id'];self.deliver(mid,lambda o:o.update(machine='unpaired'))
  self.assertEqual(self.m.inbox('TestNoopB'),[])
  mid=self.m.send('TestNoopA','TestNoopB@test-mac','Question')['id'];self.deliver(mid,lambda o:o.update({'from':'Forged'}));self.assertEqual(self.m.inbox('TestNoopB'),[])
 def test_validation(self):
  for target in ['../../etc','Tool@../../x','A@unpaired']:
   with self.assertRaises(ValueError):self.m.send('TestNoopA',target,'No')
  with self.assertRaises(ValueError):self.m.send('TestNoopA','TestNoopB@test-mac','x'*30001)
 def test_remote_sender_archive_is_not_inbound(self):
  mid=self.m.send('TestNoopA','TestNoopB@test-win','Remote')['id'];p=self.m.root/'outbox'/(mid+'.json');o=json.loads(p.read_text());o['relayed']=a.stamp();a.atomic(self.m.root/'delivered'/p.name,o);self.assertEqual(self.m.inbox('TestNoopB'),[])
 def test_idempotent_import(self):
  mid=self.m.send('TestNoopA','TestNoopB@test-mac','Hi')['id'];self.deliver(mid);self.m.sync();self.m.sync();self.assertEqual(len(self.m.inbox('TestNoopB')),1)
 def test_directory(self):
  sent=self.m.announce()[0];self.deliver(sent['id'],lambda o:o.update(machine='test-win'))
  p=self.m.root/'delivered'/(sent['id']+'.json');o=json.loads(p.read_text());v=json.loads(o['text'][len(a.PREFIX):]);v['machine']='test-win';o['text']=a.PREFIX+json.dumps(v);a.atomic(p,o)
  self.assertTrue(any(r['address']=='TestNoopA@test-win' for r in self.m.discover()['agents']))
if __name__=='__main__':unittest.main()
