#!/usr/bin/env python3
"""失败方式：取消未传递、独立时限失效、部分用量误标、范围外行为改变。
使用真实配置载入、HTTP 转发和本机模拟上游。输出可重复验收工件。
"""
import argparse
import http.client
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import select
import socket
import subprocess
import threading
import time
import traceback

parser=argparse.ArgumentParser()
parser.add_argument('--binary',type=Path,required=True)
parser.add_argument('--out',type=Path,required=True)
parser.add_argument('--provider',choices=['anthropic','openai'],default='anthropic')
args=parser.parse_args();args.out.mkdir(parents=True,exist_ok=False)
checks=[];provider_requests=[]
state={'release':threading.Event(),'started':threading.Event(),'closed':threading.Event(),'mode':'hold'}
env={**os.environ,'JWT_SECRET':'synthetic-isolated-jwt-secret-0123456789'}

def frame(kind,**fields):
    return ('event: '+kind+'\ndata: '+json.dumps({'type':kind,**fields})+'\n\n').encode()

def check(name,ok,**fields):
    checks.append({'name':name,'result':'PASS' if ok else 'FAIL',**fields})
    (args.out/'checks.json').write_text(json.dumps(checks,ensure_ascii=False,indent=2))
    assert ok,name

class Provider(BaseHTTPRequestHandler):
    def log_message(self,*_):pass
    def do_POST(self):
        self.rfile.read(int(self.headers['Content-Length']))
        provider_requests.append({'path':self.path,'time':time.monotonic()})
        state['started'].set()
        mode=state['mode']
        if mode=='conflict':
            data=json.dumps({'error':{'code':'execution_state_unknown','message':'PRIVATE_PROVIDER_MESSAGE','details':{'retryable':False,'private':'DO_NOT_EXPOSE'}}}).encode()
            self.send_response(409);self.send_header('Content-Type','application/json');self.send_header('Content-Length',str(len(data)));self.end_headers();self.wfile.write(data);return
        if mode!='before-headers':
            self.send_response(200);self.send_header('Content-Type','text/event-stream');self.end_headers()
            if args.provider=='openai':
                self.wfile.write(frame('response.created',response={'id':'resp_fixture','object':'response','model':'gpt-4.1','status':'in_progress','output':[],'usage':{'input_tokens':10,'output_tokens':5,'total_tokens':15}})+frame('response.output_text.delta',item_id='item_fixture',output_index=0,content_index=0,delta='SYNTHETIC'))
            else:
                self.wfile.write(frame('message_start',message={'id':'msg_fixture','type':'message','role':'assistant','model':('gpt-4.1' if args.provider=='openai' else 'claude-sonnet-4-6'),'content':[],
                    'usage':{'input_tokens':10,'output_tokens':0}})+frame('content_block_start',index=0,content_block={'type':'text','text':''})+
                    frame('content_block_delta',index=0,delta={'type':'text_delta','text':'SYNTHETIC'})+
                    frame('message_delta',delta={},usage={'output_tokens':3})+frame('message_delta',delta={},usage={'output_tokens':5}))
            self.wfile.flush()
        try:
            while not state['release'].wait(.02):
                r,_,_=select.select([self.connection],[],[],0)
                if r and not self.connection.recv(1,socket.MSG_PEEK):state['closed'].set();return
                if mode not in ['before-headers','normal-silent','idle-timeout']:self.wfile.write(frame('ping'));self.wfile.flush()
            if mode=='before-headers':return
            if args.provider=='openai':
                self.wfile.write(frame('response.output_text.done',item_id='item_fixture',output_index=0,content_index=0,text='SYNTHETIC')+frame('response.completed',response={'id':'resp_fixture','object':'response','model':'gpt-4.1','status':'completed','output':[{'id':'item_fixture','type':'message','role':'assistant','content':[{'type':'output_text','text':'SYNTHETIC','annotations':[]}]}],**({'usage':{'input_tokens':10,'output_tokens':6,'total_tokens':16}} if mode not in ['missing-usage','null-usage'] else ({'usage':{'input_tokens':None,'output_tokens':None}} if mode=='null-usage' else {}))}))
            else:
                self.wfile.write(frame('content_block_stop',index=0)+(frame('message_delta',delta={'stop_reason':'end_turn'},usage={'output_tokens':6}) if mode not in ['missing-usage','null-usage'] else frame('message_delta',delta={'stop_reason':'end_turn'},**({'usage':{'output_tokens':None}} if mode=='null-usage' else {})))+frame('message_stop'))
            if mode=='duplicate-usage':
                if args.provider=='openai':self.wfile.write(frame('response.completed',response={'id':'resp_fixture','status':'completed','usage':{'input_tokens':10,'output_tokens':6,'total_tokens':16}}))
                else:self.wfile.write(frame('message_delta',delta={'stop_reason':'end_turn'},usage={'output_tokens':6})+frame('message_stop'))
            self.wfile.flush()
        except (BrokenPipeError,ConnectionResetError):state['closed'].set()

provider=ThreadingHTTPServer(('127.0.0.1',0),Provider);provider.daemon_threads=True
threading.Thread(target=provider.serve_forever,daemon=True).start()

def start(extra,name):
    sock=socket.socket();sock.bind(('127.0.0.1',0));port=sock.getsockname()[1];sock.close()
    receipts=args.out/(name+'-receipts.jsonl')
    log=(args.out/(name+'.log')).open('w')
    p=subprocess.Popen([str(args.binary)],env={**env,'E2E_TARGET':f'http://127.0.0.1:{provider.server_port}',
        'E2E_PROVIDER':args.provider,'E2E_KEY':'sk-fixture','E2E_LISTEN':f'127.0.0.1:{port}','E2E_RECEIPT':str(receipts),**extra},stdout=log,stderr=log,cwd=args.out)
    for _ in range(100):
        try:
            c=http.client.HTTPConnection('127.0.0.1',port,timeout=.2);c.request('GET','/health');r=c.getresponse();c.close()
            if r.status==200:return p,port,receipts,log
        except OSError:pass
        if p.poll() is not None:break
        time.sleep(.025)
    p.terminate();p.wait();log.close();raise RuntimeError(name+' 未就绪')

def stop(p,log):
    p.terminate()
    try:p.wait(timeout=3)
    except subprocess.TimeoutExpired:p.kill();p.wait()
    log.close()

passed=False
try:
    for value,valid in [(None,True),('0',True),('1',True),('300',True),('-1',False),('301',False),('1.5',False),('0.5',False),('invalid',False)]:
        e=dict(env)
        if value is not None:e['GATEWAY_CLIENT_DISCONNECT_DRAIN_TIMEOUT']=value
        r=subprocess.run([str(args.binary),'--config-check'],env=e,capture_output=True,text=True,cwd=args.out)
        check('启动配置 '+str(value),(r.returncode==0)==valid,exit_code=r.returncode)
    for value,valid in [('0',True),('300',True),('1.5',False),('1.0',False),('-1',False),('301',False)]:
        config_file=args.out/'case-config.yaml'
        config_file.write_text("gateway:\n  client_disconnect_drain_timeout: "+value+"\n")
        r=subprocess.run([str(args.binary),'--config-check'],env={**env,'CONFIG_FILE':str(config_file)},capture_output=True,text=True,cwd=args.out)
        check('YAML 配置 '+value,(r.returncode==0)==valid,exit_code=r.returncode)
    config_file.unlink()
    for endpoint in ['/v1/messages','/v1/chat/completions','/v1/responses']:
        for mode,timeout,interval in [('before-headers','0','180'),('hold','0','0'),('hold','1','300'),('normal','0','0'),('missing-usage','0','0'),('null-usage','0','0'),('normal-silent','0','0'),('idle-timeout','0','30'),('finish-race','0','0'),('finish-race','1','0'),('duplicate-usage','0','0'),('late-usage','1','0')]:
            # 基线 OpenAI Responses 透传尚无数据间隔计时器；不扩大本次取消变更。
            if args.provider=='openai' and endpoint=='/v1/responses' and mode=='idle-timeout':continue
            state['mode']=mode
            for key in ['release','started','closed']:state[key].clear()
            name=endpoint.rsplit('/',1)[-1]+'-'+mode+'-'+timeout
            p,port,receipts,log=start({'GATEWAY_CLIENT_DISCONNECT_DRAIN_TIMEOUT':timeout,'GATEWAY_STREAM_DATA_INTERVAL_TIMEOUT':interval},name)
            try:
                body={'model':('gpt-4.1' if args.provider=='openai' else 'claude-sonnet-4-6'),'max_tokens':64,'stream':True,'messages':[{'role':'user','content':'SYNTHETIC'}]}
                if endpoint.endswith('/responses'):body={'model':('gpt-4.1' if args.provider=='openai' else 'claude-sonnet-4-6'),'stream':True,'input':'SYNTHETIC'}
                raw=json.dumps(body).encode();client=socket.create_connection(('127.0.0.1',port),timeout=5)
                client.sendall(f'POST {endpoint} HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: {len(raw)}\r\n\r\n'.encode()+raw)
                check(name+':上游开始',state['started'].wait(5))
                if mode!='before-headers':
                    data=b''
                    while b'SYNTHETIC' not in data:
                        chunk=client.recv(8192);assert chunk,'客户端未收到正文';data+=chunk
                if mode=='finish-race':
                    before=len(provider_requests)
                    threading.Thread(target=state['release'].set,daemon=True).start()
                    client.shutdown(socket.SHUT_RDWR);client.close();client.close()
                    check(name+':竞争不重新派发',len(provider_requests)==before)
                elif mode=='late-usage':
                    client.shutdown(socket.SHUT_RDWR);client.close()
                    time.sleep(.2);state['release'].set()
                elif mode=='idle-timeout':
                    check(name+':在线静默由数据间隔取消',state['closed'].wait(32))
                    client.close();state['release'].set()
                elif mode in ['normal','missing-usage','null-usage','normal-silent','duplicate-usage']:
                    # 保持连接超过兼容模式 1 秒，证明未断开时不启动断开时限。
                    time.sleep(1.3);check(name+':正常长流未取消',not state['closed'].is_set())
                    state['release'].set()
                    while True:
                        chunk=client.recv(8192)
                        if not chunk:break
                        data+=chunk
                        if b'message_stop' in data or b'[DONE]' in data or b'response.completed' in data:break
                    client.close()
                else:
                    time.sleep(.08)
                    began=time.monotonic();client.shutdown(socket.SHUT_RDWR);client.close()
                    if timeout=='1':check(name+':未提前取消',not state['closed'].wait(.55))
                    check(name+':上游按独立时限取消',state['closed'].wait(2),elapsed_ms=round((time.monotonic()-began)*1000,1))
                    state['release'].set()
                deadline=time.monotonic()+4
                while (not receipts.exists() or not receipts.read_text().strip()) and time.monotonic()<deadline:time.sleep(.02)
                rows=[json.loads(x) for x in receipts.read_text().splitlines()]
                result=rows[-1]['result']
                if mode in ['normal','missing-usage','null-usage','normal-silent','duplicate-usage']:
                    check(name+':完整用量',result is not None and (result['Usage']['output_tokens']==6 and result.get('UsageIncomplete') is False if mode in ['normal','normal-silent','duplicate-usage'] else result.get('UsageIncomplete') is True))
                elif mode=='late-usage':
                    check(name+':断开后时限内最终用量保留',result is not None and result['Usage']['output_tokens']==6 and result.get('UsageIncomplete') is False)
                elif mode=='finish-race':
                    check(name+':竞争仅收尾一次',len(rows)==1 and result is not None and isinstance(result.get('UsageIncomplete'),bool) and 5<=result['Usage']['output_tokens']<=6)
                elif mode!='before-headers':
                    check(name+':部分用量保留且不累加',result is not None and result['Usage']['input_tokens']==10 and result['Usage']['output_tokens']==5)
                    check(name+':部分用量明确标记',result.get('UsageIncomplete') is True)
            finally:state['release'].set();stop(p,log)
    for policy in ['default','explicit']:
        state['mode']='before-headers'
        for key in ['release','started','closed']:state[key].clear()
        p,port,_,log=start({'GATEWAY_CLIENT_DISCONNECT_DRAIN_TIMEOUT':'0'},'legacy-'+policy)
        try:
            client=socket.create_connection(('127.0.0.1',port),timeout=5)
            client.sendall(('POST /legacy/'+policy+' HTTP/1.1\r\nHost: localhost\r\nContent-Length: 2\r\n\r\n{}').encode())
            check('旧策略 '+policy+':实际派发',state['started'].wait(5))
            client.shutdown(socket.SHUT_RDWR);client.close()
            check('旧策略 '+policy+':未采用文本立即取消',not state['closed'].wait(.55))
            if policy=='explicit':check('旧策略显式总时限保留',state['closed'].wait(1.5))
            else:state['release'].set()
        finally:state['release'].set();stop(p,log)
    for endpoint in ['/v1/messages','/v1/chat/completions','/v1/responses']:
        state['mode']='conflict'
        before=len(provider_requests)
        p,port,_,log=start({},endpoint.rsplit('/',1)[-1]+'-conflict')
        try:
            model='gpt-4.1' if args.provider=='openai' else 'claude-sonnet-4-6'
            body={'model':model,'stream':True,'max_tokens':64,'messages':[{'role':'user','content':'SYNTHETIC'}]}
            if endpoint.endswith('/responses'):body={'model':model,'stream':True,'input':'SYNTHETIC'}
            client=http.client.HTTPConnection('127.0.0.1',port,timeout=10)
            client.request('POST',endpoint,json.dumps(body),{'Content-Type':'application/json'})
            response=client.getresponse();raw=response.read().decode();client.close()
            error=json.loads(raw).get('error',{})
            check(endpoint+':结构化冲突禁止重试',response.status==409 and error.get('code')=='execution_state_unknown' and error.get('details',{}).get('retryable') is False and len(provider_requests)==before+1)
            check(endpoint+':冲突不泄露上游详情','PRIVATE_PROVIDER_MESSAGE' not in raw and 'DO_NOT_EXPOSE' not in raw)
        finally:stop(p,log)
    passed=True
except Exception:
    (args.out/'failure.txt').write_text(traceback.format_exc())
finally:
    state['release'].set();provider.shutdown();provider.server_close()
    (args.out/'summary.json').write_text(json.dumps({'result':'PASS' if passed else 'FAIL','checks':checks,'provider_requests':provider_requests,'real_provider':False,'scope_notes':['基线 OpenAI Responses 透传未实现数据间隔计时器，未把此既有缺口误记为通过。'] if args.provider=='openai' else []},ensure_ascii=False,indent=2))
print('PASS' if passed else 'FAIL')
raise SystemExit(0 if passed else 1)
