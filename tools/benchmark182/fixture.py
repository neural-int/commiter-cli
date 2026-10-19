"""Materialize an authored public 16-file/four-purpose study; never read user repos."""
import argparse, hashlib, json, os, pathlib, subprocess

def materialize(destination):
    destination.mkdir(parents=True, exist_ok=False)
    repo=destination/'repo';repo.mkdir()
    def git(*args):return subprocess.check_output(['git',*args],cwd=repo,env=os.environ|{'GIT_AUTHOR_DATE':'2026-10-19T00:00:00Z','GIT_COMMITTER_DATE':'2026-10-19T00:00:00Z'},stderr=subprocess.DEVNULL)
    git('init','-b','codex/public-fixture');git('config','user.name','Public Fixture');git('config','user.email','fixture@example.invalid')
    cases=[('expiry','Expired','now, deadline int','now > deadline','now >= deadline','Expired(10, 10)','false','true','有効期限と同時刻のキャッシュを期限切れにする'),
      ('pagination','Limit','requested, maximum int','requested','min(requested, maximum)','Limit(25, 10)','25','10','ページ件数を上限に制限する'),
      ('retry','CanRetry','attempt, maximum int','attempt <= maximum','attempt < maximum','CanRetry(3, 3)','true','false','最大試行回数に達した再試行を停止する'),
      ('httpstatus','IsSuccess','status int','status >= 200 && status <= 300','status >= 200 && status < 300','IsSuccess(300)','true','false','HTTP 300を成功扱いしない')]
    after={};groups=[];contexts=[]
    for name,fn,args,old,new,call,before,expected,purpose in cases:
        ret='int' if name=='pagination' else 'bool'
        paths=[f'{name}/{fn.lower()}.go',f'{name}/{fn.lower()}_test.go',f'{name}/README.md',f'{name}/examples.json']
        contents=[(f'package {name}\n\nfunc {fn}({args}) {ret} {{ return {old} }}\n', f'package {name}\n\nfunc {fn}({args}) {ret} {{ return {new} }}\n'),
          (f'package {name}\n\nimport "testing"\n\nfunc TestBoundary(t *testing.T) {{ if got := {call}; got != {before} {{ t.Fatalf("boundary: %v", got) }} }}\n',f'package {name}\n\nimport "testing"\n\nfunc TestBoundary(t *testing.T) {{ if got := {call}; got != {expected} {{ t.Fatalf("boundary: %v", got) }} }}\n'),
          (f'# {name}\nBoundary example: `{call}` returns `{before}`.\n',f'# {name}\nBoundary example: `{call}` returns `{expected}`.\n{purpose}。\n'),
          (json.dumps({'call':call,'expected':json.loads(before)})+'\n',json.dumps({'call':call,'expected':json.loads(expected)})+'\n')]
        for path,(prior,later) in zip(paths,contents):
            target=repo/path;target.parent.mkdir(exist_ok=True);target.write_text(prior);after[path]=later
        groups.append({'files':paths,'type':'fix','purpose':purpose,'breaking':False})
        contexts.append({'paths':paths,'user_request':purpose+'。実装・境界テスト・説明・例を一緒に更新する。','final_response':'対応する境界条件とテスト・説明・例を修正した。'})
    (repo/'go.mod').write_text('module public-fixture\n\ngo 1.24\n')
    git('add','.');git('commit','-m','public fixture base')
    for path,content in after.items():(repo/path).write_text(content)
    result={'case_id':'C16','data_classification':'authored_synthetic_public_fixture','reference_human_approved':False,'groups':groups,'codex_context':contexts,'scoring':'exact file-set commit recall (matched exact groups / 4); category accuracy on exact matched groups; end-to-end validator acceptance; summaries manually reviewed against purpose; no model judge','limitations':'one simple authored fixture; no held-out set or production-quality claim','files_sha256':{p:hashlib.sha256((repo/p).read_bytes()).hexdigest() for p in sorted(after)}}
    (destination/'fixture.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    subprocess.run(['go','test','./...'],cwd=repo,check=True)
    return result

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('destination',type=pathlib.Path);a=p.parse_args();materialize(a.destination)
