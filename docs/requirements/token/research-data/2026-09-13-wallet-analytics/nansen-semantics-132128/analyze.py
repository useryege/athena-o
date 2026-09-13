"""Offline analysis of saved evidence. Makes no network calls."""
import json, math
from pathlib import Path
P=Path(__file__).resolve().parent
OLD=P.parent/'nansen-integration-sample-130328'
def read(name, folder=P):return json.loads((folder/(name+'.json')).read_text())
def rows(name, folder=P):return read(name,folder)['response']['data']
def ids(rs):return [r['token_address'] for r in rs]
def hashes(rs):return [r['transaction_hash'] for r in rs]
requests=[]
for b in sorted(P.glob('batch*.json')):
 for item in json.loads(b.read_text()):
  e=read(item['name']);assert e['request']['body']==item.get('body')
  assert e['http_status']==200
  requests.append({'file':item['name']+'.json','checked_at_utc':e['checked_at_utc'],'http_status':e['http_status'],'credits_used':int(e['response_headers']['x-nansen-credits-used']),'credits_remaining':int(e['response_headers']['x-nansen-credits-remaining']),'request_id':e['response_headers'].get('x-request-id'),'records':len(e['response'].get('data',[])),'pagination':e['response'].get('pagination')})
requests.sort(key=lambda r:r['checked_at_utc'])
original=rows('transactions',OLD);second=rows('transactions-page2')
alltx=original+second
true=rows('active-pnl-true');false=rows('active-pnl-false');asc=rows('active-pnl-true-asc');falseasc=rows('active-pnl-false-asc')
assert ids(true)==ids(rows('active-pnl-true-page1-size2')+rows('active-pnl-true-page2-size2'))
assert ids(asc)==list(reversed(ids(true)))
assert len(set(hashes(alltx)))==len(alltx)
assert alltx==sorted(alltx,key=lambda r:r['block_timestamp'],reverse=True)
for name,token in [('pnl-cate-filter','0x712f3378cd1a0476b53f0e29b1a9586e00d78683')]:assert all(r['token_address']==token for r in rows(name))
for name,token in [('transactions-cate-filter','0x712f3378cd1a0476b53f0e29b1a9586e00d78683'),('transactions-usdt-filter','0xdac17f958d2ee523a2206206994597c13d831ec7')]:
 assert all(any(t['token_address']==token for t in (r.get('tokens_sent') or [])+(r.get('tokens_received') or [])) for r in rows(name))
assert not set(hashes(rows('transactions-cate-filter')))&set(hashes(original))
checks={'original_transaction_pages':{'rows':len(alltx),'unique_hashes':len(set(hashes(alltx))),'descending_across_boundary':True,'complete_history':False},'exact_filters':{'pnl_cate_count':len(rows('pnl-cate-filter')),'transactions_cate_count':len(rows('transactions-cate-filter')),'transactions_usdt_count':len(rows('transactions-usdt-filter')),'cate_found_beyond_original_first_page':True},'active_pnl':{'true_symbols':[r['token_symbol'] for r in true],'false_symbols':[r['token_symbol'] for r in false],'true_asc_is_reverse':True,'true_size2_pages_reconstruct_full_sample':True,'false_desc_observed_values':[r['pnl_usd_realised'] for r in false],'false_asc_observed_values':[r['pnl_usd_realised'] for r in falseasc],'false_desc_is_sorted_as_requested':all(false[i]['pnl_usd_realised']>=false[i+1]['pnl_usd_realised'] for i in range(len(false)-1)),'false_asc_desc_same_order':ids(false)==ids(falseasc)},'summary_diagnostics':[],'date_windows':[]}
for label,sname,pname,folder in [('original30','pnl-summary','pnl-show-realized-false',OLD),('active30','active-summary','active-pnl-true',P),('active90','active-summary-90d','active-pnl-90d',P)]:
 s=read(sname,folder)['response'];rs=rows(pname,folder)
 checks['summary_diagnostics'].append({'sample':label,'win_rate_raw':s['win_rate'],'traded_token_count':s['traded_token_count'],'traded_times':s['traded_times'],'sum_nof_buys':sum(int(r['nof_buys']) for r in rs),'sum_nof_sells':sum(int(r['nof_sells']) for r in rs),'positive_realized_tokens':sum(r['pnl_usd_realised']>0 for r in rs),'positive_total_pnl_tokens':sum(r['pnl_usd_realised']+r['pnl_usd_unrealised']>0 for r in rs),'rows':len(rs),'unit_cost_relation_matches':all(math.isclose(r['holding_usd']-r['holding_amount']*r['cost_basis_usd'],r['pnl_usd_unrealised'],abs_tol=1e-6,rel_tol=1e-9) for r in rs),'interpretation':'Correlations only; no inferred win-rate formula or reimplemented PnL.'})
for n in ['transactions-7d','transactions-90d']:
 e=read(n);dates=e['request']['body']['date'];rs=e['response']['data']
 inrange=all(dates['from']<=r['block_timestamp'].removesuffix('Z')+'Z'<=dates['to'] for r in rs)
 assert inrange
 checks['date_windows'].append({'file':n+'.json','records':len(rs),'within_requested_window_if_naive_timestamps_are_UTC':inrange})
dex=rows('active-dex-trades');common=set(hashes(dex))&set(hashes(rows('active-transactions')))
tmap={r['transaction_hash']:r for r in rows('active-transactions')}
checks['dex_crosscheck']={'dex_records':len(dex),'unique_hashes':len(set(hashes(dex))),'common_hashes':len(common),'all_common_general_records_marked_transfer':all(tmap[h]['source_type']=='transfer' for h in common),'timestamps_match_with_UTC_suffix':all(r['block_timestamp']==tmap[r['transaction_hash']]['block_timestamp']+'Z' for r in dex if r['transaction_hash'] in common),'sent_amount_signs':sorted(set(-1 if t['token_amount']<0 else 1 if t['token_amount']>0 else 0 for r in rows('active-transactions') for t in r.get('tokens_sent') or []))}
checks['timestamp_independent_rpc']=read('ethereum-timestamp-crosscheck')['matches_if_nansen_is_utc']
result={'scope':'Research observations, not ATHENA integration acceptance','total_nansen_requests':len(requests),'total_credits_used':sum(r['credits_used'] for r in requests),'last_reported_credits_remaining':requests[-1]['credits_remaining'],'requests':requests,'checks':checks,'provider_contract_gap':'show_realized=false did not honor requested realized PnL DESC for the active sample.'}
(P/'analysis.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
print(json.dumps({k:v for k,v in result.items() if k!='requests'},ensure_ascii=False,indent=2))
