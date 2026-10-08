import json,subprocess,unittest
SOURCE='''package binding
import "testing"
func TestValue(t *testing.T) {
 rows := []struct { input string; expected uint64 }{{"日本語", 18446744073709551615}}
 for _, row := range rows {
  got, err := Unknown(row.input)
  if err != nil { t.Errorf("error"); continue }
  if got != row.expected { t.Errorf("mismatch") }
 }
}
'''
def bind(s):
 return json.loads(subprocess.check_output(['/tmp/issue155-error-guard-binding'],input=json.dumps({'files':[dict(id='F1',path='value_test.go',before=s,after=s)]}).encode()))
class ErrorGuardTests(unittest.TestCase):
 def test_guard_and_exact_uint64_literal_without_evaluation(self):
  rows=bind(SOURCE);self.assertEqual(len(rows),2)
  for b in rows:
   self.assertEqual(b['Expected']['text'],'18446744073709551615');self.assertEqual(b['Guard']['text'],'err != nil');self.assertEqual(b['Continue']['text'],'continue')
   for k in ('Row','Input','Expected','Call','Condition','Guard','Continue'):
    a,z=b[k]['span'];self.assertEqual(SOURCE.encode()[a:z].decode(),b[k]['text'])
 def test_wrong_guard_missing_continue_or_mutation_are_rejected(self):
  for s in (SOURCE.replace('err != nil','got != nil'),SOURCE.replace('; continue',''),SOURCE.replace('if got != row.expected','got = 1; if got != row.expected'),SOURCE.replace('"testing"','"untrusted/testing"'),SOURCE.replace('func TestValue', 'var nil = 0\nfunc TestValue')):
   self.assertEqual(bind(s),[])
if __name__=='__main__':unittest.main()
