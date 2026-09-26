#!/usr/bin/env python3

from __future__ import annotations

import json
import io
from contextlib import redirect_stdout
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

SCRIPT_DIR = Path(__file__).resolve().parent
REPOSITORY_ROOT = SCRIPT_DIR.parents[2]
sys.path.insert(0, str(SCRIPT_DIR))

from test_release_notes import body


def _inline_pr_policy_script() -> str:
    workflow = (REPOSITORY_ROOT / ".github/workflows/pr-policy.yml").read_text(encoding="utf-8")
    marker = "          python3 - <<'PY'\n"
    start = workflow.index(marker) + len(marker)
    end = workflow.index("          PY\n", start)
    return "".join(
        line[10:] if line.startswith(" " * 10) else line for line in workflow[start:end].splitlines(keepends=True)
    )


def _policy_body(note: str = "Fixes the Ollama guidance for `commiter doctor`.") -> str:
    return f"""## Summary
Summary text

## Related issue
Closes #92

## Changes
- Add release note metadata validation.

## Verification
- [x] `git diff --check`

Skipped / not applicable:
go test, go vet, and go build are not applicable to this fixture-only check.

## Requirements impact
- [x] I believe this PR may affect requirements.
Notes: The release contract has been updated.

## Safety impact
Release metadata is validated before publishing.

## Release note
{note}

## Release category
- [ ] Added
- [ ] Changed
- [x] Fixed
- [ ] Security
- [ ] Distribution
- [ ] Internal
- [ ] None

## Breaking change
- [ ] Yes
- [x] No
"""


class PullRequestPolicyFixtureTests(unittest.TestCase):
    def _run_policy(self, body_text: str) -> subprocess.CompletedProcess[str]:
        event = {
            "pull_request": {
                "title": "ci: validate release note metadata",
                "body": body_text,
                "user": {"login": "contributor"},
                "draft": False,
            }
        }
        with tempfile.TemporaryDirectory() as directory:
            event_path = Path(directory) / "event.json"
            event_path.write_text(json.dumps(event), encoding="utf-8")
            environment = {"GITHUB_EVENT_PATH": str(event_path)}
            return subprocess.run(
                [sys.executable],
                input=_inline_pr_policy_script(),
                text=True,
                capture_output=True,
                env=environment,
            )

    def test_inline_policy_accepts_the_same_valid_fixture(self):
        result = self._run_policy(_policy_body())
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_inline_policy_accepts_crlf_body(self):
        result = self._run_policy(_policy_body().replace("\n", "\r\n"))
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_inline_policy_rejects_missing_release_note(self):
        result = self._run_policy(_policy_body(note="None"))
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("English Release note is required", result.stdout)

    def test_inline_policy_rejects_breaking_change_without_note(self):
        body_text = _policy_body(note="None").replace("- [x] Fixed", "- [ ] Fixed").replace(
            "- [ ] Internal", "- [x] Internal"
        ).replace("- [ ] Yes\n- [x] No", "- [x] Yes\n- [ ] No")
        result = self._run_policy(body_text)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Breaking changes require an English Release note", result.stdout)

    def _run_bot_policy(self, event_body: str, live_body: str, draft: bool = False):
        with tempfile.TemporaryDirectory() as directory:
            event_path = Path(directory) / "event.json"
            event_path.write_text(json.dumps({"pull_request": {
                "number": 42,
                "title": "build(deps): bump dependency",
                "body": event_body,
                "user": {"login": "dependabot[bot]"},
                "draft": draft,
            }}), encoding="utf-8")
            output = io.StringIO()
            calls = []

            def respond(req, timeout):
                calls.append(req)
                return io.BytesIO(json.dumps({"body": live_body}).encode())

            with patch.dict("os.environ", {
                "GITHUB_EVENT_PATH": str(event_path),
                "GITHUB_REPOSITORY": "owner/repo",
                "GH_TOKEN": "test-token",
            }), patch("urllib.request.urlopen", side_effect=respond), redirect_stdout(output):
                try:
                    exec(_inline_pr_policy_script(), {"__name__": "__main__"})
                    exit_code = 0
                except SystemExit as exc:
                    exit_code = exc.code
        return exit_code, output.getvalue(), calls

    def test_bot_without_release_metadata_fails_before_merge(self):
        for draft in (False, True):
            with self.subTest(draft=draft):
                exit_code, output, calls = self._run_bot_policy(
                    "Bumps a dependency.", "Bumps a dependency.", draft=draft
                )
                self.assertEqual(exit_code, 1)
                self.assertIn("Missing required section: ## Release note", output)
                self.assertEqual([req.get_method() for req in calls], ["GET"])

    def test_bot_with_reviewed_release_metadata_passes(self):
        release = """## Release note

None

## Release category

- [ ] Added
- [ ] Changed
- [ ] Fixed
- [ ] Security
- [ ] Distribution
- [x] Internal
- [ ] None

## Breaking change

- [ ] Yes
- [x] No
"""
        exit_code, output, calls = self._run_bot_policy(
            "Bumps a dependency.", "Bumps a dependency.\n\n" + release
        )
        self.assertEqual(exit_code, 0, output)
        self.assertEqual([req.get_method() for req in calls], ["GET"])

if __name__ == "__main__":
    unittest.main()
