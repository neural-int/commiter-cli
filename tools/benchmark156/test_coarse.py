import json,pathlib,unittest
from coarse import prepare,refine
from planner import prepare as eager

class DeferredRefinement(unittest.TestCase):
    def test_public_budget_failure_retains_complete_coarse_ownership(self):
        root=pathlib.Path(__file__).resolve().parents[2]/'docs/benchmarks/issue-156/corpus'
        snapshot=json.loads(next(root.glob('google-go-cmp-b133f1f1*-snapshots.json')).read_text())
        f=next(f for f in snapshot['files'] if f['path']=='cmp/compare_test.go')
        with self.assertRaisesRegex(ValueError,'unit_budget'):eager([f])
        units=prepare([f]);retained,reason=refine(f)
        self.assertEqual(reason,'unit_budget')
        self.assertEqual(retained,units)
        self.assertLessEqual(len(units),256)

if __name__=='__main__':unittest.main()
