import copy
import json
import unittest
from pathlib import Path
import r67_compare


class R67IsolationContract(unittest.TestCase):
    def test_real_requests_change_only_inserted_system_paragraph(self):
        manifest = json.loads((r67_compare.BASE / 'run-3/frozen-inputs/manifest.json').read_text())
        for row in manifest:
            if row['arm'] != 'A':
                continue
            original = json.loads((r67_compare.BASE / 'run-3/frozen-inputs' / row['request_file']).read_text())
            snapshot = copy.deepcopy(original)
            changed = r67_compare.treatment(original)
            self.assertEqual(original, snapshot)
            changed['messages'][0]['content'] = changed['messages'][0]['content'].replace(r67_compare.RULE + '\n', '', 1)
            self.assertEqual(changed, original)

    def test_missing_or_repeated_safety_anchor_is_rejected(self):
        for content in ['', r67_compare.ANCHOR * 2]:
            with self.assertRaises(ValueError):
                r67_compare.treatment({'messages': [{'role': 'system', 'content': content}]})


if __name__ == '__main__':
    unittest.main()
