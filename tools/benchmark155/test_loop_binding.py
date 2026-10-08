import json,subprocess,unittest
SOURCE='''package binding
import "testing"
func check(t testing.TB) {
 cases := [][]string{{"日本語", "ok"}}
 for _, row := range cases {
  input := row[0]
  expected := row[1]
  result := Unknown(input)
  if result != expected { t.Errorf("mismatch") }
 }
}
'''
def bind(s):
 return json.loads(subprocess.check_output(['/tmp/issue155-loop-binding'],input=json.dumps({'files':[dict(id='F1',path='loop_test.go',before=s,after=s)]}).encode()))
class LoopBindingTests(unittest.TestCase):
 def test_source_columns_and_unicode_without_evaluating_unknown_call(self):
  rows=bind(SOURCE);self.assertEqual(len(rows),2)
  for b in rows:
   self.assertEqual(b['Input']['text'],'"日本語"');self.assertEqual(b['Expected']['text'],'"ok"');self.assertEqual(b['Call']['text'],'Unknown(input)')
   for k in ('Row','Input','Expected','Call','Condition'):
    a,z=b[k]['span'];self.assertEqual(SOURCE.encode()[a:z].decode(),b[k]['text'])
 def test_mutation_extra_control_and_false_testing_do_not_bind(self):
  for s in (SOURCE.replace('expected := row[1]','expected = row[1]'),SOURCE.replace('result := Unknown(input)','if true { input = "other" }; result := Unknown(input)'),SOURCE.replace('"testing"','"untrusted/testing"')):
   self.assertEqual(bind(s),[])
if __name__=='__main__':unittest.main()
