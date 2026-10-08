import pathlib,sys,unittest
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parent))
import payload_budget as p
class BudgetTest(unittest.TestCase):
 def test_lossless_proposal_and_mapping(self):
  snap={'F1':(b'a=1\n',b'a=2\n')};messages,schema,mapping,size=p.build(snap)
  self.assertIn('a=1',messages[1]['content']);self.assertIn('a=2',messages[1]['content'])
  self.assertEqual(p.validate_decisions({'P001':'refine'},mapping),{mapping['P001']:'refine'})
 def test_partial_unknown_and_extra_rejected(self):
  m={'P001':'long-id'}
  for answer in ({},{'P001':'accept','P002':'accept'},{'P001':{'action':'accept'}},{'P001':'unresolved'}):
   with self.assertRaises(ValueError):p.validate_decisions(answer,m)
 def test_oversize_rejected_before_backend(self):
  snap={'F1':(b'a'*4000+b'1\n',b'a'*4000+b'2\n')}
  with self.assertRaisesRegex(ValueError,'message_byte_budget'):p.build(snap)
if __name__=='__main__':unittest.main()
