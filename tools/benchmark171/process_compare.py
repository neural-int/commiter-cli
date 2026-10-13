"""Isolated, user-authorized run; reuse all original #171 contracts unchanged."""
import argparse,importlib.util,json,sys
from pathlib import Path
ROOT=Path(__file__).resolve().parents[2]
def module(name,path):
 spec=importlib.util.spec_from_file_location(name,path);value=importlib.util.module_from_spec(spec);spec.loader.exec_module(value);return value

def main():
 p=argparse.ArgumentParser();p.add_argument('mode',choices=['validate','run','evaluate']);p.add_argument('--data',type=Path,required=True);p.add_argument('--runtime',type=Path,required=True);a=p.parse_args();data=a.data.resolve()
 if not data.is_relative_to(ROOT/'docs/benchmarks/issue-171') or data==ROOT/'docs/benchmarks/issue-171':raise ValueError('separate research run required')
 r=module('runner171_repeat',Path(__file__).with_name('run_process_memory.py'));r.DATA=data;local=json.loads(a.runtime.read_text());pr=json.loads((data/'preregistration.json').read_text());r.prior.check_pins(pr,local)
 if a.mode=='validate':
  r.check_model(pr,local,full=True)
  rows=json.loads((data/'frozen-inputs/manifest.json').read_text());assert len(rows)==96 and all(x['admitted'] for x in rows);assert pr['max_generation_calls']==96 and pr['retries']==0
  for x in rows:
   root=Path(local['fixture_roots'][x['fixture']]);assert r.prior.repo_digest(root)==x['repository_sha256']
  print(json.dumps({'frozen_contracts':'PASS','fixture_Git':'PASS','model_artifacts':'PASS','requests':96,'generation_calls':0}));return
 if a.mode=='evaluate':
  e=module('evaluate171_repeat',Path(__file__).with_name('evaluate_process_memory.py'));e.DATA=data;e.main();return
 if (data/'termination.json').exists() or (data/'attempts.jsonl').exists():raise ValueError('run already attempted or terminal; never restart/replay')
 sys.argv=['run171','--runtime',str(a.runtime)]
 try:r.main()
 except BaseException as exc:
  used=len((data/'attempts.jsonl').read_text().splitlines()) if (data/'attempts.jsonl').exists() else 0
  (data/'termination.json').write_text(json.dumps({'reason':str(exc),'calls_used':used,'phase1_completed':False,'further_generation_permitted_in_this_run':False},indent=2)+'\n');raise
 used=len((data/'attempts.jsonl').read_text().splitlines())
 (data/'termination.json').write_text(json.dumps({'reason':'fixed_cell_matrix_completed','calls_used':used,'phase1_completed':used==96,'further_generation_permitted_in_this_run':False},indent=2)+'\n')
if __name__=='__main__':main()
