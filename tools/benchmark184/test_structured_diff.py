import copy
import unittest
from structured_diff import project_payload, split_code


def block(start, end):
    return {"block_id": "B1", "diff_range": {"start_line": start, "end_line": end},
            "agent_history_candidates": [{"context_ref": "H1", "match": "exact"}]}


class StructuredDiffTests(unittest.TestCase):
    def test_preserves_code_symbols_whitespace_and_literal_escapes(self):
        raw = 'diff --git a/x b/x\n--- a/x\n+++ b/x\n@@ -1 +1,3 @@\n---old\n+++i\n+    regex = r"^<([^>]*)>$\\n"\n+\n'
        result = split_code(raw, [block(5, 8)])[0]
        self.assertEqual(result["deleted_code"], '--old\n')
        self.assertEqual(result["added_code"], '++i\n    regex = r"^<([^>]*)>$\\n"\n\n')
        self.assertNotIn("diff_range", result)

    def test_add_delete_and_missing_final_newline(self):
        raw = '@@ -1 +1 @@\n-old\n\\ No newline at end of file\n+new\n\\ No newline at end of file\n'
        result = split_code(raw, [block(2, 5)])[0]
        self.assertEqual((result["deleted_code"], result["added_code"]), ("old", "new"))
        self.assertEqual(split_code('@@ -0,0 +1 @@\n+x\n', [block(2, 2)])[0]["deleted_code"], "")
        self.assertEqual(split_code('@@ -1 +0,0 @@\n-x\n', [block(2, 2)])[0]["added_code"], "")

    def test_never_changes_host_payload_or_history(self):
        host = {"files": [{"file_id": "F1", "raw_diff": '@@ -1 +1 @@\n-a\n+b\n',
                            "change_blocks": [block(2, 3)]}], "agent_history_contexts": [{"final_responses": ["結果\n"]}]}
        original = copy.deepcopy(host)
        view, audit = project_payload(host)
        self.assertEqual(host, original)
        self.assertEqual(view["agent_history_contexts"], original["agent_history_contexts"])
        self.assertNotIn("raw_diff", view["files"][0])
        self.assertEqual(audit[0]["host_diff_range"], block(2, 3)["diff_range"])

    def test_refuses_context_overlap_and_missing_coverage(self):
        for raw, blocks in [
            ('@@ -1,2 +1,2 @@\n context\n-a\n+b\n', [block(3, 4)]),
            ('@@ -1 +1 @@\n-a\n+b\n', [block(2, 2)]),
            ('@@ -1 +1 @@\n-a\n+b\n', [block(2, 3), block(2, 3)]),
        ]:
            with self.assertRaises(ValueError):
                split_code(raw, blocks)

    def test_unicode_line_separator_is_not_a_diff_line_separator(self):
        result = split_code('@@ -0,0 +1 @@\n+a\u2028b\n', [block(2, 2)])[0]
        self.assertEqual(result["added_code"], 'a\u2028b\n')
