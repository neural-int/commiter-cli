#!/usr/bin/env python3

from __future__ import annotations

import importlib
import io
import json
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parent))
fill_bot_metadata = importlib.import_module("fill_bot_metadata")
metadata = importlib.import_module("metadata")


def event(conclusion: str = "failure") -> dict:
    return {"workflow_run": {
        "id": 123,
        "event": "pull_request_target",
        "conclusion": conclusion,
        "pull_requests": [{"number": 42}],
    }}


class FillBotMetadataTests(unittest.TestCase):
    def run_fill(self, body: str, author: str = "dependabot[bot]"):
        calls = []

        def open_url(req, timeout):
            calls.append(req)
            if req.get_method() == "GET":
                payload = {
                    "user": {"login": author},
                    "state": "open",
                    "base": {"repo": {"full_name": "owner/repo"}},
                    "body": body,
                }
                return io.BytesIO(json.dumps(payload).encode())
            if req.get_method() == "PATCH":
                return io.BytesIO(req.data)
            return io.BytesIO()

        changed = fill_bot_metadata.fill(event(), "owner/repo", "test-token", open_url)
        return changed, calls

    def test_missing_metadata_is_appended_and_policy_is_rerun(self):
        changed, calls = self.run_fill("Bumps a dependency.")
        self.assertTrue(changed)
        self.assertEqual([req.get_method() for req in calls], ["GET", "PATCH", "POST"])
        updated = json.loads(calls[1].data)["body"]
        self.assertTrue(updated.startswith("Bumps a dependency.\n\n"))
        parsed = metadata.parse_metadata(updated)
        self.assertEqual((parsed.release_note, parsed.category, parsed.breaking), ("None", "Internal", False))
        self.assertTrue(calls[2].full_url.endswith("/actions/runs/123/rerun"))

        again_changed, again_calls = self.run_fill(updated)
        self.assertFalse(again_changed)
        self.assertEqual([req.get_method() for req in again_calls], ["GET"])

    def test_reviewed_or_partial_metadata_is_not_overwritten(self):
        for body in (
            fill_bot_metadata.DEFAULT_METADATA.replace("- [x] Internal", "- [ ] Internal").replace(
                "- [ ] Changed", "- [x] Changed"
            ).replace("## Release note\n\nNone", "## Release note\n\nUpdates dependencies."),
            "Bumps a dependency.\n\n## Release note\n\nNone",
        ):
            with self.subTest(body=body):
                changed, calls = self.run_fill(body)
                self.assertFalse(changed)
                self.assertEqual([req.get_method() for req in calls], ["GET"])

    def test_non_bot_or_successful_policy_is_not_modified(self):
        changed, calls = self.run_fill("Bumps a dependency.", author="contributor")
        self.assertFalse(changed)
        self.assertEqual([req.get_method() for req in calls], ["GET"])
        self.assertFalse(fill_bot_metadata.fill(event("success"), "owner/repo", "test-token", lambda *_: self.fail("API called")))

    def test_rerun_failure_does_not_report_success(self):
        def open_url(req, timeout):
            if req.get_method() == "GET":
                payload = {
                    "user": {"login": "dependabot[bot]"},
                    "state": "open",
                    "base": {"repo": {"full_name": "owner/repo"}},
                    "body": "Bumps a dependency.",
                }
                return io.BytesIO(json.dumps(payload).encode())
            if req.get_method() == "PATCH":
                return io.BytesIO(req.data)
            raise OSError("rerun failed")

        with self.assertRaises(OSError):
            fill_bot_metadata.fill(event(), "owner/repo", "test-token", open_url)


if __name__ == "__main__":
    unittest.main()
