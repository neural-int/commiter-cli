import pathlib,tempfile,subprocess,unittest
from history_scan import scan
class Scanner(unittest.TestCase):
 def test_real_commits_rename_missing_and_determinism(self):
  with tempfile.TemporaryDirectory() as tmp:
   p=pathlib.Path(tmp)
   def git(*args):subprocess.run(['git',*args],cwd=p,check=True,capture_output=True)
   git('init','-q');git('config','user.name','Benchmark');git('config','user.email','benchmark@example.invalid')
   f=p/'value.go';f.write_text('package p\nfunc Left() int { return 2 }\nfunc Right() int { return 3 }\n');git('add','.');git('commit','-qm','initial')
   f.write_text(f.read_text().replace('return 2','return 4'));git('add','.');git('commit','-qm','left')
   a=scan(p,[('value.go','Left'),('value.go','Right')],'/tmp/issue151-remeasure-symbols')
   self.assertEqual(a,scan(p,[('value.go','Left'),('value.go','Right')],'/tmp/issue151-remeasure-symbols'))
   self.assertEqual(a['summary']['touches']['value.go::Left'],1)
   git('mv','value.go','moved.go');git('commit','-qm','rename')
   b=scan(p,[('moved.go','Left')],'/tmp/issue151-remeasure-symbols')
   self.assertEqual(b['events'][0]['status'],'unknown_rename')
   self.assertTrue(all(x['status'].startswith('unknown') for x in b['events']))

 def test_real_merge_is_unknown(self):
  with tempfile.TemporaryDirectory() as tmp:
   p=pathlib.Path(tmp)
   def git(*args):return subprocess.check_output(['git',*args],cwd=p,stderr=subprocess.DEVNULL)
   git('init','-q');git('config','user.name','Benchmark');git('config','user.email','benchmark@example.invalid')
   (p/'value.go').write_text('package p\nfunc Left() int { return 2 }\n');git('add','.');git('commit','-qm','initial')
   original=git('branch','--show-current').decode().strip();git('checkout','-qb','side')
   (p/'side.txt').write_text('side');git('add','.');git('commit','-qm','side')
   git('checkout',original);(p/'main.txt').write_text('main');git('add','.');git('commit','-qm','main');git('merge','--no-ff','side','-m','merge')
   r=scan(p,[('value.go','Left')],'/tmp/issue151-remeasure-symbols')
   self.assertEqual(r['events'][0]['status'],'unknown_merge')
