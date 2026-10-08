import json,subprocess,unittest
SOURCE='''package binding
import "testing"
func check(t testing.TB) {
 cases := [][]string{{"日本語", "ok"}, {"b", "out", "é"}}
 for _, row := range cases {
  input := row[0]
  expected := row[1]
  var extra string
  extra = ""
  if len(row) == 3 { extra = row[2] }
  result := Unknown(input, extra)
  if result != expected {
   message := ""
   if len(row) == 3 { message = " extra " + row[2] }
   t.Errorf("mismatch", message)
  }
 }
}
'''
def bind(s):
 return json.loads(subprocess.check_output(['/tmp/issue155-column-guard-binding'],input=json.dumps({'files':[dict(id='F1',path='value_test.go',before=s,after=s)]}).encode()))
class ColumnGuardTests(unittest.TestCase):
 def test_string_default_and_conditional_literal(self):
  rows=bind(SOURCE);self.assertEqual(len(rows),4);self.assertEqual([r['SelectedArgument'] for r in rows],['','é','','é'])
  for b in rows:
   for k in ('Row','Input','Expected','Call','Condition','Guard','ArgumentDefinition','ArgumentAssignment'):
    a,z=b[k]['span'];self.assertEqual(SOURCE.encode()[a:z].decode(),b[k]['text'])
 def test_byte_default_and_utf8_first_byte_not_codepoint(self):
  s=SOURCE.replace('var extra string\n  extra = ""','var extra uint8').replace('extra = row[2]','extra = row[2][0]')
  self.assertEqual([b['SelectedArgument'] for b in bind(s)],['0','195','0','195'])
  self.assertEqual(bind(s.replace('"é"','""')),[])
 def test_wrong_guard_or_diagnostic_mutation_do_not_bind(self):
  for s in (SOURCE.replace('len(row) == 3','len(row) == 2'),SOURCE.replace('message = " extra " + row[2]','result = "other"'),SOURCE.replace('t.Errorf','other.Errorf'),SOURCE.replace('import "testing"','import "untrusted/testing"')):
   self.assertEqual(bind(s),[])
if __name__=='__main__':unittest.main()
