#!/usr/bin/env python3

from __future__ import annotations

import importlib
import io
import json
import sys
import unittest
from unittest.mock import patch

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

    def test_unquoted_command_short_flag_and_config_key_are_masked(self):
        source = "Run commiter doctor -v with analysis.preserve_outside_staged."
        masked, _ = translate.protect_literals(source)
        for literal in ("commiter doctor", "-v", "analysis.preserve_outside_staged"):
            self.assertNotIn(literal, masked)
        self.assertEqual(translate.translate_text(source, lambda text: text), source)

    def test_cloud_request_sends_only_masked_note_to_nmt(self):
        response = {"data": {"translations": [{"model": "nmt", "translatedText": "修正 __RN_PROTECTED_0000__。"}]}}
        with patch.object(translate.request, "urlopen", return_value=io.BytesIO(json.dumps(response).encode())) as open_url:
            result = translate.translate_text("Fix `commiter doctor`.", translate._cloud_translator("test-key"))
        self.assertEqual(result, "修正 `commiter doctor`。")
        sent = open_url.call_args.args[0]
        self.assertEqual(sent.full_url, translate.TRANSLATE_URL)
        self.assertEqual(sent.get_header("X-goog-api-key"), "test-key")
        self.assertEqual(json.loads(sent.data), {"q": "Fix __RN_PROTECTED_0000__.", "source": "en", "target": "ja", "format": "text", "model": "nmt"})

    def test_cloud_translation_fails_closed(self):
        with self.assertRaisesRegex(RuntimeError, "GOOGLE_TRANSLATE_API_KEY"):
            translate._cloud_translator("")
        response = {"data": {"translations": [{"model": "nmt", "translatedText": "修正。"}]}}
        with patch.object(translate.request, "urlopen", return_value=io.BytesIO(json.dumps(response).encode())):
            with self.assertRaisesRegex(ValueError, "placeholder missing"):
                translate.translate_text("Fix `commiter doctor`.", translate._cloud_translator("test-key"))


class ValidationAndRenderTests(unittest.TestCase):
    def test_translation_validation_preserves_identity_and_literals(self):
        english = [{"number": 12, "category": "Fixed", "breaking": False, "english": "Fix `--dry-run`."}]
        japanese = [{"number": 12, "category": "Fixed", "breaking": False, "japanese": "`--dry-run` を修正。"}]
        validate.validate_translation(english, japanese)
        with self.assertRaises(metadata.MetadataError):
            validate.validate_translation(english, [{**japanese[0], "number": 13}])

    def test_translation_validation_rejects_changed_unquoted_literals(self):
        source = "Run commiter doctor -v with analysis.preserve_outside_staged."
        english = [{"number": 92, "category": "Fixed", "breaking": False, "english": source}]
        for changed in (
            source.replace("commiter doctor", "commiter setup"),
            source.replace("-v", "-h"),
            source.replace("analysis.preserve_outside_staged", "analysis.preserve_outside_stage"),
        ):
            with self.subTest(changed=changed), self.assertRaisesRegex(metadata.MetadataError, "protected literal"):
                validate.validate_translation(english, [{"number": 92, "category": "Fixed", "breaking": False, "japanese": changed}])

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
