import importlib.util,pathlib,unittest
spec=importlib.util.spec_from_file_location('study_build',pathlib.Path(__file__).with_name('build.py'))
study=importlib.util.module_from_spec(spec);spec.loader.exec_module(study)

class StudyFileLimitsTest(unittest.TestCase):
    def test_old_sixteen_limit_preserved(self):
        source=(study.ROOT/'internal/planning/three_phase.go').read_text()
        self.assertIn('const CandidateMaxFiles = 16',study.planner_overlay(source))
        self.assertIn(study.FILE_GUARD,study.planner_overlay(source))
    def test_unlimited_removes_only_planner_guard(self):
        source=(study.ROOT/'internal/planning/three_phase.go').read_text()
        result=study.planner_overlay(source,True)
        self.assertNotIn('if len(ids) > CandidateMaxFiles',result)
        self.assertIn('context.WithTimeout(parent, CandidateCycleTimeout)',result)
        self.assertIn('three-phase planner requires a supported context ceiling',result)
    def test_unlimited_removes_both_live_limits(self):
        source=(study.ROOT/'internal/cli/attributed_history_live_test.go').read_text()
        result=study.live_overlay(source)
        self.assertNotIn('len(paths) > 4',result)
        self.assertNotIn('len(snapshot.Changes) > 4',result)
        self.assertIn('len(snapshot.Changes) == 0',result)
        self.assertIn('len(paths) == 0',result)

if __name__=='__main__':unittest.main()
