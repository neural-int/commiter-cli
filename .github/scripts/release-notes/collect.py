#!/usr/bin/env python3
"""Collect canonical release note entries for merged pull requests."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
import sys

from metadata import MetadataError, parse_metadata

_CATEGORY_ORDER = {
    "Added": 0,
    "Changed": 1,
    "Fixed": 2,
    "Security": 3,
    "Distribution": 4,
    "Internal": 5,
    "None": 6,
}


def _load(path: Path):
    with path.open(encoding="utf-8") as stream:
        return json.load(stream)


def _merge_oid(pr: dict) -> str | None:
    merge_commit = pr.get("mergeCommit") or pr.get("merge_commit")
    if not merge_commit:
        merge_commit = pr.get("merge_commit_sha")
    if isinstance(merge_commit, dict):
        return merge_commit.get("oid") or merge_commit.get("sha")
    return merge_commit if isinstance(merge_commit, str) else None


def collect(prs: list[dict], release_commits: set[str]) -> list[dict]:
    """Filter PRs to the tag range and return deterministic canonical entries."""

    entries: list[dict] = []
    seen: set[int] = set()
    errors: list[str] = []
    for pr in prs:
        number = pr.get("number")
        if not isinstance(number, int) or number in seen:
            continue
        if _merge_oid(pr) not in release_commits:
            continue
        seen.add(number)
        try:
            metadata = parse_metadata(pr.get("body") or "", pr_number=number)
        except MetadataError as error:
            errors.append(str(error))
            continue
        if metadata.category == "None" or metadata.release_note == "None":
            continue
        entries.append(
            {
                "number": number,
                "category": metadata.category,
                "breaking": metadata.breaking,
                "english": metadata.release_note,
            }
        )
    if errors:
        raise MetadataError("release-note metadata is incomplete.\n" + "\n".join(errors))
    entries.sort(key=lambda item: (_CATEGORY_ORDER[item["category"]], item["number"]))
    return entries


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--prs", type=Path, required=True)
    parser.add_argument("--commits", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()

    prs = _load(args.prs)
    if not isinstance(prs, list):
        raise SystemExit("PR input must be a JSON array")
    commits = {
        line.strip()
        for line in args.commits.read_text(encoding="utf-8").splitlines()
        if line.strip()
    }
    try:
        entries = collect(prs, commits)
    except MetadataError as error:
        print(f"::error::{error}", file=sys.stderr)
        return 1
    args.output.write_text(
        json.dumps({"entries": entries}, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
