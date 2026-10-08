import json,subprocess,unittest

class ScopeEvidenceTests(unittest.TestCase):
    def test_comments_are_not_conditions_and_spans_are_utf8(self):
        source='package scope\n// 日本語: if fabricated { t.Errorf("fake") }\nfunc TestValue(t T) { if value != "日本語" { t.Errorf("mismatch") } }\n'
        request={'files':[dict(id='F1',path='value_test.go',before=source,after=source)]}
        nodes=json.loads(subprocess.check_output(['/tmp/issue155-assertion-scope'],input=json.dumps(request).encode()))
        conditions=[n for n in nodes if n['kind']=='failure_condition_syntax']
        self.assertEqual(len(conditions),2)
        for node in conditions:
            a,b=node['span']
            self.assertEqual(source.encode()[a:b].decode(),'value != "日本語"')
            self.assertNotIn('fabricated',node['text'])

if __name__=='__main__':unittest.main()
