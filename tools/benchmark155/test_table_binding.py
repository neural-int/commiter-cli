import json,subprocess,unittest

SOURCE='''package binding
import "testing"
type rows []struct { name, got, want string }
func (r rows) check(t *testing.T) {
 for _, item := range r { if item.got != item.want { t.Errorf("mismatch", item.want) } }
}
func TestRows(t *testing.T) { rows{{"日本語", Value(3), "ok"}}.check(t) }
'''
def bind(source):
    req={'files':[dict(id='F1',path='binding_test.go',before=source,after=source)]}
    return json.loads(subprocess.check_output(['/tmp/issue155-table-binding'],input=json.dumps(req).encode()))
class TableBindingTests(unittest.TestCase):
    def test_literal_roles_have_exact_unicode_source_and_no_value_execution(self):
        rows=bind(SOURCE);self.assertEqual(len(rows),2)
        for row in rows:
            self.assertEqual(row['actual_expression']['text'],'Value(3)')
            self.assertEqual(row['expected_literal']['text'],'"ok"')
            self.assertEqual(row['failure_condition']['text'],'item.got != item.want')
            for ref in row.values():
                a,b=ref['span'];self.assertEqual(SOURCE.encode()[a:b].decode(),ref['text'])
    def test_reassignment_and_non_testing_receiver_do_not_bind(self):
        self.assertEqual(bind(SOURCE.replace('for _, item := range r {','for _, item := range r { item.want = "other";')),[])
        self.assertEqual(bind(SOURCE.replace('import "testing"','import testing "untrusted/testing"')),[])
        self.assertEqual(bind(SOURCE.replace('rows{{"日本語", Value(3), "ok"}}.check(t)','t = other; rows{{"日本語", Value(3), "ok"}}.check(t)')),[])
if __name__=='__main__':unittest.main()
