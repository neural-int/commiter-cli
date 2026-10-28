import importlib.util,json,pathlib,tempfile,unittest
spec=importlib.util.spec_from_file_location('driver',pathlib.Path(__file__).with_name('kimdel_inference.py'));driver=importlib.util.module_from_spec(spec);spec.loader.exec_module(driver)
class EvaluationTest(unittest.TestCase):
    def fixture(self,root,merged=False):
        files=[];answer={};groups=[];i=0;category={}
        for n,g in enumerate(driver.GOLD):
            members=[]
            for path in g['files']:
                i+=1;f=dict(file_id=f'F{i}',new_path=path);files.append(f);members.append(f);answer[f'F{i}']='X' if merged else f'X{n}'
            groups.append(dict(group_id=f'G{n}',files=members));category[f'G{n}']=dict(type=g['type'],breaking_evidence_ref='none')
        driver.save(root/'01-request.json',dict(messages=[{},dict(content=json.dumps(dict(files=files)))]))
        driver.save(root/'01-response.json',dict(generated_json=json.dumps(answer)))
        driver.save(root/'result.json',dict(error='',git_unchanged=True))
        if not merged:
            driver.save(root/'02-request.json',dict(messages=[{},dict(content=json.dumps(dict(groups=groups)))]))
            driver.save(root/'02-response.json',dict(generated_json=json.dumps(category)))
    def test_partition_labels_do_not_affect_scores(self):
        with tempfile.TemporaryDirectory() as tmp:
            p=pathlib.Path(tmp);self.fixture(p);score=driver.score(p)
            self.assertTrue(score['exact_partition']);self.assertEqual(score['pairs'],dict(tp=2,fp=0,fn=0,tn=8,precision=1,recall=1));self.assertEqual(score['category_correct_on_exact_groups'],3)
    def test_merging_everything_is_not_grouping_success(self):
        with tempfile.TemporaryDirectory() as tmp:
            p=pathlib.Path(tmp);self.fixture(p,True);score=driver.score(p)
            self.assertFalse(score['exact_partition']);self.assertEqual(score['pairs']['fp'],8);self.assertEqual(score['exact_groups'],0)
if __name__=='__main__':unittest.main()
