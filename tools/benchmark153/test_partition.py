import unittest
from partition import partition
class ExactPartition(unittest.TestCase):
 def test_global_objective_and_input_order(self):
  scores={'A__B':2,'A__C':-2,'B__C':-2}
  a,meta=partition(['A','B','C'],scores)
  self.assertEqual(a,{'A':'G001','B':'G001','C':'G002'})
  self.assertEqual((a,meta),partition(['C','B','A'],dict(reversed(list(scores.items())))))
  self.assertEqual(meta['states'],5)
 def test_ambiguous_and_incomplete_fail_closed(self):
  with self.assertRaisesRegex(ValueError,'ambiguous'):partition(['A','B'],{'A__B':0})
  with self.assertRaisesRegex(ValueError,'coverage'):partition(['A','B'],{})
  with self.assertRaisesRegex(ValueError,'invalid_score'):partition(['A','B'],{'A__B':True})
 def test_bounded_eight_units_enumerates_all_partitions(self):
  ids=list('ABCDEFGH');scores={a+'__'+b:-2 for i,a in enumerate(ids) for b in ids[i+1:]}
  p,m=partition(ids,scores);self.assertEqual(m['states'],4140);self.assertEqual(len(set(p.values())),8)
  with self.assertRaisesRegex(ValueError,'budget'):partition(list('ABCDEFGHI'),{})
