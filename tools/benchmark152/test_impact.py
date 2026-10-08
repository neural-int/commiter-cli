"""Counterfactual pilot on synthetic Go only; no gold used by probes."""
import argparse,json,os,pathlib,subprocess,tempfile,time,sys,re,itertools
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parents[1]/'benchmark151'))
from inline import extract
from evaluate import reconstruct


def main():
 p=argparse.ArgumentParser();p.add_argument('--output',required=True);p.add_argument('--fixtures',type=pathlib.Path);p.add_argument('--pairs',action='store_true');a=p.parse_args()
 records=json.loads((a.fixtures or pathlib.Path(__file__).with_name('inline-fixtures.json')).read_text());rows=[]
 for r in records:
  files=r['Files'];units={};all_ids=[]
  for f in files:
   units[f['ID']]=extract(f['Before'].encode(),f['After'].encode(),f['ID'])
   all_ids += [u['id'] for u in units[f['ID']]]
  if len(all_ids)>16: raise ValueError('probe_budget')
  probes=[('baseline',[])]+[(uid,[uid]) for uid in all_ids]+[('all',all_ids)]
  if a.pairs:
   if len(all_ids)>8: raise ValueError('pair_probe_budget')
   probes += [('+'.join(pair),list(pair)) for pair in itertools.combinations(all_ids,2) if len(pair)<len(all_ids)]
  for label,selected in probes:
   with tempfile.TemporaryDirectory(prefix='benchmark152-impact-') as tmp:
    root=pathlib.Path(tmp);(root/'go.mod').write_text('module fixture\n\ngo 1.23.0\n')
    for f in files:
     dst=root/f['Path'];dst.parent.mkdir(parents=True,exist_ok=True)
     dst.write_bytes(reconstruct(f['Before'].encode(),units[f['ID']],[uid for uid in selected if uid in {u['id'] for u in units[f['ID']]}]))
    for f in r['Repository']:
     dst=root/f['Path'];dst.parent.mkdir(parents=True,exist_ok=True);dst.write_text(f['Content'])
    env=dict(os.environ,GOCACHE='/tmp/issue151-remeasure-gocache',GOWORK='off',GOTOOLCHAIN='local',GOPROXY='off')
    start=time.perf_counter()
    result=subprocess.run(['go','test','-json','-p','1','./...'],cwd=root,env=env,capture_output=True,timeout=90)
    events=[]
    for line in result.stdout.splitlines():
     try: events.append(json.loads(line))
     except ValueError: pass
    failures=sorted({e['Package']+'/'+e['Test'] for e in events if e.get('Action')=='fail' and e.get('Test')})
    locations=sorted({(e.get('Package',''),e.get('Test',''),match.group(1),int(match.group(2))) for e in events if e.get('Test') and e.get('Action')=='output' for match in re.finditer(r'(?m)^\s+([^\s:]+\.go):(\d+):',e.get('Output',''))})
    rows.append(dict(fixture=r['Name'],probe=label,selected_unit_ids=selected,returncode=result.returncode,failed_tests=failures,assertion_locations=locations,unknown=result.returncode!=0 and not failures,wall_seconds=time.perf_counter()-start))
    print(r['Name'],label,failures,flush=True)
 pathlib.Path(a.output).write_text(json.dumps(rows,indent=2)+'\n')

if __name__=='__main__':main()
