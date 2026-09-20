#!/usr/bin/env python3

from __future__ import annotations

import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

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

    def test_inline_policy_rejects_missing_release_note(self):
        result = self._run_policy(_policy_body(note="None"))
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("English Release note is required", result.stdout)


if __name__ == "__main__":
    unittest.main()
