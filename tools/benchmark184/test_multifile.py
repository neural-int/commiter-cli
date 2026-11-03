import unittest
from multifile import partition,score_groups,groups_from_membership,payload_for,validate
class MultifileTests(unittest.TestCase):
 def test_labels_do_not_score(self):
  self.assertTrue(score_groups({'F1':'G009','F2':'G009','F3':'G001'},{'groups':[{'file_ids':['F1','F2']},{'file_ids':['F3']}]})['exact_partition'])
 def test_real_partition_and_unresolved(self):
  groups=groups_from_membership({'F1':'G009','F2':'G003'},[{'file_id':'F1'},{'file_id':'F2'}]);self.assertEqual(groups[0]['group_id'],'G001')
  with self.assertRaisesRegex(ValueError,'unresolved'):payload_for({'agent_history_contexts':[]},groups,{'G001':{'type':'fix','breaking_evidence_ref':'unresolved'},'G002':{'type':'docs','breaking_evidence_ref':'none'}})
 def test_eleven_ids_and_missing_rejected(self):
  p={f'F{i}':{'type':'string','enum':['G001']} for i in range(1,12)};s={'type':'object','properties':p,'required':list(p),'additionalProperties':False};v={k:'G001' for k in p};validate(v,s);v.pop('F11')
  with self.assertRaises(ValueError):validate(v,s)
if __name__=='__main__':unittest.main()
