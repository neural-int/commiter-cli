import copy
import unittest
from test_planner import file
from planner import prepare
from selective import finish

class SelectiveSafety(unittest.TestCase):
    def setup_files(self):
        files=[file(),dict(file(),id='F002',path='test.txt'),dict(file(),id='F003',path='other.txt')]
        return files,prepare(files)
    def answer(self,ids,score=3):
        return dict(score=score,members={fid:dict(status='single',coverage='all',evidence_refs=['E'+fid]) for fid in ids})
    def test_partial_mixed_file_cannot_be_absorbed_by_high_score(self):
        files,units=self.setup_files();candidate=dict(id='P001',file_ids=['F001','F002'])
        response=self.answer(candidate['file_ids']);response['members']['F001'].update(status='mixed',coverage='partial')
        groups=finish(files,units,[candidate],{'P001':response})
        self.assertEqual([c['file_ids'] for c in groups],[['F001'],['F002'],['F003']])
        self.assertTrue(all(c['fallback'] for c in groups))
    def test_equal_bridge_proposals_never_form_transitive_union(self):
        files,units=self.setup_files()
        candidates=[dict(id='P001',file_ids=['F001','F002']),dict(id='P002',file_ids=['F002','F003'])]
        response={c['id']:self.answer(c['file_ids']) for c in candidates}
        groups=finish(files,units,candidates,response)
        self.assertTrue(all(len(c['file_ids'])==1 for c in groups))
        bad=copy.deepcopy(response);bad['P001']['members']['F001']['evidence_refs']=['invented']
        with self.assertRaisesRegex(ValueError,'invalid_member_schema'):finish(files,units,candidates,bad)

if __name__=='__main__':unittest.main()
