import copy
import unittest
from attribution import prepare,validate,validate_direct

class AttributionTest(unittest.TestCase):
    def setUp(self):
        files=[dict(id='F1',path='value.go',before='package sample\nfunc Value(n int) int { return n+1 }\n',after='package sample\nfunc Value(n int) int { return n+2 }\n'),dict(id='F2',path='value_test.go',before='package sample\nimport "testing"\nfunc TestValue(t *testing.T) { if Value(2)!=3 {t.Fatal("bad")} }\n',after='package sample\nimport "testing"\nfunc TestValue(t *testing.T) { if Value(2)!=4 {t.Fatal("bad")} }\n')]
        self.payload,_=prepare(files)
        anchor=self.payload['observations'][0]
        refs=[c['id'] for c in self.payload['source_context']]
        self.good=dict(assignments=[dict(role='implements_changed_behavior',unit_ids=['U001'],observed_contract_ids=[anchor['id']],evidence_refs=refs,reason='direct_source_evidence'),dict(role='asserts_changed_behavior',unit_ids=['U002'],observed_contract_ids=[anchor['id']],evidence_refs=refs,reason='direct_source_evidence')])

    def test_valid_mapping_and_unknown_are_distinct(self):
        result=validate(self.payload,self.good)
        self.assertEqual(result['U001']['role'],'implements_changed_behavior')
        unknown=dict(assignments=[dict(role='unknown',unit_ids=['U001','U002'],observed_contract_ids=[],evidence_refs=[],reason='ambiguous_or_missing_context')])
        self.assertTrue(all(x['role']=='unknown' for x in validate(self.payload,unknown).values()))

    def test_missing_duplicate_forged_and_mismatched_refs_reject(self):
        variants=[]
        bad=copy.deepcopy(self.good);bad['assignments'].pop();variants.append(bad)
        bad=copy.deepcopy(self.good);bad['assignments'][1]['unit_ids']=['U001'];variants.append(bad)
        bad=copy.deepcopy(self.good);bad['assignments'][0]['observed_contract_ids']=['fake'];variants.append(bad)
        bad=copy.deepcopy(self.good);bad['assignments'][0]['evidence_refs']=[];variants.append(bad)
        bad=copy.deepcopy(self.good);bad['assignments'][0]['role']='asserts_changed_behavior';variants.append(bad)
        for bad in variants:
            with self.assertRaises(ValueError):validate(self.payload,bad)

    def test_direct_partial_and_unresolved_reject(self):
        self.assertEqual(len(validate_direct(self.payload,dict(groups=[['U001','U002']],unresolved=False))),2)
        for answer in (dict(groups=[['U001']],unresolved=False),dict(groups=[['U001','U002']],unresolved=True)):
            with self.assertRaises(ValueError):validate_direct(self.payload,answer)

if __name__=='__main__':unittest.main()
