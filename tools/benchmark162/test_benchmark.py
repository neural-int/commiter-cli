import unittest
from benchmark import validate, strict, cases, mechanical

class Contracts(unittest.TestCase):
 def test_fabricated_or_missing_source_cannot_support_definite_relation(self):
  for refs in (['EA'],['EA','EC'],['EA','EA']):
   with self.assertRaises(ValueError): validate({'decision':'together','evidence_refs':refs})
  self.assertEqual(validate({'decision':'unknown','evidence_refs':[]})['decision'],'unknown')
 def test_duplicate_decision_cannot_overwrite_first_answer(self):
  with self.assertRaisesRegex(ValueError,'duplicate_json_key'):strict([('decision','separate'),('decision','together')])
 def test_symbol_link_does_not_prove_review_coupling(self):
  c=next(c for c in cases() if c['name']=='shared-name-independent')
  self.assertTrue(mechanical(c)['linked'])
  self.assertEqual(c['gold'],'separate')
if __name__=='__main__':unittest.main()
