import unittest
from poc import classify,edit,evaluate
class AttributionLimits(unittest.TestCase):
 def test_reverse_direction_is_not_correspondence(self):
  self.assertEqual(classify(edit(),[edit(before='B',after='A')])['status'],'unmatched')
 def test_repeated_identical_edit_is_ambiguous(self):
  self.assertEqual(classify(edit(),[edit(),edit('T2')])['status'],'ambiguous')
 def test_composed_change_is_not_fabricated(self):
  self.assertEqual(classify(edit(after='C'),[edit(),edit('T2',before='B',after='C')])['turns'],[])
 def test_no_false_strong_synthetic_candidates(self):
  self.assertTrue(all(r['strong_false_candidates']==0 for r in evaluate()['cases']))
if __name__=='__main__':unittest.main()
