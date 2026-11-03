import unittest
from multifile004 import prune,describe,load_templates,NEW_SUMMARY_INSTRUCTION
class ProjectionTests(unittest.TestCase):
 def test_nonempty_requests_are_display_only_omissions(self):
  source={'user_requests':['past request'],'final_responses':['completed work']};display=prune(source)
  self.assertNotIn('user_requests',display);self.assertEqual(source['user_requests'],['past request']);self.assertEqual(display['final_responses'],['completed work'])
 def test_host_unchanged_and_uncertainty_preserved(self):
  source={'block_id':'B1','execution_ref':'S1','user_requests':[],'raw_diff':'@@\n-a\n+b\n','agent_history_match_status':'ambiguous','agent_history_candidates':[{'context_ref':'H1','match':'exact'}]};result=prune(source)
  self.assertIn('block_id',source);self.assertNotIn('block_id',result);self.assertEqual(result['raw_diff'],source['raw_diff']);self.assertEqual(result['agent_history_match_status'],'ambiguous');self.assertEqual(result['agent_history_candidates'],source['agent_history_candidates'])
 def test_dynamic_fields(self):
  s={'type':'object','properties':{f'F{i}':{'type':'string'} for i in range(1,12)},'required':[f'F{i}' for i in range(1,12)]};lines=describe(s);self.assertEqual(len(lines),12);self.assertIn('$.F11: type=string; required',lines)
  self.assertLess(lines.index('$.F2: type=string; required'),lines.index('$.F11: type=string; required'))
 def test_repository_prompts_without_private_template(self):
  templates=load_templates()
  self.assertEqual(set(templates),{'grouping','category','text'})
  self.assertEqual(templates['grouping']['messages'][:2],templates['text']['messages'][:2])
  self.assertIn(NEW_SUMMARY_INSTRUCTION,templates['text']['messages'][2]['content'])
 def test_validator_permission_is_checked_before_input_and_inference(self):
  import subprocess,sys,tempfile,pathlib
  with tempfile.TemporaryDirectory() as directory:
   root=pathlib.Path(directory);validator=root/'validator';validator.write_text('not executable')
   script=pathlib.Path(__file__).with_name('run004.py')
   result=subprocess.run([sys.executable,str(script),'--manifest',str(root/'missing.json'),'--helper',sys.executable,'--validator',str(validator),'--thermal-probe',sys.executable,'--output',str(root/'output')],text=True,capture_output=True)
   self.assertNotEqual(result.returncode,0);self.assertIn('required executable unavailable: validator',result.stderr);self.assertFalse((root/'output').exists())
 def test_summary_variant_changes_instructions_only(self):
  from multifile004 import summary_instruction_variant,OLD_SUMMARY_INSTRUCTION,PHASE_INPUT
  import copy
  q={'generation_profile':'bounded-text','schema':{'properties':{'G001':{'properties':{'summary':{'maxLength':48}}}}},'messages':[{'content':'common system'},{'content':'common data'},{'content':OLD_SUMMARY_INSTRUCTION+PHASE_INPUT+'{"groups":[]}'}]};before=copy.deepcopy(q);new=summary_instruction_variant(q)
  self.assertEqual(q,before);self.assertEqual(new['schema'],q['schema']);self.assertEqual(new['messages'][:2],q['messages'][:2]);self.assertEqual(new['messages'][2]['content'].split(PHASE_INPUT)[1],'{"groups":[]}')
if __name__=='__main__':unittest.main()
