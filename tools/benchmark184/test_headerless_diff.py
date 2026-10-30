import json
import unittest
from headerless_diff import project_content


class HeaderlessTests(unittest.TestCase):
    def test_headers_removed_but_code_signs_and_ranges_preserved(self):
        raw = 'diff --git a/x b/x\n--- a/x\n+++ b/x\n@@ -1 +1 @@\n---old\n+++new\n'
        value = {"files": [{"file_id": "F1", "raw_diff": raw, "change_blocks": [{"block_id": "B1", "diff_range": {"start_line": 5, "end_line": 6}}]}]}
        original = json.dumps(value, separators=(",", ":"))
        actual, audit = project_content(original)
        result = json.loads(actual)["files"][0]
        self.assertEqual(result["raw_diff"], '---old\n+++new\n')
        self.assertEqual(result["change_blocks"][0]["diff_range"], {"start_line": 1, "end_line": 2})
        self.assertEqual(json.loads(original), value)
        self.assertTrue(audit[0]["code_and_diff_markers_identical"])
        kept, audit = project_content(original, keep_hunks=True)
        kept_file = json.loads(kept)["files"][0]
        self.assertEqual(kept_file["raw_diff"], '@@ -1 +1 @@\n---old\n+++new\n')
        self.assertEqual(kept_file["change_blocks"][0]["diff_range"], {"start_line": 2, "end_line": 3})
        self.assertTrue(audit[0]["hunk_headers_retained"])
