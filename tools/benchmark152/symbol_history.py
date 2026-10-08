"""Bounded snapshot symbol touches; co-change is not shared intent."""
import json,subprocess,hashlib
MAX_BYTES=1048576

def symbols(source,helper):
 if len(source)>MAX_BYTES: return None
 p=subprocess.run([helper],input=source,capture_output=True)
 if p.returncode: return None
 try: rows=json.loads(p.stdout)
 except (ValueError,TypeError): return None
 if len(rows)>256: return None
 result={}
 for s in rows:
  name=s['name']
  if name in result: return None
  result[name]=hashlib.sha256(source[s['start']:s['end']]).hexdigest()
 return result

def touches(before,after,helper):
 if before is None or after is None: return dict(status='unknown_snapshot_boundary',symbols=[])
 a,b=symbols(before,helper),symbols(after,helper)
 if a is None or b is None: return dict(status='unknown_parse_or_budget',symbols=[])
 return dict(status='observed_symbol_snapshot',symbols=sorted(n for n in set(a)|set(b) if a.get(n)!=b.get(n)))

def summarize(events,current):
 """Fixed observed window; missing observations never imply independence."""
 import itertools
 if len(events)>64 or len(current)>64: raise ValueError('history_budget')
 current=set(current);counts={x:0 for x in sorted(current)};together={};separate={};unknown=0
 for event in events:
  if event['status']!='observed_symbol_snapshot': unknown+=1;continue
  touched=set(event['symbols'])&current
  for x in touched: counts[x]+=1
  for a,b in itertools.combinations(sorted(current),2):
   if a in touched and b in touched: together[a,b]=together.get((a,b),0)+1
   elif (a in touched)!=(b in touched): separate[a,b]=separate.get((a,b),0)+1
 return dict(window=len(events),unknown_events=unknown,touches=counts,
  relations=[dict(symbols=[a,b],cochange=together.get((a,b),0),separate_touches=separate.get((a,b),0),status='observed_soft_history' if counts[a] and counts[b] else 'unknown_sparse_history') for a,b in itertools.combinations(sorted(current),2)],semantics='historical_observation_not_intent; absence_unknown_no_default_merge')
