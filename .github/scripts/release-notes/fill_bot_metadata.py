#!/usr/bin/env python3
"""Add default release metadata to bot PRs after read-only PR policy runs."""

from __future__ import annotations

import json
import os
import re
from urllib import request

BOT_AUTHORS = {"dependabot[bot]", "github-actions[bot]"}
RELEASE_SECTIONS = {"Release note", "Release category", "Breaking change"}
DEFAULT_METADATA = """## Release note

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


def fill(event: dict, repository: str, token: str, open_url=request.urlopen) -> bool:
    run = event.get("workflow_run") or {}
    prs = run.get("pull_requests") or []
    if run.get("event") != "pull_request_target" or run.get("conclusion") != "failure":
        return False
    if len(prs) != 1 or not isinstance(prs[0].get("number"), int):
        return False
    if not repository or not token or not isinstance(run.get("id"), int):
        raise ValueError("bot metadata workflow is missing its GitHub context")

    number = prs[0]["number"]
    url = f"https://api.github.com/repos/{repository}/pulls/{number}"
    headers = {
        "Accept": "application/vnd.github+json",
        "Authorization": f"Bearer {token}",
        "X-GitHub-Api-Version": "2026-03-10",
    }
    with open_url(request.Request(url, headers=headers), timeout=10) as response:
        pr = json.load(response)
    if (pr.get("user") or {}).get("login") not in BOT_AUTHORS or pr.get("state") != "open":
        return False
    if ((pr.get("base") or {}).get("repo") or {}).get("full_name") != repository:
        return False

    body = pr.get("body") or ""
    headings = {
        match.group(1).strip()
        for match in re.finditer(r"(?m)^##[ \t]+(.+?)[ \t]*$", body)
    }
    if headings & RELEASE_SECTIONS:
        return False

    updated_body = body + ("\n" if body.endswith("\n") else "\n\n") + DEFAULT_METADATA if body else DEFAULT_METADATA
    payload = json.dumps({"body": updated_body}).encode("utf-8")
    update = request.Request(
        url, data=payload, headers={**headers, "Content-Type": "application/json"}, method="PATCH"
    )
    with open_url(update, timeout=10) as response:
        updated = json.load(response)
    if updated.get("body") != updated_body:
        raise ValueError("GitHub did not return the updated PR body")

    rerun_url = f"https://api.github.com/repos/{repository}/actions/runs/{run['id']}/rerun"
    with open_url(request.Request(rerun_url, data=b"", headers=headers, method="POST"), timeout=10):
        pass
    return True


def main() -> int:
    with open(os.environ["GITHUB_EVENT_PATH"], encoding="utf-8") as stream:
        event = json.load(stream)
    try:
        changed = fill(event, os.environ.get("GITHUB_REPOSITORY", ""), os.environ.get("GH_TOKEN", ""))
    except (OSError, ValueError):
        print("::error::Bot Release metadata could not be filled or PR Policy could not be rerun.")
        return 1
    if changed:
        print("Added default bot Release metadata and reran PR Policy.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
