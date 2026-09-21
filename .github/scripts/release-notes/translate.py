#!/usr/bin/env python3
"""Translate release notes with Cloud Translation NMT while preserving literals."""

from __future__ import annotations

import argparse
from collections.abc import Callable, Iterable
import json
import os
from pathlib import Path
import re
from urllib import error, request

TRANSLATE_URL = "https://translation.googleapis.com/language/translate/v2"

_PROTECTED_PATTERNS = (
    re.compile(r"__RN_PROTECTED_[0-9]+__"),
    re.compile(r"`[^`\n]+`"),
    re.compile(r"https?://[^\s)]+"),
    re.compile(r"(?<![A-Za-z0-9])#[0-9]+(?![A-Za-z0-9])"),
    re.compile(r"(?<![A-Za-z0-9])v[0-9]+(?:\.[0-9]+)+(?:-[0-9A-Za-z.-]+)?(?![A-Za-z0-9])"),
    re.compile(r"(?<![A-Za-z0-9])--[A-Za-z0-9][A-Za-z0-9-]*(?:=[A-Za-z0-9_.-]+)?"),
    re.compile(r"(?<!\w)(?:\.{0,2}/)?[A-Za-z0-9_.-]+(?:/[A-Za-z0-9_.-]+)+(?!\w)"),
    re.compile(r"(?<![A-Za-z0-9])[A-Z][A-Z0-9_]{2,}(?![A-Za-z0-9])"),
)
_PROTECTED_NAMES = ("commiter", "Ollama", "Homebrew", "GitHub")


def _literal_matches(text: str) -> list[tuple[int, int, str]]:
    matches: list[tuple[int, int, str]] = []
    for pattern in _PROTECTED_PATTERNS:
        matches.extend((match.start(), match.end(), match.group(0)) for match in pattern.finditer(text))
    for name in _PROTECTED_NAMES:
        pattern = re.compile(re.escape(name))
        matches.extend((match.start(), match.end(), match.group(0)) for match in pattern.finditer(text))
    selected: list[tuple[int, int, str]] = []
    for start, end, literal in sorted(matches, key=lambda item: (item[0], -(item[1] - item[0]))):
        if any(start < existing_end and end > existing_start for existing_start, existing_end, _ in selected):
            continue
        selected.append((start, end, literal))
    return selected


def protect_literals(text: str) -> tuple[str, list[tuple[str, str]]]:
    """Replace protected literals with deterministic collision-safe tokens."""

    matches = _literal_matches(text)
    if not matches:
        return text, []
    replacements: list[tuple[str, str]] = []
    occupied = set(re.findall(r"__RN_PROTECTED_[0-9]+__", text))
    for index, (_, _, literal) in enumerate(matches):
        token_index = index
        token = f"__RN_PROTECTED_{token_index:04d}__"
        while token in occupied:
            token_index += 1
            token = f"__RN_PROTECTED_{token_index:04d}__"
        occupied.add(token)
        replacements.append((token, literal))
    masked = text
    for (start, end, _), (token, _) in reversed(list(zip(matches, replacements))):
        masked = masked[:start] + token + masked[end:]
    return masked, replacements


def restore_literals(text: str, literals: Iterable[tuple[str, str]]) -> str:
    restored = text
    for token, literal in literals:
        if restored.count(token) != 1:
            raise ValueError(f"protected literal placeholder missing or duplicated: {token}")
        restored = restored.replace(token, literal)
    allowed_literal_tokens = {
        literal for _, literal in literals if re.fullmatch(r"__RN_PROTECTED_[0-9]+__", literal)
    }
    unknown_tokens = [
        match.group(0)
        for match in re.finditer(r"__RN_PROTECTED_[0-9]+__", restored)
        if match.group(0) not in allowed_literal_tokens
    ]
    if unknown_tokens:
        raise ValueError("unknown protected literal placeholder remains after translation")
    return restored


def translate_text(text: str, translator: Callable[[str], str]) -> str:
    masked, literals = protect_literals(text)
    translated = translator(masked).strip()
    if not translated:
        raise ValueError("translation returned an empty result")
    return restore_literals(translated, literals).strip()


def _cloud_translator(api_key: str) -> Callable[[str], str]:
    if not api_key:
        raise RuntimeError("GOOGLE_TRANSLATE_API_KEY is required for release-note translation")

    def translate(text: str) -> str:
        payload = json.dumps({"q": text, "source": "en", "target": "ja", "format": "text", "model": "nmt"}).encode("utf-8")
        translation_request = request.Request(
            TRANSLATE_URL,
            data=payload,
            headers={"Content-Type": "application/json; charset=utf-8", "X-goog-api-key": api_key},
            method="POST",
        )
        try:
            with request.urlopen(translation_request, timeout=30) as response:
                result = json.load(response)
        except error.HTTPError as exc:
            raise RuntimeError(f"Cloud Translation request failed with HTTP {exc.code}") from None
        except error.URLError as exc:
            raise RuntimeError("Cloud Translation request failed") from None
        translations = result.get("data", {}).get("translations", []) if isinstance(result, dict) else []
        if not isinstance(translations, list) or len(translations) != 1:
            raise ValueError("Cloud Translation returned an invalid translation count")
        item = translations[0]
        if not isinstance(item, dict) or item.get("model") != "nmt" or not isinstance(item.get("translatedText"), str):
            raise ValueError("Cloud Translation returned an invalid NMT response")
        return item["translatedText"]

    return translate


def translate_entries(entries: list[dict], translator: Callable[[str], str]) -> list[dict]:
    translated: list[dict] = []
    for entry in entries:
        item = dict(entry)
        item["japanese"] = translate_text(entry["english"], translator)
        translated.append(item)
    return translated


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    payload = json.loads(args.input.read_text(encoding="utf-8"))
    entries = payload.get("entries") if isinstance(payload, dict) else payload
    if not isinstance(entries, list):
        raise SystemExit("canonical release notes must contain an entries array")
    translated = translate_entries(entries, _cloud_translator(os.environ.get("GOOGLE_TRANSLATE_API_KEY", ""))) if entries else []
    args.output.write_text(
        json.dumps({"entries": translated}, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
