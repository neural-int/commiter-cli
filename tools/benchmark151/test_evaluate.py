import copy
import unittest
from evaluate import extract, reconstruct, stage_verify, MAX_BYTES

class ChangeUnitContracts(unittest.TestCase):
    def test_repeat_lines_unicode_and_partial_reconstruction(self):
        before = 'あ\nx\nx\nb\n'.encode()
        after = 'い\nx\nx\nc\n'.encode()
        units = extract(before, after, 'f')
        self.assertEqual(len(units), 2)
        self.assertEqual(reconstruct(before, units, [units[0]['id']]), 'い\nx\nx\nb\n'.encode())
        self.assertTrue(stage_verify(before, after, units))

    def test_unknown_duplicate_stale_and_overlap_rejected(self):
        units = extract(b'a\nx\nb\n', b'c\nx\nd\n', 'f')
        for selected in (['unknown'], [units[0]['id']]*2):
            with self.assertRaises(ValueError): reconstruct(b'a\nx\nb\n', units, selected)
        with self.assertRaises(ValueError): reconstruct(b'z\nx\nb\n', units, [])
        bad = copy.deepcopy(units)
        bad[1]['old_span'] = [0, 2]
        with self.assertRaises(ValueError): reconstruct(b'a\nx\nb\n', bad, [])

    def test_size_budget_fails_without_truncation(self):
        with self.assertRaisesRegex(ValueError, 'byte_budget'):
            extract(b'', b'x'*(MAX_BYTES+1), 'f')

    def test_binary_null_bytes_preserved_by_git(self):
        before, after = b'a\0b', b'c\0d'
        units = extract(before, after, 'f')
        self.assertEqual(reconstruct(before, units, [u['id'] for u in units]), after)
        self.assertTrue(stage_verify(before, after, units))

if __name__ == '__main__': unittest.main()
