import importlib.util,json,unittest
from pathlib import Path
ROOT=Path(__file__).resolve().parents[2]
spec=importlib.util.spec_from_file_location('audit170',Path(__file__).with_name('audit.py'));audit=importlib.util.module_from_spec(spec);spec.loader.exec_module(audit)
class Contracts(unittest.TestCase):
 def test_system_only_pairs_and_full_diff_source_binding(self):audit.audit()
 def test_future_grouping_gate_is_fail_closed(self):
  import sys,tempfile
  from unittest.mock import patch
  sys.path.insert(0,str(ROOT/'tools/benchmark169'));import run
  with tempfile.TemporaryDirectory() as d:
   data=Path(d);(data/'preregistration.json').write_text('{"sha256":{}}');(data/'phase1-summary.json').write_text('{"gate1":"NO-GO"}');runtime=data/'runtime.json';runtime.write_text('{}')
   with patch.object(run,'DATA',data),patch.object(sys,'argv',['run','--cohort','holdout','--phase','grouping','--runtime',str(runtime)]):
    with self.assertRaisesRegex(ValueError,'Phase 2 gated out'):run.main()
   self.assertFalse((data/'attempts.jsonl').exists())
 def test_freeze_pins_detect_tampering(self):
  import sys,tempfile
  sys.path.insert(0,str(ROOT/'tools/benchmark169'));import run
  with tempfile.TemporaryDirectory() as d:
   path=Path(d)/'request';path.write_text('original');pr={'sha256':{'runtime:probe':run.sha(path)}};local={'probe':str(path)}
   run.check_pins(pr,local);path.write_text('tampered')
   with self.assertRaises(ValueError):run.check_pins(pr,local)
if __name__=='__main__':unittest.main()
