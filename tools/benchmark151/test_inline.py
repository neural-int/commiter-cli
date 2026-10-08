import itertools
import unittest
from inline import extract, MAX_INLINE_BYTES
from evaluate import reconstruct, stage_verify
from remeasure import matching_subsets, subsets, coverage


class InlineContracts(unittest.TestCase):
    def test_unicode_insert_delete_and_all_subsets_match_exhaustive(self):
        for before, after in ((b'ab\n', b'cd\n'), ('猫 赤\n'.encode(), '犬 青\n'.encode()),
                              (b'x=1,y=2\n', b'x=12,y=3\n'), (b'x=123,y=2\n', b'x=1,y=3\n')):
            units = extract(before, after, 'f')
            coverage(before, after, units)
            for u in units:
                for k in ('old_span', 'new_span'):
                    data = before if k == 'old_span' else after
                    data[:u[k][0]].decode('utf8')
                    data[:u[k][1]].decode('utf8')
            ids = [u['id'] for u in units]
            for choice in subsets(ids):
                target = reconstruct(before, units, choice)
                exhaustive = [s for s in subsets(ids) if reconstruct(before, units, s) == target]
                actual = matching_subsets(before, units, target)
                self.assertEqual(len(actual), min(len(exhaustive), 2))
                self.assertTrue(all(s in exhaustive for s in actual))
            self.assertTrue(stage_verify(before, after, units))

    def test_fail_closed_fragmentation_and_long_line(self):
        with self.assertRaisesRegex(ValueError, 'unit_budget'):
            extract(b'a'*257, b'b'*257, 'f')
        with self.assertRaisesRegex(ValueError, 'inline_byte_budget'):
            extract(b'a'*(MAX_INLINE_BYTES+1), b'b'*(MAX_INLINE_BYTES+1), 'f')

    def test_invalid_utf8_retains_atomic_bytes(self):
        before, after = b'\xff x\n', b'\xfe y\n'
        units = extract(before, after, 'f')
        self.assertEqual(len(units), 1)
        self.assertEqual(reconstruct(before, units, [units[0]['id']]), after)
        self.assertTrue(stage_verify(before, after, units))


if __name__ == '__main__':
    unittest.main()
