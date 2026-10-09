import unittest
from c_cases import cases
from coarse import prepare
from adaptive import request,finish
from evaluate_c2 import authoritative
from planner import fallback

class PartialFileCompatibility(unittest.TestCase):
    def test_same_file_split_is_rejected_and_unknown_keeps_complete_file(self):
        case=next(c for c in cases() if c['name']=='separate-hunks')
        files=case['files'];units=prepare(files);_,_,labels=request(files,units)
        answer=dict(status={'F001':'mixed'},assignment={k:i+1 for i,k in enumerate(labels)})
        split=finish(files,units,labels,answer)
        self.assertFalse(authoritative(files,split)['valid'])
        answer['status']['F001']='unknown'
        safe=finish(files,units,labels,answer)
        self.assertEqual(safe[0]['unit_ids'],fallback(files,units)[0]['unit_ids'])
        self.assertTrue(authoritative(files,safe)['valid'])

if __name__=='__main__':unittest.main()
