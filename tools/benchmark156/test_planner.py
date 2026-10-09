import base64
import copy
import unittest
from planner import prepare, fallback, validate, coverage, replay


def file(before=b'a b\n', after=b'c d\n', **extra):
    return dict(id='F001',path='settings.txt',before_b64=base64.b64encode(before).decode(),after_b64=base64.b64encode(after).decode(),**extra)

class SafetyContracts(unittest.TestCase):
    def test_corrupt_source_and_duplicate_assignment_stop_before_git(self):
        files=[file()];units=prepare(files);commits=fallback(files,units)
        for fault in ('missing','duplicate','unknown'):
            bad=copy.deepcopy(commits)
            if fault=='missing':bad[0]['unit_ids'].pop()
            if fault=='duplicate':bad[0]['unit_ids'].append(bad[0]['unit_ids'][0])
            if fault=='unknown':bad[0]['unit_ids'][0]='invented'
            with self.assertRaisesRegex(ValueError,'invalid_assignment'):validate(units,bad)
        wrong=copy.deepcopy(units);wrong[0]['new_span']=[0,0]
        with self.assertRaisesRegex(ValueError,'after_source_mapping'):coverage(b'a b\n',b'c d\n',wrong)

    def test_scope_mode_and_fixed_resource_budget_stop(self):
        for path in ('../outside','.git/index','/tmp/outside'):
            f=file();f['path']=path
            with self.assertRaisesRegex(ValueError,'invalid_file_mapping'):prepare([f])
        with self.assertRaisesRegex(ValueError,'unsupported_mode'):prepare([file(mode='120000')])
        with self.assertRaisesRegex(ValueError,'file_budget'):prepare([file()]*17)
        with self.assertRaisesRegex(ValueError,'unit_budget'):prepare([file(b'a'*257,b'b'*257)])
        with self.assertRaisesRegex(ValueError,'inline_byte_budget'):prepare([file(b'a'*4097,b'b'*4097)])

    def test_partial_inline_replay_and_unknown_metadata(self):
        files=[file('猫 赤\n'.encode(),'犬 青\n'.encode())];units=prepare(files)
        commits=[dict(file_ids=['F001'],unit_ids=[u['id']]) for u in units]
        result=replay(files,units,commits)
        self.assertTrue(result['forward_reverse'] and result['final_tree_equality'])
        group=fallback(files,units)[0]
        self.assertEqual(group['purpose_status'],'unknown')
        self.assertEqual(group['mixed_intent_risk'],'unresolved')
        self.assertFalse(group['compose_candidate'])

if __name__=='__main__':unittest.main()
