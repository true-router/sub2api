#!/usr/bin/env python3
"""Run the production binary against isolated PostgreSQL, Redis and HTTPS fixtures."""
import argparse, concurrent.futures, hashlib, http.server, json, os, pathlib, select, shutil, socket, ssl, subprocess, tempfile, threading, time, urllib.request, urllib.error

ROOT = pathlib.Path(__file__).resolve().parents[2]
parser = argparse.ArgumentParser()
parser.add_argument('--output', type=pathlib.Path, required=True)
parser.add_argument('--binary', type=pathlib.Path)
parser.add_argument('--hold', action='store_true')
args = parser.parse_args()
args.output.mkdir(parents=True, exist_ok=True)
tmp = pathlib.Path(tempfile.mkdtemp(prefix='sub2api-opencode-oauth-'))
checks, containers, processes, servers, calls = [], [], [], [], []
changed_files = subprocess.check_output(['git','-C',str(ROOT),'diff','--name-only'],text=True).splitlines() + subprocess.check_output(['git','-C',str(ROOT),'ls-files','--others','--exclude-standard'],text=True).splitlines()
source_files = {name:hashlib.sha256((ROOT/name).read_bytes()).hexdigest() for name in sorted(set(changed_files)) if not name.startswith('frontend/node_modules') and '__pycache__' not in name and (ROOT/name).is_file() and not (ROOT/name).is_symlink()}
complete = False
state = {'rotation': 0, 'pending': 0}
lock = threading.Lock()

def run(*cmd, **kwargs):
    return subprocess.check_output(cmd, text=True, stderr=subprocess.STDOUT, **kwargs).strip()

def check(name, condition):
    checks.append({'name': name, 'passed': bool(condition)})
    print(('PASS ' if condition else 'FAIL ') + name, flush=True)
    if not condition:
        raise AssertionError(name)

def port():
    with socket.socket() as s:
        s.bind(('127.0.0.1', 0))
        return s.getsockname()[1]

class Fixture(http.server.BaseHTTPRequestHandler):
    protocol_version = 'HTTP/1.1'
    def log_message(self, *_): pass
    def answer(self, body, status=200, content='application/json'):
        data = body.encode() if isinstance(body, str) else json.dumps(body).encode()
        self.send_response(status)
        self.send_header('Content-Type', content)
        self.send_header('Content-Length', str(len(data)))
        self.end_headers()
        self.wfile.write(data)
    def do_GET(self):
        if self.path == '/console/api/user': return self.answer({'id': 'fixture-user', 'email': 'fixture@example.invalid'})
        if self.path == '/console/api/orgs': return self.answer([{'id': 'org-fixture' if state['rotation'] < 2 else 'org-rotated', 'name': 'Fixture'}])
        if self.path.endswith('/models'): return self.answer({'object': 'list', 'data': [{'id': 'glm-5.3', 'object': 'model', 'owned_by': 'fixture'}]})
        self.answer({'error': 'unexpected_fixture_path'}, 404)
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers.get('Content-Length', 0))) or '{}')
        if self.path == '/console/auth/device/code':
            check('Device authorization uses official client and org scope', body == {'client_id': 'opencode-cli', 'supports_org_scope': True})
            return self.answer({'device_code': 'fixture-device', 'user_code': 'TEST-CODE', 'verification_uri_complete': 'https://opencode.ai/console/auth/device?code=TEST-CODE', 'expires_in': 600, 'interval': 1})
        if self.path == '/console/auth/device/token':
            if body.get('grant_type') == 'refresh_token':
                with lock:
                    current = state['rotation']
                    if body.get('refresh_token') == 'invalid': return self.answer({'error': 'invalid_grant'}, 400)
                    if body.get('refresh_token') != f'fixture-refresh-{current}': return self.answer({'error': 'invalid_grant'}, 400)
                    state['rotation'] += 1
                    current = state['rotation']
                return self.answer({'access_token': f'fixture-access-{current}', 'refresh_token': f'fixture-refresh-{current}', 'expires_in': 3600, 'org_id': 'org-fixture' if current < 2 else 'org-rotated'})
            if body.get('device_code') == 'denied': return self.answer({'error': 'access_denied'}, 400)
            if body.get('device_code') == 'expired': return self.answer({'error': 'expired_token'}, 400)
            with lock:
                state['pending'] += 1
                step = state['pending']
            if step == 1: return self.answer({'error': 'authorization_pending'}, 400)
            if step == 2: return self.answer({'error': 'slow_down'}, 400)
            return self.answer({'access_token': f'fixture-access-{state["rotation"]}', 'refresh_token': f'fixture-refresh-{state["rotation"]}', 'expires_in': 3600, 'org_id': 'org-fixture' if state['rotation'] < 2 else 'org-rotated'})
        legacy = self.path.startswith('/zen/go/')
        record = {'path': self.path, 'auth_ok': self.headers.get('Authorization') == ('Bearer fixture-key' if legacy else f'Bearer fixture-access-{state["rotation"]}'), 'org_ok': legacy or self.headers.get('X-Org-Id') == ('org-fixture' if state['rotation'] < 2 else 'org-rotated'), 'session_ok': bool(self.headers.get('X-OpenCode-Session')), 'no_api_key': not self.headers.get('X-Api-Key')}
        with lock: calls.append(record)
        if not all(record[k] for k in ['auth_ok', 'org_ok', 'session_ok', 'no_api_key']): return self.answer({'error': 'invalid_fixture_headers'}, 401)
        model = body.get('model', 'glm-5.3')
        if self.path.endswith('/chat/completions'):
            chunks = [ {'id':'chatcmpl-fixture','object':'chat.completion.chunk','model':model,'choices':[{'index':0,'delta':{'role':'assistant','content':'OK'},'finish_reason':None}]}, {'id':'chatcmpl-fixture','object':'chat.completion.chunk','model':model,'choices':[{'index':0,'delta':{},'finish_reason':'stop'}],'usage':{'prompt_tokens':10,'completion_tokens':1,'total_tokens':11}} ]
            data = ''.join('data: '+json.dumps(v)+'\n\n' for v in chunks)+'data: [DONE]\n\n'
        elif self.path.endswith('/responses'):
            response = {'id':'resp_fixture','object':'response','status':'completed','model':model,'output':[{'id':'msg_fixture','type':'message','role':'assistant','status':'completed','content':[{'type':'output_text','text':'OK','annotations':[]}]}],'usage':{'input_tokens':10,'output_tokens':1,'total_tokens':11}}
            events = [('response.created',{'type':'response.created','response':{**response,'status':'in_progress','output':[]}}),('response.output_text.delta',{'type':'response.output_text.delta','item_id':'msg_fixture','output_index':0,'content_index':0,'delta':'OK'}),('response.completed',{'type':'response.completed','response':response})]
            data = ''.join('event: '+n+'\ndata: '+json.dumps(v)+'\n\n' for n,v in events)
        elif self.path.endswith('/messages'):
            events = [('message_start', {'type':'message_start','message':{'id':'msg_fixture','type':'message','role':'assistant','model':model,'content':[],'usage':{'input_tokens':10,'output_tokens':0}}}),('content_block_start',{'type':'content_block_start','index':0,'content_block':{'type':'text','text':''}}),('content_block_delta',{'type':'content_block_delta','index':0,'delta':{'type':'text_delta','text':'OK'}}),('content_block_stop',{'type':'content_block_stop','index':0}),('message_delta',{'type':'message_delta','delta':{'stop_reason':'end_turn'},'usage':{'output_tokens':1}}),('message_stop',{'type':'message_stop'})]
            data = ''.join('event: '+n+'\ndata: '+json.dumps(v)+'\n\n' for n,v in events)
        else: return self.answer({'error':'unexpected_fixture_path'},404)
        self.answer(data, content='text/event-stream')

class Proxy(http.server.BaseHTTPRequestHandler):
    def log_message(self,*_): pass
    def do_CONNECT(self):
        if self.path != 'opencode.ai:443': self.send_error(403); return
        upstream = socket.create_connection(('127.0.0.1', tls_port))
        self.send_response(200);self.end_headers()
        try:
            while True:
                ready,_,_=select.select([self.connection,upstream],[],[],30)
                if not ready: break
                for source in ready:
                    data=source.recv(65536)
                    if not data:return
                    (upstream if source is self.connection else self.connection).sendall(data)
        except (OSError,ssl.SSLError):pass
        finally:upstream.close()

def start_server(server):
    servers.append(server)
    threading.Thread(target=server.serve_forever, daemon=True).start()
    return server.server_port

def api(method, path, body=None, auth=None, expected=200):
    data = None if body is None else json.dumps(body).encode()
    request=urllib.request.Request(origin+path,data=data,method=method,headers={'Content-Type':'application/json',**({'Authorization':'Bearer '+auth} if auth else {})})
    try:
        with urllib.request.urlopen(request,timeout=45) as response:
            status=response.status;raw=response.read().decode()
    except urllib.error.HTTPError as error:
        status=error.code;raw=error.read().decode()
    if status != expected: raise RuntimeError(f'{method} {path}: expected {expected}, got {status}: {raw[:700]}')
    try:
        result=json.loads(raw)
        return result.get('data',result)
    except json.JSONDecodeError:return raw

try:
    # Container IDs, ports and all temporary files belong to this invocation.
    pg=run('docker','run','-d','--label','task=opencode-oauth-e2e','-e','POSTGRES_PASSWORD=fixture','-e','POSTGRES_DB=sub2api','-p','127.0.0.1::5432','postgres:16-alpine').splitlines()[-1];containers.append(pg)
    redis=run('docker','run','-d','--label','task=opencode-oauth-e2e','-p','127.0.0.1::6379','redis:7-alpine').splitlines()[-1];containers.append(redis)
    pgport=run('docker','port',pg,'5432/tcp').rsplit(':',1)[1]
    redisport=run('docker','port',redis,'6379/tcp').rsplit(':',1)[1]
    for attempt in range(100):
        try:run('docker','exec',pg,'pg_isready','-h','127.0.0.1','-U','postgres');break
        except subprocess.CalledProcessError:time.sleep(.2)
    cert=tmp/'ca.pem';private=tmp/'ca.key'
    run('openssl','req','-x509','-newkey','rsa:2048','-nodes','-keyout',str(private),'-out',str(cert),'-days','1','-subj','/CN=opencode.ai','-addext','subjectAltName=DNS:opencode.ai','-addext','basicConstraints=critical,CA:TRUE')
    tls=http.server.ThreadingHTTPServer(('127.0.0.1',0),Fixture)
    context=ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER);context.load_cert_chain(cert,private);tls.socket=context.wrap_socket(tls.socket,server_side=True)
    tls_port=start_server(tls)
    proxy_port=start_server(http.server.ThreadingHTTPServer(('127.0.0.1',0),Proxy))
    app_port=port();origin=f'http://127.0.0.1:{app_port}'
    binary=args.binary.resolve() if args.binary else tmp/'server'
    if not args.binary:run('go','build','-o',str(binary),'./cmd/server',cwd=ROOT/'backend')
    (tmp/'data').mkdir()
    env={**os.environ,'AUTO_SETUP':'true','DATA_DIR':str(tmp/'data'),'DATABASE_HOST':'127.0.0.1','DATABASE_PORT':pgport,'DATABASE_USER':'postgres','DATABASE_PASSWORD':'fixture','DATABASE_DBNAME':'sub2api','DATABASE_SSLMODE':'disable','REDIS_HOST':'127.0.0.1','REDIS_PORT':redisport,'ADMIN_EMAIL':'fixture@example.invalid','ADMIN_PASSWORD':'FixturePassword123!','JWT_SECRET':'opencode-oauth-e2e-fixture-secret-123456789','SERVER_HOST':'127.0.0.1','SERVER_PORT':str(app_port),'TOKEN_REFRESH_ENABLED':'false','SSL_CERT_FILE':str(cert),'PRICING_DATA_DIR':str(tmp/'pricing'),'PRICING_REMOTE_URL':'https://127.0.0.1:1/pricing','PRICING_HASH_URL':'','PRICING_FALLBACK_FILE':str(ROOT/'backend/resources/model-pricing/model_prices_and_context_window.json')}
    log=(tmp/'server.log').open('w')
    app=subprocess.Popen([str(binary)],cwd=tmp,env=env,stdout=log,stderr=log);processes.append(app)
    resources={'harness_pid':os.getpid(),'temporary_directory':str(tmp),'container_ids':containers,'server_pid':app.pid,'origin':origin,'fixture_proxy_port':proxy_port}
    (args.output/'resources.json').write_text(json.dumps(resources,indent=2))
    for attempt in range(240):
        if app.poll() is not None:raise RuntimeError('Server exited: '+(tmp/'server.log').read_text()[-5000:])
        try:
            api('GET','/health');break
        except Exception:time.sleep(.25)
    else:raise RuntimeError('Server readiness timed out')
    check('Production binary and migrations boot against isolated PostgreSQL and Redis',True)
    session=api('POST','/api/v1/auth/login',{'email':'fixture@example.invalid','password':'FixturePassword123!'})
    jwt=session['access_token'];user=session['user']
    compliance=api('GET','/api/v1/admin/compliance',auth=jwt)
    api('POST','/api/v1/admin/compliance/accept',{'phrase':compliance['ack_phrase_en'],'language':'en'},jwt)
    api('POST','/api/v1/admin/opencode/oauth/device',{},expected=401)
    check('Device authorization requires administrator authentication',True)
    proxy=api('POST','/api/v1/admin/proxies',{'name':'OAuth fixture','protocol':'http','host':'127.0.0.1','port':proxy_port},jwt)
    proxy_id=proxy['id']
    device=api('POST','/api/v1/admin/opencode/oauth/device',{'proxy_id':proxy_id},jwt)
    check('Device authorization returns verification URL and user code',device['user_code']=='TEST-CODE')
    for expected_status in ['authorization_pending','slow_down','authorized']:
        result=api('POST','/api/v1/admin/opencode/oauth/poll',{'device_code':device['device_code'],'proxy_id':proxy_id},jwt)
        check('Device poll handles '+expected_status,result['status']==expected_status)
    credentials=result['credentials']
    for code in ['denied','expired']:
        api('POST','/api/v1/admin/opencode/oauth/poll',{'device_code':code,'proxy_id':proxy_id},jwt,expected=400)
        check('Device poll rejects '+code,True)
    api('POST','/api/v1/admin/opencode/oauth/import',{'credential':{}},jwt,expected=400)
    check('Credential import rejects malformed OAuth documents',True)
    native={'type':'oauth','access':'fixture-access-0','refresh':'fixture-refresh-0','expires':int((time.time()+3600)*1000),'metadata':{'orgID':'org-fixture','server':'https://opencode.ai/console'}}
    imported=api('POST','/api/v1/admin/opencode/oauth/import',{'credential':native},jwt)
    check('Native OpenCode credential import converts millisecond expiry',abs(int(imported['credentials']['expires_at'])-int(time.time())-3600)<3)
    native['metadata']['server']='https://attacker.invalid'
    api('POST','/api/v1/admin/opencode/oauth/import',{'credential':native},jwt,expected=400)
    check('Credential import rejects nonofficial Console servers',True)
    group=api('POST','/api/v1/admin/groups',{'name':'OAuth E2E','platform':'opencode_go','rate_multiplier':1,'subscription_type':'standard'},jwt)
    account=api('POST','/api/v1/admin/accounts',{'name':'OpenCode OAuth fixture','platform':'opencode_go','type':'oauth','credentials':credentials,'proxy_id':proxy_id,'concurrency':8,'priority':1,'group_ids':[group['id']]},jwt)
    account_id=account['id']
    check('OAuth account persists organization and tokens',account['credentials']['org_id']=='org-fixture' and account['type']=='oauth')
    api('POST',f'/api/v1/admin/users/{user["id"]}/balance',{'balance':100,'operation':'set'},jwt)
    key=api('POST','/api/v1/keys',{'name':'OAuth E2E','group_id':group['id']},jwt)['key']
    for protocol,path,body in [('chat_completions','/v1/chat/completions',{'messages':[{'role':'user','content':'Reply OK'}]}),('responses','/v1/responses',{'input':'Reply OK'}),('anthropic','/v1/messages',{'max_tokens':32,'messages':[{'role':'user','content':'Reply OK'}]})]:
        current=api('GET',f'/api/v1/admin/accounts/{account_id}',auth=jwt)['credentials']
        current['api_protocol']=protocol
        api('PUT',f'/api/v1/admin/accounts/{account_id}',{'credentials':current},jwt)
        probe=api('POST',f'/api/v1/admin/accounts/{account_id}/test',{'model_id':'glm-5.3'},jwt)
        check('Administrator connectivity test supports OAuth '+protocol,'test_complete' in probe and '"success":true' in probe)
        result=api('POST',path,{'model':'glm-5.3','stream':True,**body},key)
        check('Production gateway forwards OAuth '+protocol,'OK' in result)
    check('OAuth uses Go inference URLs and organization/session headers',len(calls)>=3 and all(x['path'].startswith('/inference/go/') and x['auth_ok'] and x['org_ok'] and x['session_ok'] and x['no_api_key'] for x in calls))
    models=api('GET','/v1/models',auth=key)
    check('Public model discovery supports OpenCode OAuth',bool(models))
    refreshed=api('POST',f'/api/v1/admin/accounts/{account_id}/refresh',{},jwt)
    check('Manual refresh persists rotated access and refresh tokens',refreshed['credentials_status']['has_refresh_token'] and json.loads(run('docker','exec',pg,'psql','-U','postgres','-d','sub2api','-Atc',f'SELECT credentials FROM accounts WHERE id={account_id}'))['refresh_token']=='fixture-refresh-1')
    check('Token refresh preserves configured protocol',refreshed['credentials']['api_protocol']=='anthropic')
    old=api('GET',f'/api/v1/admin/accounts/{account_id}',auth=jwt)['credentials'];old['expires_at']=str(int(time.time())-1)
    api('PUT',f'/api/v1/admin/accounts/{account_id}',{'credentials':old},jwt)
    def concurrent_request(_):return api('POST','/v1/messages',{'model':'glm-5.3','max_tokens':32,'stream':True,'messages':[{'role':'user','content':'OK'}]},key)
    with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:results=list(pool.map(concurrent_request,range(4)))
    check('Concurrent expired requests rotate once through the shared refresh lock',state['rotation']==2 and all('OK' in x for x in results))
    check('Refresh persists the new workspace and subsequent requests use it',api('GET',f'/api/v1/admin/accounts/{account_id}',auth=jwt)['credentials']['org_id']=='org-rotated')
    current=api('GET',f'/api/v1/admin/accounts/{account_id}',auth=jwt)['credentials'];current['refresh_token']='invalid';current['expires_at']=str(int(time.time())-1)
    api('PUT',f'/api/v1/admin/accounts/{account_id}',{'credentials':current},jwt)
    api('POST',f'/api/v1/admin/accounts/{account_id}/refresh',{},jwt,expected=400)
    check('Revoked refresh token requires reauthorization',True)
    restored={**credentials,'access_token':'fixture-access-2','refresh_token':'fixture-refresh-2','org_id':'org-rotated'}
    api('POST',f'/api/v1/admin/accounts/{account_id}/apply-oauth-credentials',{'type':'oauth','credentials':restored},jwt)
    check('Reauthorization restores credentials using the existing account endpoint',True)
    usage=api('GET',f'/api/v1/admin/accounts/{account_id}/opencode-go-usage',auth=jwt)
    check('OAuth accounts explicitly report legacy usage as unavailable',usage['eligible'] is False)
    check('Legacy API-key usage is not queried with OAuth credentials',True)
    # Replace fixture account credentials through API and verify existing API-key behavior.
    legacy={'api_key':'fixture-key','account_mode':'go','api_protocol':'chat_completions'}
    api('PUT',f'/api/v1/admin/accounts/{account_id}',{'type':'apikey','credentials':legacy},jwt)
    result=api('POST','/v1/chat/completions',{'model':'glm-5.3','stream':True,'messages':[{'role':'user','content':'OK'}]},key)
    check('Existing API-key accounts retain the legacy Go endpoint', 'OK' in result and calls[-1]['path']=='/zen/go/v1/chat/completions')
    # Leave an OAuth fixture ready for optional browser acceptance.
    api('PUT',f'/api/v1/admin/accounts/{account_id}',{'type':'oauth','credentials':restored},jwt)
    for attempt in range(20):
        count=int(run('docker','exec',pg,'psql','-U','postgres','-d','sub2api','-Atc',"SELECT count(*) FROM audit_logs WHERE path LIKE '%/opencode/oauth/import%' AND request_body='<credential-bearing body omitted>'"))
        if count:break
        time.sleep(.2)
    check('OAuth credential bodies are omitted from durable administrator audit logs',count>0)
    complete = True
    if args.hold:
        ready=tmp/'ready.json';ready.write_text(json.dumps({'origin':origin,'access_token':jwt,'user':user,'account_id':account_id,'proxy_id':proxy_id,'fixture_credential':{'type':'oauth','access':f'fixture-access-{state["rotation"]}','refresh':f'fixture-refresh-{state["rotation"]}','expires':int((time.time()+3600)*1000),'metadata':{'server':'https://opencode.ai/console','orgID':'org-fixture' if state['rotation']<2 else 'org-rotated'}}}));ready.chmod(0o600)
        print('READY '+str(ready),flush=True)
        while not (tmp/'release').exists():time.sleep(.5)
finally:
    for process in reversed(processes):
        if process.poll() is None:
            process.terminate()
            try:process.wait(timeout=15)
            except subprocess.TimeoutExpired:print('Process still running: '+str(process.pid),flush=True)
    for server in servers:server.shutdown();server.server_close()
    cleanup=[]
    for container in containers:
        try:
            run('docker','stop','-t','10',container)
            run('docker','rm','-v',container)
            cleanup.append(container)
        except Exception:pass
    receipt={'source_head':run('git','-C',str(ROOT),'rev-parse','HEAD'),'source_files':source_files,'binary_sha256':hashlib.sha256(binary.read_bytes()).hexdigest() if 'binary' in globals() and binary.exists() else None,'checks':checks,'upstream_requests':calls,'released_container_ids':cleanup,'passed':complete and bool(checks) and all(x['passed'] for x in checks),'temporary_directory':str(tmp)}
    (args.output/'receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
    if len(cleanup)==len(containers) and all(p.poll() is not None for p in processes):shutil.rmtree(tmp)
