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
