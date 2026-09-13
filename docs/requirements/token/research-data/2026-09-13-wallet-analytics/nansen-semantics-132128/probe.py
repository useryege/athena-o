"""Explicit, bounded Nansen research probes; no retries or application changes.
Run from repository root: python3 <this-file> <batch.json>
Each batch item: name, endpoint (profiler/address/*), body, purpose.
"""
import datetime, json, os, re, sys, time, urllib.request, urllib.error
from pathlib import Path
ROOT = Path.cwd()
OUT = Path(__file__).resolve().parent
key_text = (ROOT / 'docs/requirements/token/wallet-trading-performance.md').read_text()
os.environ['NANSEN_API_KEY'] = re.search(r'^NANSEN_API_KEY=(nsn_[a-zA-Z0-9]+)$', key_text, re.M).group(1)
class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs): return None
opener = urllib.request.build_opener(NoRedirect)
for item in json.loads(Path(sys.argv[1]).read_text()):
    dest = OUT / (item['name'] + '.json')
    if dest.exists(): raise SystemExit('Refusing to overwrite evidence: '+str(dest))
    endpoint = item['endpoint']
    if endpoint not in ['profiler/address/pnl', 'profiler/address/pnl-summary', 'profiler/address/transactions', 'tgm/pnl-leaderboard', 'profiler/dex-trades', 'account']:
        raise SystemExit('Endpoint not allowed by this research script')
    url = 'https://api.nansen.ai/api/v1/' + endpoint
    body = item.get('body')
    method = 'GET' if endpoint == 'account' else 'POST'
    req = urllib.request.Request(url, data=json.dumps(body).encode() if body is not None else None, method=method, headers={'apikey':os.environ['NANSEN_API_KEY'], 'Content-Type':'application/json', 'Accept':'application/json', 'User-Agent':'ATHENA-Nansen-Requirements-Probe/1.0'})
    started = datetime.datetime.now(datetime.timezone.utc).isoformat()
    ts = time.monotonic()
    try:
        reply = opener.open(req, timeout=45)
    except urllib.error.HTTPError as e:
        reply = e
    except (urllib.error.URLError, TimeoutError) as e:
        dest.write_text(json.dumps({'checked_at_utc':started,'purpose':item['purpose'],'request':{'method':method,'url':url,'body':body},'transport_error':str(e),'billing_outcome':'unknown; do not assume uncharged'},ensure_ascii=False,indent=2)+'\n')
        raise SystemExit('Transport error; evidence saved; no retry')
    with reply:
        status = reply.status if hasattr(reply, 'status') else reply.code
        headers = {k.lower():v for k,v in reply.headers.items()}
        raw = reply.read().decode()
    try: response = json.loads(raw)
    except ValueError: response = raw
    data = {'checked_at_utc':started,'purpose':item['purpose'],'request':{'method':method,'url':url,'body':body},'http_status':status,'elapsed_seconds':round(time.monotonic()-ts,3),'response_headers':headers,'response':response}
    with dest.open('x') as f: f.write(json.dumps(data,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'name':item['name'],'status':status,'used':headers.get('x-nansen-credits-used'),'remaining':headers.get('x-nansen-credits-remaining'),'count':len(response.get('data',[])) if isinstance(response,dict) else None,'pagination':response.get('pagination') if isinstance(response,dict) else None,'error':response if status != 200 else None},ensure_ascii=False),flush=True)
    if status != 200: raise SystemExit('Non-success response; stopped without retry')
