#!/usr/bin/env python3

from __future__ import annotations

import importlib
import sys
import unittest

from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR))

collect = importlib.import_module("collect")
metadata = importlib.import_module("metadata")
render = importlib.import_module("render")
translate = importlib.import_module("translate")
validate = importlib.import_module("validate")


def body(note: str = "Fixes the Ollama guidance for `commiter doctor`.", category: str = "Fixed", breaking: str = "No") -> str:
    categories = "\n".join(
        f"- [{'x' if item == category else ' '}] {item}" for item in metadata.CATEGORIES
    )
    breaking_options = "\n".join(
        f"- [{'x' if item == breaking else ' '}] {item}" for item in metadata.BREAKING_VALUES
    )
    return f"""## Summary
summary

## Release note

{note}

## Release category

{categories}

## Breaking change

{breaking_options}
"""


class MetadataTests(unittest.TestCase):
    def test_parse_metadata_requires_exactly_one_choice(self):
        result = metadata.parse_metadata(body())
        self.assertEqual(result.category, "Fixed")
        self.assertFalse(result.breaking)

        invalid = body().replace("- [ ] Added", "- [x] Added")
        with self.assertRaisesRegex(metadata.MetadataError, "exactly one"):
            metadata.parse_metadata(invalid)

    def test_user_facing_none_is_rejected(self):
        with self.assertRaises(metadata.MetadataError):
            metadata.parse_metadata(body(note="None"))

    def test_internal_none_is_allowed(self):
        result = metadata.parse_metadata(body(note="None", category="Internal"))
        self.assertEqual(result.release_note, "None")


class CollectionTests(unittest.TestCase):
    def test_collects_only_commits_in_range_and_sorts(self):
        prs = [
            {"number": 4, "mergeCommit": {"oid": "in-range"}, "body": body(category="Added")},
            {"number": 3, "mergeCommit": {"oid": "out-of-range"}, "body": body(category="Fixed")},
            {"number": 5, "mergeCommit": {"oid": "in-range-2"}, "body": body(category="Fixed")},
        ]
        entries = collect.collect(prs, {"in-range", "in-range-2"})
        self.assertEqual([entry["number"] for entry in entries], [4, 5])

    def test_old_pr_without_metadata_fails_closed(self):
        prs = [{"number": 7, "mergeCommit": {"oid": "in-range"}, "body": "## Summary\nold"}]
        with self.assertRaisesRegex(metadata.MetadataError, "PR #7"):
            collect.collect(prs, {"in-range"})

    def test_collect_supports_rest_merge_commit_shape(self):
        prs = [{"number": 8, "merge_commit_sha": "in-range", "body": body(category="Internal", note="CI output is now deterministic.")}]
        entries = collect.collect(prs, {"in-range"})
        self.assertEqual(entries[0]["number"], 8)


class TranslationTests(unittest.TestCase):
    def test_protected_literals_are_restored(self):
        source = "Fixed `commiter doctor` when --dry-run is used (#12): https://example.test/v1.2.0"
        translated = translate.translate_text(source, lambda text: "修正 " + text)
        self.assertIn("`commiter doctor`", translated)
        self.assertIn("--dry-run", translated)
        self.assertIn("#12", translated)
        self.assertIn("v1.2.0", translated)
        self.assertIn("https://example.test/v1.2.0", translated)

    def test_placeholder_collision_is_safe(self):
        source = "Keep __RN_PROTECTED_0000__ and `flag`."
        translated = translate.translate_text(source, lambda text: text)
        self.assertEqual(translated, source)

    def test_long_input_is_rejected_before_model_call(self):
        class Tokenizer:
            def __call__(self, text, **kwargs):
                return {"input_ids": list(range(len(text)))}

        with self.assertRaisesRegex(ValueError, "too long"):
            translate.ensure_input_length(Tokenizer(), "x" * 5, max_tokens=4)


class ValidationAndRenderTests(unittest.TestCase):
    def test_translation_validation_preserves_identity_and_literals(self):
        english = [{"number": 12, "category": "Fixed", "breaking": False, "english": "Fix `--dry-run`."}]
        japanese = [{"number": 12, "category": "Fixed", "breaking": False, "japanese": "`--dry-run` を修正。"}]
        validate.validate_translation(english, japanese)
        with self.assertRaises(metadata.MetadataError):
            validate.validate_translation(english, [{**japanese[0], "number": 13}])

    def test_render_is_deterministic_and_omits_empty_categories(self):
        entries = [
            {"number": 5, "category": "Fixed", "breaking": False, "english": "Fix B", "japanese": "Bを修正"},
            {"number": 2, "category": "Added", "breaking": False, "english": "Add A", "japanese": "Aを追加"},
            {"number": 3, "category": "Internal", "breaking": True, "english": "Break C", "japanese": "Cを変更"},
        ]
        output = render.render(entries, previous_tag="v1.0.0", current_tag="v1.1.0", repository="org/repo")
        self.assertLess(output.index("#### Added"), output.index("#### Fixed"))
        self.assertIn("#### Breaking Changes", output)
        self.assertNotIn("#### Changed", output)
        self.assertNotIn("(#2) (#2)", output)
        self.assertEqual(output, render.render(entries, previous_tag="v1.0.0", current_tag="v1.1.0", repository="org/repo"))


if __name__ == "__main__":
    unittest.main()
