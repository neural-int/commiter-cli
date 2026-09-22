#!/usr/bin/env python3
"""Validate canonical metadata and derived Japanese release notes."""

from __future__ import annotations

import argparse
from collections import Counter
import json
from pathlib import Path
import sys

from metadata import MetadataError
from translate import _literal_matches


def validate_translation(english: list[dict], japanese: list[dict]) -> None:
    if len(english) != len(japanese):
        raise MetadataError(
            f"English/Japanese entry count differs: {len(english)} != {len(japanese)}"
        )
    expected_keys = [(item.get("number"), item.get("category"), item.get("breaking")) for item in english]
    actual_keys = [(item.get("number"), item.get("category"), item.get("breaking")) for item in japanese]
    if expected_keys != actual_keys:
        raise MetadataError("English/Japanese entry identity or category mapping differs")

    for index, (source, derived) in enumerate(zip(english, japanese), start=1):
        source_text = source.get("english")
        translated_text = derived.get("japanese")
        if not isinstance(source_text, str) or not source_text.strip():
            raise MetadataError(f"English entry {index} is empty")
        if not isinstance(translated_text, str) or not translated_text.strip():
            raise MetadataError(f"Japanese entry {index} is empty")
        expected_literals = Counter(literal for _, _, literal in _literal_matches(source_text))
        actual_literals = Counter(literal for _, _, literal in _literal_matches(translated_text))
        if expected_literals != actual_literals:
            missing = expected_literals - actual_literals
            changed = actual_literals - expected_literals
            details = []
            if missing:
                details.append("missing " + ", ".join(sorted(missing)))
            if changed:
                details.append("changed " + ", ".join(sorted(changed)))
            raise MetadataError(f"PR #{source.get('number')}: protected literal validation failed ({'; '.join(details)})")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--english", type=Path, required=True)
    parser.add_argument("--japanese", type=Path, required=True)
    args = parser.parse_args()
    english_payload = json.loads(args.english.read_text(encoding="utf-8"))
    japanese_payload = json.loads(args.japanese.read_text(encoding="utf-8"))
    english = english_payload.get("entries") if isinstance(english_payload, dict) else english_payload
    japanese = japanese_payload.get("entries") if isinstance(japanese_payload, dict) else japanese_payload
    if not isinstance(english, list) or not isinstance(japanese, list):
        print("::error::release-note entries must be JSON arrays", file=sys.stderr)
        return 1
    try:
        validate_translation(english, japanese)
    except MetadataError as error:
        print(f"::error::{error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
