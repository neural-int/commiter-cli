#!/usr/bin/env python3
"""Parse and validate the release metadata in a pull request body."""

from __future__ import annotations

import re
from dataclasses import dataclass

CATEGORIES = (
    "Added",
    "Changed",
    "Fixed",
    "Security",
    "Distribution",
    "Internal",
    "None",
)
USER_FACING_CATEGORIES = frozenset(CATEGORIES[:-2])
BREAKING_VALUES = ("Yes", "No")

_COMMENT_RE = re.compile(r"<!--.*?-->", re.DOTALL)
_HEADING_RE = re.compile(r"(?m)^##[ \t]+(.+?)[ \t]*$")
_CHECKBOX_RE = re.compile(r"(?m)^\s*-[ \t]+\[([ xX])\][ \t]+(.+?)[ \t]*$")


@dataclass(frozen=True)
class ReleaseMetadata:
    release_note: str
    category: str
    breaking: bool


class MetadataError(ValueError):
    """Raised when release metadata is missing or ambiguous."""


def _sections(body: str) -> tuple[dict[str, str], set[str]]:
    matches = list(_HEADING_RE.finditer(body or ""))
    sections: dict[str, str] = {}
    duplicates: set[str] = set()
    for index, match in enumerate(matches):
        name = match.group(1).strip()
        start = match.end()
        end = matches[index + 1].start() if index + 1 < len(matches) else len(body)
        if name in sections:
            duplicates.add(name)
        sections[name] = body[start:end]
    return sections, duplicates


def _clean(value: str) -> str:
    return _COMMENT_RE.sub("", value).strip()


def _selected(section: str, options: tuple[str, ...], label: str) -> str:
    choices: list[str] = []
    for match in _CHECKBOX_RE.finditer(section):
        choice = match.group(2).strip()
        if choice in options and match.group(1).lower() == "x":
            choices.append(choice)
        elif choice not in options:
            raise MetadataError(f"{label} contains an invalid option: {choice}")
    if len(choices) != 1:
        raise MetadataError(f"{label} must select exactly one option")
    return choices[0]


def parse_metadata(body: str, *, pr_number: int | None = None) -> ReleaseMetadata:
    """Return validated release metadata from a PR body.

    The headings and checkbox values are deliberately exact. This keeps the
    release workflow fail-closed when a contributor deletes or renames a
    required section.
    """

    sections, duplicates = _sections(body)
    required = ("Release note", "Release category", "Breaking change")
    prefix = f"PR #{pr_number}: " if pr_number is not None else ""
    missing = [name for name in required if name not in sections]
    if missing:
        raise MetadataError(prefix + "missing section(s): " + ", ".join(missing))
    duplicated = [name for name in required if name in duplicates]
    if duplicated:
        raise MetadataError(prefix + "duplicate section(s): " + ", ".join(duplicated))

    try:
        category = _selected(sections["Release category"], CATEGORIES, "Release category")
        breaking_value = _selected(sections["Breaking change"], BREAKING_VALUES, "Breaking change")
    except MetadataError as error:
        raise MetadataError(prefix + str(error)) from error

    release_note = _clean(sections["Release note"])
    if not release_note:
        raise MetadataError(prefix + "English Release note must not be empty")
    if category in USER_FACING_CATEGORIES and release_note.casefold() == "none":
        raise MetadataError(
            prefix + f"English Release note is required for category {category}"
        )
    if release_note.casefold() == "none":
        release_note = "None"

    return ReleaseMetadata(
        release_note=release_note,
        category=category,
        breaking=breaking_value == "Yes",
    )
