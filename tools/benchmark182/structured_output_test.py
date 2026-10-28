import contextlib, importlib.util, io, json, pathlib, tempfile, unittest
from unittest.mock import patch
spec=importlib.util.spec_from_file_location('study',pathlib.Path(__file__).with_name('structured_output.py'))
study=importlib.util.module_from_spec(spec);spec.loader.exec_module(study)

class StudyHarnessTest(unittest.TestCase):
    def test_matrix_keeps_eight_models_and_two_constraints(self):
        with tempfile.TemporaryDirectory() as temp:
            out=pathlib.Path(temp)/'study';study.prepare(out)
            matrix=json.loads((out/'matrix.json').read_text())
            self.assertEqual((matrix['model_count'],matrix['model_thinking_cases'],len(matrix['cases'])),(8,9,18))
            configs=[json.loads((out/r['config']).read_text()) for r in matrix['cases']]
            self.assertEqual(len({c['model'] for c in configs}),8)
            self.assertTrue(all(not c['thinking'] for c in configs if c['id']=='qwen2-off'))
            for key in {c['id'] for c in configs}:
                modes=[c for c in configs if c['id']==key]
                self.assertEqual({c.pop('output_constraint') for c in modes},{'none','json_schema'})
                self.assertEqual(modes[0],modes[1])
    def test_proxy_passes_actual_schema_and_payload_without_fixed_groups(self):
        with tempfile.TemporaryDirectory() as temp:
            root=pathlib.Path(temp);config=root/'case.json';assets=root/'assets';assets.mkdir()
            study.write(config,dict(model='test@revision',output_constraint='none'))
            study.write(assets/'asset-manifest.json',dict(model='test@revision',max_context=32768))
            request=dict(generation_profile='bounded-text',model='old',schema={'type':'object','properties':{'G9':{'type':'string'}}},
                         messages=[{'role':'user','content':'actual G9 payload'}],context_tokens=65536,output_tokens=512)
            with patch.object(study.sys,'stdin',io.StringIO(json.dumps(request))),patch.object(study.subprocess,'run') as run:
                run.return_value.returncode=0
                with self.assertRaises(SystemExit) as result:study.proxy(root/'helper',config,assets)
            self.assertEqual(result.exception.code,0)
            self.assertEqual(run.call_args.args[0], [str(root/'helper'),'--benchmark-single'])
            actual=json.loads(run.call_args.kwargs['input'])
            self.assertEqual(actual['schema'],request['schema']);self.assertEqual(actual['messages'],request['messages'])
            self.assertEqual((actual['context_tokens'],actual['output_tokens']),(32768,0))
            self.assertNotIn('COMMITER_182_PATH',run.call_args.kwargs['env'])
            self.assertEqual(run.call_args.kwargs['env']['COMMITER_180_KV_BITS'],'8')

if __name__=='__main__':unittest.main()
