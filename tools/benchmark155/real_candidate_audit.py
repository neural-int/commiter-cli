import hashlib,json,pathlib,subprocess
root=pathlib.Path(__file__).resolve().parents[2]
sha=lambda b:hashlib.sha256(b).hexdigest()
def git(*args):return subprocess.check_output(['git',*args],cwd=root)
rows=[]
corpus=[p for base in ('docs/benchmarks','tools/benchmark149','tools/benchmark151','tools/benchmark152','tools/benchmark153') for p in (root/base).rglob('*') if p.is_file() and p.suffix in ('.json','.jsonl','.py','.go','.md')]
for commit,paths in [('8092acd2',['internal/relation/graph.go','internal/relation/graph_test.go']),('c643154b',['internal/relation/extract.go','internal/relation/extract_test.go'])]:
 full=git('rev-parse',commit).decode().strip();parent=git('rev-parse',commit+'^').decode().strip()
 files=[dict(id=f'F{i+1:03}',path=p,before=git('show',parent+':'+p).decode(),after=git('show',full+':'+p).decode()) for i,p in enumerate(paths)]
 payload=json.dumps({'files':files},sort_keys=True).encode()
 facts=json.loads(subprocess.run(['/tmp/issue155-anchor-observer'],input=payload,capture_output=True,check=True,timeout=10).stdout)
 # A reference scan is only provenance triage, not proof of prior model-input non-overlap.
 hits=[str(p.relative_to(root)) for p in corpus if commit.encode() in p.read_bytes()]
 changed=[e for e in facts['evidence'] if e['kind']=='failure_predicate' and e['changed']]
 rows.append(dict(commit=full,parent=parent,paths=paths,input_sha256=sha(payload),input_bytes=len(payload),
  snapshot_sha256=[dict(file=f['id'],before=sha(f['before'].encode()),after=sha(f['after'].encode())) for f in files],
  prior_commit_reference_hits=hits,changed_failure_predicates=changed,
  disposition='provisional candidate only; no holdout certification or gold yet',model_calls=0))
out=root/'docs/benchmarks/issue-155/iteration-1-real-candidates.json'
with out.open('x') as s:json.dump(rows,s,ensure_ascii=False,indent=2);s.write('\n')
print(json.dumps([dict(commit=r['commit'],input_bytes=r['input_bytes'],reference_hits=len(r['prior_commit_reference_hits']),changed_predicates=len(r['changed_failure_predicates'])) for r in rows],indent=2))
