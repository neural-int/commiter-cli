import copy
import unittest
from local_facts import prepare,validate

class LocalFactsTest(unittest.TestCase):
    def test_exact_source_trace_and_wrong_path_reject(self):
        files=[dict(id='F1',path='count.go',before='package scope\nfunc Count(n int) int {return n}\n',after='package scope\nfunc Count(n int) int {if n<0 {return 0};return n}\n')]
        payload,oracle=prepare(files,[dict(package='.',function='Count',arguments=[3])])
        validate(payload,oracle,oracle)
        wrong=copy.deepcopy(oracle);wrong['Q001']['after']['value']=0
        with self.assertRaisesRegex(ValueError,'value_mismatch'):validate(payload,wrong,oracle)
        wrong=copy.deepcopy(oracle)
        correct=set(oracle['Q001']['after']['return_refs'])
        other=next(n['id'] for n in payload['witnesses'] if n['kind']=='return' and n['version']=='after' and n['id']not in correct)
        wrong['Q001']['after']['return_refs']=[other]
        with self.assertRaisesRegex(ValueError,'return_path_mismatch'):validate(payload,wrong,oracle)

    def test_boolean_does_not_pass_as_float(self):
        files=[dict(id='F1',path='value.go',before='package scope\nfunc Value() float64 {return 1}\n',after='package scope\nfunc Value() float64 {return 1}\n')]
        payload,oracle=prepare(files,[dict(package='.',function='Value',arguments=[])])
        wrong=copy.deepcopy(oracle);wrong['Q001']['before']['value']=True
        with self.assertRaisesRegex(ValueError,'value_mismatch'):validate(payload,wrong,oracle)

    def test_unknown_is_not_numeric_zero_or_fabricated_knowledge(self):
        files=[dict(id='F1',path='opaque.go',before='package scope\nfunc Opaque(n int) int {return Missing(n)}\n',after='package scope\nfunc Opaque(n int) int {return Missing(n)}\n')]
        payload,oracle=prepare(files,[dict(package='.',function='Opaque',arguments=[3])])
        validate(payload,oracle,oracle)
        wrong=copy.deepcopy(oracle);wrong['Q001']['before']['value']=0
        with self.assertRaises(ValueError):validate(payload,wrong,oracle)

if __name__=='__main__':unittest.main()
