import unittest
from symbol_history import summarize
class HistoryWindow(unittest.TestCase):
 def test_unknown_boundaries_and_mixed_commit_do_not_become_gold(self):
  events=[dict(status=s,symbols=[]) for s in ['unknown_initial','unknown_merge','unknown_rename','unknown_missing']]
  events += [dict(status='observed_symbol_snapshot',symbols=x) for x in [['Left'],[],['Right'],['Left','Right']]]
  r=summarize(events,['Left','Right','New'])
  self.assertEqual(r['unknown_events'],4)
  pair=next(x for x in r['relations'] if x['symbols']==['Left','Right'])
  self.assertEqual((pair['cochange'],pair['separate_touches']),(1,2))
  self.assertTrue(all(x['status']=='unknown_sparse_history' for x in r['relations'] if 'New' in x['symbols']))
 def test_window_budget(self):
  with self.assertRaisesRegex(ValueError,'history_budget'): summarize([{}]*65,[])
