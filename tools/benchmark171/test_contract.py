import copy,json,sys,tempfile,unittest
from pathlib import Path
ROOT=Path(__file__).resolve().parents[2]
sys.path.insert(0,str(Path(__file__).parent))
import audit,evaluate
from run import validate_grouping,check_pins,sha
class Contracts(unittest.TestCase):
 def test_gold_derivation_and_unused_source(self):audit.audit_gold()
 def test_fallback_not_false_split(self):
  f=json.loads((ROOT/'tools/benchmark171/fixtures.json').read_text())[4]
  obj={'path:'+p:'G'+str(i) for i,g in enumerate(f['gold_groups']) for p in g};s=evaluate.score(f,obj)
  self.assertEqual((s['exact'],s['fs'],s['fm']),(1,0,0));self.assertEqual(s['gold_fallback_fragmented_purposes'],2)
  merged={p:'G1' for p in obj};m=evaluate.score(f,merged);self.assertEqual(m['fm'],3);self.assertEqual(m['exact'],0)
 def test_pair_counts_and_normalization(self):
  f={'gold_status':'confirmed','files':[{'path':x,'before':'a','after':'b'} for x in 'abc'],'allowed_partitions':[[['a','b'],['c']]],'fallback_files':[],'fallback_purpose_fragmentation':{}}
  good={'path:a':'G003','path:b':'G003','path:c':'G001'};self.assertEqual(evaluate.score(f,good)['exact'],1)
  merged={x:'G001' for x in good};self.assertEqual((evaluate.score(f,merged)['fm'],evaluate.score(f,merged)['fs']),(2,0))
  singleton={x:'G'+str(i) for i,x in enumerate(good)};self.assertEqual((evaluate.score(f,singleton)['fm'],evaluate.score(f,singleton)['fs']),(0,1))
  invalid=evaluate.score(f,None);self.assertEqual(invalid['exact'],0);self.assertIsNone(invalid['fm']);self.assertIsNone(invalid['fs'])
 def test_multiple_valid_not_main_or_favorable_pair_mixing(self):
  f={'gold_status':'multiple_valid','files':[{'path':x,'before':'a','after':'b'} for x in 'ab'],'allowed_partitions':[[['a','b']],[['a'],['b']]],'fallback_files':[],'fallback_purpose_fragmentation':{}}
  s=evaluate.score(f,{'path:a':'G001','path:b':'G002'});self.assertEqual(s['exact'],1);self.assertEqual(len(s['per_allowed_partition']),2)
  f['gold_status']='undetermined';self.assertIsNone(evaluate.score(f,None)['exact'])
 def test_fail_closed_assignments(self):
  req={'messages':[{}, {'content':json.dumps({'files':[{'id':'path:a'},{'id':'path:b'}]})}]}
  for text in ['{"path:a":"G001"}','{"path:a":"G001","path:a":"G002","path:b":"G001"}','{"path:a":"G999","path:b":"G001"}','{"path:a":"G001","path:b":"G001","path:c":"G001"}']:
   with self.assertRaises(ValueError):validate_grouping(text,req)
 def test_pin_tamper_rejected(self):
  with tempfile.TemporaryDirectory() as d:
   p=Path(d)/'request';p.write_text('original');pr={'sha256':{'runtime:probe':sha(p)}};local={'probe':str(p)};check_pins(pr,local);p.write_text('changed')
   with self.assertRaises(ValueError):check_pins(pr,local)
if __name__=='__main__':unittest.main()
