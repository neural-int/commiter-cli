import importlib.util
import pathlib
import sys
import unittest
sys.path.insert(0,str(pathlib.Path(__file__).parent))
spec=importlib.util.spec_from_file_location('run152',pathlib.Path(__file__).parent/'evaluate.py')
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)

class ScoringContracts(unittest.TestCase):
    def test_duplicate_missing_unknown_and_unresolved_fail_closed(self):
        with self.assertRaisesRegex(ValueError,'duplicate_json_key'):
            m.json.loads('{"a":1,"a":2}',object_pairs_hook=m.strict)
        for answer,reason in [(dict(groups={},unresolved=False),'invalid_schema'),(dict(membership={},unresolved=False),'invalid_assignment'),(dict(membership={'U001':'G001'},unresolved=True),'unresolved'),(dict(membership={'U001':'G999'},unresolved=False),'invalid_group')]:
            with self.assertRaisesRegex(ValueError,reason):m.validate(answer,['U001'])

    def test_native_zero_request_includes_schema_in_messages(self):
        import json
        from unittest.mock import patch
        class Result:
            returncode=0
            stdout=json.dumps(dict(ok=True,stop_reason='completed',generated_json=json.dumps(dict(membership={'U001':'G001'},unresolved=False)),benchmark_input_tokens=100,benchmark_output_tokens=10)).encode()
        with patch.object(m.subprocess,'run',return_value=Result()) as run:
            membership,meta=m.invoke({'change_units':[{'id':'U001'}]},'helper','model')
            request=json.loads(run.call_args.kwargs['input'])
        self.assertIn(json.dumps(request['schema'],sort_keys=True),request['messages'][0]['content'])
        self.assertEqual(membership,{'U001':'G001'})
        self.assertEqual(meta['stop'],'completed')
        self.assertEqual(meta['validation_reason'],'accepted')

    def test_backend_stop_preserved_for_host_schema_failure(self):
        import json
        from unittest.mock import patch
        class Result:
            returncode=0
            stdout=json.dumps(dict(ok=True,stop_reason='completed',generated_json='{"other":{},"unresolved":false}',benchmark_input_tokens=100,benchmark_output_tokens=10)).encode()
        with patch.object(m.subprocess,'run',return_value=Result()):
            membership,meta=m.invoke({'change_units':[{'id':'U001'}]},'helper','model')
        self.assertIsNone(membership)
        self.assertEqual(meta['stop'],'completed')
        self.assertEqual(meta['validation_reason'],'invalid_schema')

if __name__=='__main__': unittest.main()
