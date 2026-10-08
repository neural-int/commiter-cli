import copy,pathlib,sys,unittest
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parent))
import refinement
class RefinementTest(unittest.TestCase):
 def setUp(self):
  self.snap={'F1':(b'x=1; y=2\n',b'x=3; y=4\n')}
  self.h=refinement.contiguous.extract(*self.snap['F1'],'F1');self.p=self.h['operations'][0];self.ids=self.p['children']
 def test_accept_refine_and_order(self):
  accepted=refinement.validate(self.snap,{self.p['id']:{'action':'accept'}});self.assertEqual(len(accepted),1)
  plan={self.p['id']:{'action':'refine','groups':[[a] for a in self.ids]}}
  result=refinement.validate(self.snap,plan);self.assertEqual(len(result),2)
  plan[self.p['id']]['groups'].reverse();self.assertEqual(result,refinement.validate(self.snap,plan))
 def test_missing_duplicate_foreign_and_empty(self):
  for groups in ([[self.ids[0]],[self.ids[0]]],[[self.ids[0]],['foreign']],[[],self.ids]):
   with self.assertRaises(ValueError):refinement.validate(self.snap,{self.p['id']:{'action':'refine','groups':groups}})
  with self.assertRaisesRegex(ValueError,'parent_coverage'):refinement.validate(self.snap,{})
 def test_unknown_and_snapshot_change(self):
  with self.assertRaisesRegex(ValueError,'unresolved'):refinement.validate(self.snap,{self.p['id']:{'action':'unresolved'}})
  with self.assertRaisesRegex(ValueError,'parent_coverage'):refinement.validate({'F1':(b'x=9; y=2\n',self.snap['F1'][1])},{self.p['id']:{'action':'accept'}})
 def test_budget_rejects_without_truncation(self):
  snap={'F1':(b'a'*9+b'\n',b'b'*9+b'\n')};h=refinement.contiguous.extract(*snap['F1'],'F1');p=h['operations'][0]
  with self.assertRaisesRegex(ValueError,'refined_unit_budget'):refinement.validate(snap,{p['id']:{'action':'refine','groups':[[a] for a in p['children']]}})
if __name__=='__main__':unittest.main()
