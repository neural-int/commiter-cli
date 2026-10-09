import unittest
from partition_trial import dataset, facts, dependencies, plan, validate, review_valid, authoritative

class PartitionContracts(unittest.TestCase):
 def test_new_api_direction_is_host_owned_even_when_display_order_reverses(self):
  for name,provider,consumer in [('provider-before-consumer','EB','EA'),('consumer-after-provider','EA','EB')]:
   c=next(c for c in dataset() if c['name']==name)
   for fs in (c['files'],list(reversed(c['files']))):
    edges=dependencies(facts(fs));self.assertEqual([(e['provider'],e['consumer']) for e in edges],[(provider,consumer)])
    ordered=plan(c,'keep_separate',edges);flat=[x for g in ordered for x in g]
    self.assertLess(flat.index(provider),flat.index(consumer))
    self.assertTrue(authoritative(c,ordered)['valid'])
 def test_cycle_cannot_be_silently_ordered(self):
  c=dataset()[0]
  with self.assertRaisesRegex(ValueError,'dependency_cycle_unsupported'):
   plan(c,'keep_separate',[{'provider':'EA','consumer':'EB'},{'provider':'EB','consumer':'EA'}])
 def test_multiple_valid_does_not_force_a_hidden_author_boundary(self):
  c=next(c for c in dataset() if c['kind']=='multiple_valid')
  self.assertTrue(review_valid(c,plan(c,'merge',[])))
  self.assertTrue(review_valid(c,plan(c,'keep_separate',[])))
  with self.assertRaisesRegex(ValueError,'invalid_evidence_refs'):validate({'decision':'merge','evidence_refs':['EA','EC']})
if __name__=='__main__':unittest.main()
