import importlib.util,json,subprocess,tempfile,unittest
from pathlib import Path
spec=importlib.util.spec_from_file_location('runner',Path(__file__).with_name('run.py'));run=importlib.util.module_from_spec(spec);spec.loader.exec_module(run)
class Contracts(unittest.TestCase):
    def test_duplicate_unknown_and_nonsequential_purposes_rejected(self):
        for text in ['{"purposes":[],"purposes":[]}','{"purposes":[{"purpose_id":"P002","description":"x"}]}','{"purposes":[{"purpose_id":"P001","description":"x","gold":1}]}']:
            with self.assertRaises(ValueError):run.validate_discovery(text)
    def test_complete_assignment_required(self):
        request={'messages':[{}, {'content':json.dumps({'files':[{'id':'f1'},{'id':'f2'}]})}]}
        for text in ['{"f1":"G001"}','{"f1":"G001","f2":"G001","other":"G001"}','{"f1":"G003","f2":"G001"}','{"f1":"G001","f1":"G002","f2":"G001"}']:
            with self.assertRaises(ValueError):run.validate_grouping(text,request)
    def test_receipt_detects_index_and_source_mutation(self):
        with tempfile.TemporaryDirectory() as d:
            root=Path(d);subprocess.run(['git','init','-q',d],check=True);(root/'file.go').write_text('before')
            before=run.repo_digest(root);(root/'file.go').write_text('after');self.assertNotEqual(before,run.repo_digest(root));(root/'file.go').write_text('before');self.assertEqual(before,run.repo_digest(root));subprocess.run(['git','-C',d,'add','file.go'],check=True);self.assertNotEqual(before,run.repo_digest(root))
    def test_paired_inputs_and_native_preflight(self):
        data=Path(__file__).resolve().parents[2]/'docs/benchmarks/issue-169';rows=json.loads((data/'frozen-inputs/manifest.json').read_text())
        for r in rows:
            self.assertIsInstance(r['token_count_before_inference'],int)
            self.assertEqual(r['admitted'],r['token_count_before_inference']+768<=16384)
            req=json.loads((data/'frozen-inputs'/r['request_file']).read_text());self.assertEqual(req['output_tokens'],768)
        self.assertTrue(all(not r['admitted'] for r in rows if r['cohort']=='scale_probe'))
        self.assertTrue(all(r['admitted'] for r in rows if r['cohort'] in ['known','holdout']))
if __name__=='__main__':unittest.main()
