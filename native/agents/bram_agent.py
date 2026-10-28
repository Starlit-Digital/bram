#!/usr/bin/env python3
"""Pull based tool mailboxes over Bram's encrypted, nonexecuting TestNoop carrier."""
import argparse, datetime as dt, hashlib, json, os, pathlib, re, sqlite3, sys, tempfile, uuid
PREFIX='BRAM_AGENT_V1\n'
NAME=re.compile(r'^[A-Za-z][A-Za-z0-9_-]{0,47}$')
def stamp(): return dt.datetime.now(dt.timezone.utc).isoformat(timespec='seconds').replace('+00:00','Z')
def name(value):
    if not NAME.fullmatch(value): raise ValueError('Invalid agent or machine name')
    return value
def address(value):
    if not isinstance(value,str): raise ValueError('Invalid address')
    pieces=value.split('@')
    if len(pieces)!=2: raise ValueError('Use Agent@machine')
    return name(pieces[0])+'@'+name(pieces[1])
def atomic(path,obj):
    path.parent.mkdir(parents=True,exist_ok=True)
    fd,temp=tempfile.mkstemp(dir=path.parent,prefix='.',suffix='.tmp')
    try:
        with os.fdopen(fd,'w') as f: json.dump(obj,f,ensure_ascii=False);f.write('\n')
        os.replace(temp,path)
    finally:
        if os.path.exists(temp): os.unlink(temp)
class Mailbox:
    def __init__(self,hub):
        self.root=pathlib.Path(hub).expanduser()/'bram'
        self.machine=name(json.loads((self.root/'machine.json').read_text(encoding='utf-8-sig'))['id'])
        self.home=self.root/'agents';self.home.mkdir(mode=0o700,exist_ok=True)
        self.db=sqlite3.connect(self.home/'mailboxes.sqlite3',timeout=10)
        self.db.row_factory=sqlite3.Row
        self.db.executescript('''CREATE TABLE IF NOT EXISTS registrations(name TEXT PRIMARY KEY,description TEXT,updated TEXT);
CREATE TABLE IF NOT EXISTS messages(id TEXT PRIMARY KEY,recipient TEXT,body TEXT,state TEXT DEFAULT 'pending',claimed_at TEXT);
CREATE TABLE IF NOT EXISTS imported(file TEXT PRIMARY KEY,digest TEXT);
CREATE TABLE IF NOT EXISTS directories(machine TEXT PRIMARY KEY,body TEXT,received TEXT);
''')
        self.db.commit()
    def peers(self):
        path=self.root/'peers.json'
        return json.loads(path.read_text(encoding='utf-8-sig')) if path.exists() else {}
    def registrations(self): return [dict(r) for r in self.db.execute('SELECT * FROM registrations ORDER BY name')]
    def registered(self,agent):
        name(agent)
        if not self.db.execute('SELECT 1 FROM registrations WHERE name=?',(agent,)).fetchone(): raise ValueError('Register this local agent first')
        return agent+'@'+self.machine
    def send(self,sender,to,text,kind='request',thread=None,reply_to=None,message_id=None):
        sender=self.registered(sender);to=address(to)
        target=to.split('@')[1]
        if target!=self.machine and target not in self.peers(): raise ValueError('Destination machine is not paired')
        if not isinstance(text,str) or not text.strip(): raise ValueError('Text cannot be empty')
        mid=message_id or str(uuid.uuid4()).upper()
        payload={'schema':'bram.agent/1','id':mid,'kind':kind,'from':sender,'to':to,'thread':thread or mid,'in_reply_to':reply_to,'created':stamp(),'text':text}
        return self.queue(payload,sender.split('@')[0],target)
    def queue(self,payload,sender,target):
        text=PREFIX+json.dumps(payload,ensure_ascii=False,separators=(',',':'))
        if len(text.encode())>30000: raise ValueError('Agent envelope exceeds 30000 bytes')
        mid=payload['id']
        record={'id':mid,'from':sender,'to':'TestNoop' if target==self.machine else 'TestNoop@'+target,'machine':self.machine,'created':stamp(),'text':text}
        # Stable reply IDs make repeated reply commands harmless after a crash.
        state='queued'
        for folder,label in [('outbox','queued'),('delivered','transport_delivered'),('failed','transport_failed')]:
            path=self.root/folder/(mid+'.json')
            if path.exists():
                old=json.loads(path.read_text())
                previous=json.loads(old['text'][len(PREFIX):])
                if any(previous.get(k)!=payload.get(k) for k in ['kind','from','to','thread','in_reply_to','text']):
                    raise ValueError('This reply was already sent with different content')
                state=label;break
        else:atomic(self.root/'outbox'/(mid+'.json'),record)
        return {'id':mid,'to':payload.get('to',target),'state':state,'thread':payload.get('thread')}

    def sync(self):
        trusted=set(self.peers())|{self.machine}
        for path in sorted((self.root/'delivered').glob('*.json')):
            try:
                if path.stat().st_size>65536: continue
                raw=path.read_bytes();digest=hashlib.sha256(raw).hexdigest()
                old=self.db.execute('SELECT digest FROM imported WHERE file=?',(path.name,)).fetchone()
                if old and old['digest']==digest:continue
                outer=json.loads(raw)
                if not isinstance(outer,dict):continue
                if outer.get('to')!='TestNoop' or not outer.get('delivered') or outer.get('machine') not in trusted: continue
                text=outer.get('text','')
                if not isinstance(text,str) or not text.startswith(PREFIX):continue
                p=json.loads(text[len(PREFIX):])
                if not isinstance(p,dict):continue
                if p.get('schema')!='bram.agent/1' or p.get('id')!=outer.get('id'):continue
                uuid.UUID(p['id'])
                if p.get('kind')=='directory':
                    if outer.get('from')!='BramDirectory' or p.get('machine')!=outer['machine']:continue
                    entries=p.get('agents')
                    if not isinstance(entries,list) or len(entries)>64:continue
                    for entry in entries:
                        if not isinstance(entry,dict):raise ValueError('Invalid entry')
                        name(entry['name'])
                        if not isinstance(entry.get('description'),str) or len(entry['description'])>1000:raise ValueError('Invalid directory')
                    self.db.execute('INSERT OR REPLACE INTO directories VALUES(?,?,?)',(outer['machine'],json.dumps(entries),stamp()))
                else:
                    if p.get('kind') not in ['request','reply','notice']:continue
                    origin=address(p['from']);destination=address(p['to'])
                    if origin!=outer['from']+'@'+outer['machine'] or destination.split('@')[1]!=self.machine:continue
                    if not isinstance(p.get('thread'),str):continue
                    uuid.UUID(p['thread'])
                    if not isinstance(p.get('text'),str):continue
                    # Unknown local mailboxes remain visible through register later.
                    self.db.execute('INSERT OR IGNORE INTO messages(id,recipient,body) VALUES(?,?,?)',(p['id'],destination.split('@')[0],json.dumps(p)))
                self.db.execute('INSERT OR REPLACE INTO imported VALUES(?,?)',(path.name,digest))
            except (ValueError,KeyError,TypeError,OSError):continue
        self.db.commit()
    def discover(self):
        self.sync()
        result=[{'address':r['name']+'@'+self.machine,'description':r['description'],'availability':'registered, pull mailbox','updated':r['updated']} for r in self.registrations()]
        trusted=self.peers()
        for row in self.db.execute('SELECT * FROM directories'):
            if row['machine'] not in trusted or row['machine']==self.machine:continue
            for r in json.loads(row['body']):result.append({'address':r['name']+'@'+row['machine'],'description':r['description'],'availability':'last advertised, not a live health check','received':row['received']})
        return {'machine':self.machine,'paired_machines':sorted(trusted),'agents':result}
    def inbox(self,agent,all_states=False):
        self.registered(agent);self.sync()
        return [dict(json.loads(r['body']),state=r['state'],claimed_at=r['claimed_at']) for r in self.db.execute('SELECT * FROM messages WHERE recipient=? ORDER BY rowid',(agent,)) if all_states or r['state'] in ['pending','claimed']]
    def message(self,agent,mid):
        self.registered(agent);self.sync();uuid.UUID(mid)
        row=self.db.execute('SELECT * FROM messages WHERE id=? AND recipient=?',(mid,agent)).fetchone()
        if not row:raise ValueError('Message not found in this mailbox')
        return row
    def claim(self,agent,mid):
        self.message(agent,mid)
        with self.db:
            changed=self.db.execute("UPDATE messages SET state='claimed',claimed_at=? WHERE id=? AND recipient=? AND state='pending'",(stamp(),mid,agent)).rowcount
        if not changed:raise ValueError('Already claimed or finished, inspect before resuming')
        return {'id':mid,'state':'claimed'}
    def finish(self,agent,mid,state):
        row=self.message(agent,mid)
        if row['state'] not in ['claimed',state]:raise ValueError('Claim the message before finishing it')
        with self.db:self.db.execute('UPDATE messages SET state=? WHERE id=?',(state,mid))
        return {'id':mid,'state':state}
    def reply(self,agent,mid,text):
        row=self.message(agent,mid)
        if row['state'] not in ['claimed','done']:raise ValueError('Claim the message before replying')
        p=json.loads(row['body'])
        if p['kind']!='request':raise ValueError('Only requests get replies, preventing reply loops')
        reply_id=str(uuid.uuid5(uuid.UUID(mid),'reply:'+agent+'@'+self.machine)).upper()
        sent=self.send(agent,p['from'],text,'reply',p['thread'],mid,reply_id)
        self.finish(agent,mid,'done');return sent
    def announce(self):
        agents=self.registrations()
        if len(agents)>64:raise ValueError('Too many agents')
        result=[]
        for target in self.peers():
            payload={'schema':'bram.agent/1','kind':'directory','id':str(uuid.uuid4()).upper(),'machine':self.machine,'agents':agents,'created':stamp()}
            result.append(self.queue(payload,'BramDirectory',target))
        return result

def main(argv=None):
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--hub',default=os.environ.get('BRAM_HUB',str(pathlib.Path.home()/'Downloads/grok-muse')))
    sub=p.add_subparsers(dest='command',required=True)
    r=sub.add_parser('register');r.add_argument('agent');r.add_argument('--description',required=True)
    sub.add_parser('discover');sub.add_parser('announce')
    r=sub.add_parser('inbox');r.add_argument('--as',dest='agent',required=True);r.add_argument('--all',action='store_true')
    r=sub.add_parser('send');r.add_argument('--as',dest='agent',required=True);r.add_argument('--to',required=True);r.add_argument('--kind',choices=['request','notice'],default='request');r.add_argument('--text',required=True)
    for command in ['claim','reply','finish']:
        r=sub.add_parser(command);r.add_argument('--as',dest='agent',required=True);r.add_argument('--id',required=True)
        if command=='reply':r.add_argument('--text',required=True)
        if command=='finish':r.add_argument('--state',choices=['done','failed'],required=True)
    a=p.parse_args(argv);m=Mailbox(a.hub)
    if a.command=='register':
        name(a.agent)
        if len(a.description)>1000:raise ValueError('Description too long')
        with m.db:m.db.execute('INSERT OR REPLACE INTO registrations VALUES(?,?,?)',(a.agent,a.description,stamp()))
        result={'address':a.agent+'@'+m.machine,'state':'registered','automatic_execution':False}
    elif a.command=='discover':result=m.discover()
    elif a.command=='announce':result=m.announce()
    elif a.command=='inbox':result=m.inbox(a.agent,a.all)
    elif a.command=='send':result=m.send(a.agent,a.to,a.text,a.kind)
    elif a.command=='claim':result=m.claim(a.agent,a.id)
    elif a.command=='reply':result=m.reply(a.agent,a.id,a.text)
    elif a.command=='finish':result=m.finish(a.agent,a.id,a.state)
    print(json.dumps(result,indent=2,ensure_ascii=False))
if __name__=='__main__':
    try:main()
    except (ValueError,OSError,sqlite3.Error) as error:print(json.dumps({'error':str(error)}),file=sys.stderr);sys.exit(1)
