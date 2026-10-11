"""Safety tests for generated discovery objects and immutable fixture receipts."""
import json
from pathlib import Path
import tempfile
import unittest
from run import validate_discovery, repo_digest

class ContractTests(unittest.TestCase):
    def test_rejects_duplicate_unknown_and_nonsequential_ids(self):
        bad = ['{"purposes":[],"purposes":[]}',
               '{"purposes":[{"purpose_id":"P001","description":"a"},{"purpose_id":"P001","description":"b"}]}',
               '{"purposes":[{"purpose_id":"P009","description":"a"}]}',
               '{"purposes":[{"purpose_id":"P001","description":"a","file_id":"invented"}]}']
        for value in bad:
            with self.assertRaises(ValueError):
                validate_discovery(value)
        self.assertEqual(len(validate_discovery('{"purposes":[{"purpose_id":"P001","description":"change a"}]}')['purposes']),1)

    def test_fixture_receipt_detects_index_and_source_changes(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            (root/'.git').mkdir()
            (root/'.git/index').write_bytes(b'index')
            (root/'source.go').write_text('original')
            original=repo_digest(root)
            (root/'.git/index').write_bytes(b'changed')
            self.assertNotEqual(original,repo_digest(root))
            (root/'.git/index').write_bytes(b'index')
            (root/'source.go').write_text('changed')
            self.assertNotEqual(original,repo_digest(root))

if __name__=='__main__':
    unittest.main()
