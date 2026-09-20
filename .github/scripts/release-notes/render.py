#!/usr/bin/env python3
"""Render deterministic bilingual GitHub release notes."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
import re

CATEGORY_ORDER = ("Added", "Changed", "Fixed", "Security", "Distribution", "Internal")
JAPANESE_CATEGORIES = {
    "Added": "追加",
    "Changed": "変更",
    "Fixed": "修正",
    "Security": "セキュリティ",
    "Distribution": "配布",
    "Internal": "内部変更",
}


def _note(value: str, number: int) -> str:
    lines = [line.strip() for line in value.splitlines() if line.strip()]
    text = " ".join(lines)
    if text.startswith("- "):
        text = text[2:].lstrip()
    text = re.sub(
        rf"\s*\(#{number}\)(?P<punct>[.!。！？])?\s*$",
        lambda match: match.group("punct") or "",
        text,
    )
    return text


def _groups(entries: list[dict], key: str) -> tuple[dict[str, list[dict]], list[dict]]:
    groups = {category: [] for category in CATEGORY_ORDER}
    breaking: list[dict] = []
    for entry in entries:
        if entry.get("breaking"):
            breaking.append(entry)
        elif entry.get("category") in groups:
            groups[entry["category"]].append(entry)
    for values in groups.values():
        values.sort(key=lambda item: item["number"])
    breaking.sort(key=lambda item: item["number"])
    return groups, breaking


def _render_language(entries: list[dict], text_key: str, *, japanese: bool) -> list[str]:
    groups, breaking = _groups(entries, text_key)
    lines: list[str] = []
    for category in CATEGORY_ORDER:
        if not groups[category]:
            continue
        heading = JAPANESE_CATEGORIES[category] if japanese else category
        lines.extend((f"#### {heading}", ""))
        for entry in groups[category]:
            lines.append(f"- {_note(entry[text_key], entry['number'])} (#{entry['number']})")
        lines.append("")
    if breaking:
        lines.extend(("#### 破壊的変更" if japanese else "#### Breaking Changes", ""))
        for entry in breaking:
            lines.append(f"- {_note(entry[text_key], entry['number'])} (#{entry['number']})")
        lines.append("")
    if not lines:
        lines.extend(
            (
                "利用者向けの変更はありません。" if japanese else "There are no user-facing changes.",
                "",
            )
        )
    return lines


def render(entries: list[dict], *, previous_tag: str | None, current_tag: str, repository: str) -> str:
    english = [dict(entry) for entry in entries]
    japanese = [dict(entry) for entry in entries]
    for entry in japanese:
        entry["japanese"] = entry.get("japanese", "")
    lines = ["## What's Changed", "", "### English", ""]
    lines.extend(_render_language(english, "english", japanese=False))
    lines.extend(("### 日本語", "", "> 日本語は英語版から自動生成された参考訳です。内容に差異がある場合は英語版を正とします。", ""))
    lines.extend(_render_language(japanese, "japanese", japanese=True))
    if previous_tag:
        changelog = f"https://github.com/{repository}/compare/{previous_tag}...{current_tag}"
    else:
        changelog = f"https://github.com/{repository}/releases/tag/{current_tag}"
    lines.extend((f"**Full Changelog**: {changelog}", ""))
    return "\n".join(lines)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--entries", type=Path, required=True)
    parser.add_argument("--japanese", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--previous-tag")
    parser.add_argument("--current-tag", required=True)
    parser.add_argument("--repository", required=True)
    args = parser.parse_args()
    canonical = json.loads(args.entries.read_text(encoding="utf-8"))
    derived = json.loads(args.japanese.read_text(encoding="utf-8"))
    entries = canonical.get("entries") if isinstance(canonical, dict) else canonical
    translations = derived.get("entries") if isinstance(derived, dict) else derived
    if not isinstance(entries, list) or not isinstance(translations, list):
        raise SystemExit("release-note entries must be JSON arrays")
    combined = []
    for source, translation in zip(entries, translations):
        item = dict(source)
        item["japanese"] = translation.get("japanese", "")
        combined.append(item)
    args.output.write_text(
        render(combined, previous_tag=args.previous_tag, current_tag=args.current_tag, repository=args.repository),
        encoding="utf-8",
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
