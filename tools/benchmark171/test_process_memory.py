import importlib.util,json,unittest
from pathlib import Path
spec=importlib.util.spec_from_file_location('process_runner171',Path(__file__).with_name('run_process_memory.py'));runner=importlib.util.module_from_spec(spec);spec.loader.exec_module(runner)
class ProcessMemoryPolicy(unittest.TestCase):
 def setUp(self):
  root=Path(__file__).resolve().parents[2];self.record=json.loads((root/'docs/benchmarks/issue-171/run-2/measurements.jsonl').read_text());self.pr={'max_rss_bytes':8*1024**3,'max_mlx_peak_bytes':8*1024**3}
 def test_real_swap_excess_does_not_reject_completed_helper(self):
  self.assertGreater(self.record['resource_after']['swap_used_mib']-self.record['resource_before']['swap_used_mib'],1024)
  self.assertIsNone(runner.process_stop_reason(self.record,self.pr))
 def test_helper_allocations_still_fail_closed(self):
  self.record['peak_bytes']=self.pr['max_mlx_peak_bytes']+1
  self.assertEqual(runner.process_stop_reason(self.record,self.pr),'helper memory cap exceeded')
 def test_critical_pressure_and_timeout_are_not_ignored(self):
  self.record['resource_after']['pressure_level']=4
  self.assertEqual(runner.process_stop_reason(self.record,self.pr),'missing or critical memory-pressure observer')
  self.record['status']='timeout'
  self.assertEqual(runner.process_stop_reason(self.record,self.pr),'timeout')
if __name__=='__main__':unittest.main()
