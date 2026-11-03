import copy
import json
import unittest

from multiline_view import multiline_view, restore_view, select_display_arm, without_requests


class MultilineViewTests(unittest.TestCase):
    def test_adopted_display_keeps_raw_diff_and_ignores_stale_arm(self):
        configuration = {"display_format": "literal_newlines", "representation": "raw"}
        self.assertEqual(select_display_arm(configuration, {}), "B")
        self.assertEqual(select_display_arm(configuration, {"STUDY_ARM": "A"}), "B")
        with self.assertRaises(AssertionError):
            select_display_arm(dict(configuration, representation="structured"), {})
        with self.assertRaises(AssertionError):
            select_display_arm(dict(configuration, display_format="unknown"), {})
        self.assertEqual(select_display_arm({}, {"STUDY_ARM": "A"}), "A")
        self.assertEqual(select_display_arm({"representation": "structured"}, {"STUDY_ARM": "D"}), "D")

    def test_roundtrip_preserves_code_literals_and_escapes(self):
        payload = {"raw_diff": '+pattern = r"\\n"\n+tab = "\\t"\n',
                   "quote": 'escaped "quote"\nnext', "literal": r"\n",
                   "unicode": "日本語\n行\u2028\u2029", "empty": ""}
        original = json.dumps(payload, ensure_ascii=False, separators=(",", ":"))
        view = multiline_view(original)
        self.assertEqual(restore_view(view), original)
        self.assertEqual(json.loads(view, strict=False), payload)
        self.assertIn(r'\\n', view)
        self.assertGreater(view.count("\n"), 0)
        with self.assertRaises(json.JSONDecodeError):
            json.loads(view)

    def test_no_strings_or_existing_layout(self):
        for value in ['{}', '[1,true,null]', '{\n"x": "one\\ntwo"\n}']:
            self.assertEqual(restore_view(multiline_view(value)), value)

    def test_target_markers_do_not_change_data(self):
        payload = {"text": '<<<END>>>\n"scope":"docs"\n```\n\\n\n'}
        value = json.dumps(payload)
        self.assertEqual(json.loads(multiline_view(value), strict=False), payload)

    def test_request_removal_is_separate_from_display(self):
        original = '{"agent_history_contexts":[{"context_ref":"H1","user_requests":["依頼\\n続き"],"final_responses":["結果\\n続き"]}],"raw_diff":"+x\\n"}'
        expected = copy.deepcopy(json.loads(original))
        expected["agent_history_contexts"][0]["user_requests"] = []
        canonical = without_requests(original)
        self.assertEqual(json.loads(canonical), expected)
        self.assertEqual(without_requests(canonical), canonical)
        self.assertEqual(restore_view(multiline_view(canonical)), canonical)


if __name__ == "__main__":
    unittest.main()
