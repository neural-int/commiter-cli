import hashlib,importlib.util,json,pathlib,re
root=pathlib.Path('$HOME/.codex/worktrees/issue-182-model-comparison/commiter-cli')
p=pathlib.Path('/private/tmp/issue182-overlap-ab');spec=importlib.util.spec_from_file_location('h',root/'tools/benchmark182/prompt_compare.py');h=importlib.util.module_from_spec(spec);spec.loader.exec_module(h)
contract=json.loads((p/'contract.json').read_text());rows=json.loads((p/'results.json').read_text());completion=json.loads((p/'completion.json').read_text());assert len(rows)==completion['cells']==8 and not completion['monitor_failed']
for n,d in contract['hashes'].items():assert h.sha(n)==d,n
out=[];audited=0
for row in rows:
 c=next(c for c in contract['cells'] if c['label']==row['cell']);folder=p/c['label'];phases=[]
 for i in [1,2,3]:
  req=json.loads((folder/f'{i:02d}-request.json').read_text());wire=json.loads((folder/f'{i:02d}-actual-request.json').read_text());wire.pop('model_path',None);assert wire==h.transform(req,c['arm']);h.invariant(req,wire);audited+=1
  reply=json.loads((folder/f'{i:02d}-response.json').read_text());native=reply['native'];assert native['ok'] and native['stop_reason']=='completed'
  m=native['benchmark_metrics'];assert m['nativeThoughtTokens']<=1024
  phases.append({'content':reply['response']['Content'],'tokens':m['generatedTokens'],'thinking':m['nativeThoughtTokens'],'forced':m['nativeThoughtForcedClose'],'decode_seconds':m['runtimeGenerationSeconds']})
 assert not row['result']['error'];groups=json.loads(phases[0]['content']);assert len(set(groups.values()))==1 and set(groups)==set(row['result']['fixed_membership'])
 out.append({'cell':c['label'],'wall_seconds':row['result']['wall_seconds'],'tokens':sum(v['tokens'] for v in phases),'tps':sum(v['tokens'] for v in phases)/sum(v['decode_seconds'] for v in phases),'phases':phases})
pairs=[]
for a in out:
 if not a['cell'].endswith('-A'):continue
 b=next(b for b in out if b['cell']==a['cell'][:-1]+'B')
 assert all(all(x[k]==y[k] for k in ['content','tokens','thinking','forced']) for x,y in zip(a['phases'],b['phases']))
 pairs.append({'case':a['cell'][:-2],'A':a,'B':b,'tps_change_percent':100*(b['tps']/a['tps']-1)})
monitor=[json.loads(line) for line in (p/'monitor.jsonl').read_text().splitlines()];assert all(s['valid'] for s in monitor)
summary={'id':'H182-007','completion':completion,'audited_requests':audited,'monitor_samples':len(monitor),'all_pairs_outputs_and_counts_equal':True,'pairs':pairs};h.save(p/'summary.json',summary)
goal=root/'docs/benchmarks/issue-182/overlap-ab/goal.md';s=goal.read_text().replace('（A/B測定中）','（測定完了）',1).replace('生成計測はまだ行っていない。','この節は準備時点の記録。測定結果は末尾に記載。',1)
s+='\n## H182-007 最終結果\n\n全8セルが自然完了、全24phaseがcompleted、有効JSONとplan受理・実Grouping基準一致を確認。全4組で応答内容・生成tokens・Thinking数・強制遷移が一致した。\n\n| ケース | A token/s | B token/s | 生成速度差 | A cycle秒 | B cycle秒 |\n| --- | ---: | ---: | ---: | ---: | ---: |\n'
for v in pairs:
 name=('Gemma' if v['case'].startswith('gemma') else 'Qwen4B')+' '+('C01' if '-C01-' in v['case'] else 'C02');a=v['A'];b=v['B'];s+=f"| {name} | {a['tps']:.2f} | {b['tps']:.2f} | {v['tps_change_percent']:+.1f}% | {a['wall_seconds']:.2f} | {b['wall_seconds']:.2f} |\n"
s+=f"\n全{audited}実送信要求の不変条件と全契約hashを実行後にも確認。OS監視{len(monitor)}件すべて有効、monitor_failed=false、driver全体{completion['seconds']:.2f}秒。生成token/sはThinkingを含む全phase生成tokens合計/生成区間秒合計で、入力処理時間を含めない。\n"
s+='\nBは3組で高速、Qwen C01では約1.1%低速。単発測定であり、一貫した改善や有意差は未確認。GPU/CPUが実際に重なった時間の直接profilingも行っていないため、同期位置だけの因果的効果やハードウェア上限は断定しない。30〜70 token/sには届いていない。Bの採用は保留。\n\n自然終了と予算到達の遷移はA/B一致。Gemmaは全phase自然終了、Qwen C01 CategoryとC02 Grouping/Categoryが1024で強制遷移後JSON回答を生成。Qwen C02のscope internal/gitstatなど既存の意味品質上の問題も同じで、形式受理と品質を区別する。後段固定partitionのためend-to-end成功率ではない。\n\n今回の測定・集計は完了。追加推論・再試行・条件変更は行わず、原本を保持。raw/に実行ログ・応答・監視・summaryを追加保存した。\n';goal.write_text(s)
target=root/'docs/benchmarks/issue-182/overlap-ab/raw';email=re.compile(r'[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}');home=re.compile(r'$HOME/\s"\\]+')
for f in p.rglob('*'):
 if not f.is_file() or 'runtime' in f.relative_to(p).parts:continue
 text=home.sub('$HOME',email.sub('redacted@example.invalid',f.read_text()));assert not re.search(r'(?i)(?:gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|sk-[A-Za-z0-9]{24,}|-----BEGIN (?:RSA |OPENSSH |EC )?PRIVATE KEY-----)',text)
 dest=target/f.relative_to(p);dest.parent.mkdir(parents=True,exist_ok=True);dest.write_text(text)
checks=[hashlib.sha256(f.read_bytes()).hexdigest()+'  '+str(f.relative_to(target)) for f in sorted(target.rglob('*')) if f.is_file() and f.name!='SHA256SUMS.txt'];(target/'SHA256SUMS.txt').write_text('\n'.join(checks)+'\n')
print('complete',len(pairs),'pairs',audited,'requests',len(monitor),'valid monitor samples')
