#!/usr/bin/env python3
from __future__ import annotations

from pathlib import Path
import re
import sys

DOCUMENT_PAIRS = (
    ("README.md", "README.en.md"),
    ("docs/installation.md", "docs/installation.en.md"),
    ("docs/usage.md", "docs/usage.en.md"),
    ("docs/configuration.md", "docs/configuration.en.md"),
    ("docs/architecture.md", "docs/architecture.en.md"),
    ("docs/development.md", "docs/development.en.md"),
    ("docs/repository-map.md", "docs/repository-map.en.md"),
    ("docs/troubleshooting.md", "docs/troubleshooting.en.md"),
    ("CONTRIBUTING.md", "CONTRIBUTING.en.md"),
    ("SECURITY.md", "SECURITY.en.md"),
)

CYRILLIC = re.compile(r"[А-Яа-яЁё]")
LOCAL_LINK = re.compile(r"\[[^\]]+\]\(([^)]+)\)")


def structure(text: str) -> dict[str, object]:
    return {
        "heading levels": [len(item) for item in re.findall(r"^(#+) ", text, re.MULTILINE)],
        "bullet count": len(re.findall(r"^- ", text, re.MULTILINE)),
        "numbered-item count": len(re.findall(r"^\d+\. ", text, re.MULTILINE)),
        "table-row count": len(re.findall(r"^\|", text, re.MULTILINE)),
        "fence count": len(re.findall(r"^```", text, re.MULTILINE)),
    }


def fenced_blocks(text: str) -> list[tuple[str, str]]:
    return [
        (language, body.strip())
        for language, body in re.findall(
            r"^```([^\n]*)\n(.*?)^```$", text, re.MULTILINE | re.DOTALL
        )
    ]


def check_local_links(path: Path, text: str) -> list[str]:
    errors: list[str] = []
    for raw_target in LOCAL_LINK.findall(text):
        target = raw_target.split("#", 1)[0]
        if not target or re.match(r"^(https?://|ton://|mailto:)", target):
            continue
        if not (path.parent / target).exists():
            errors.append(f"{path}: broken local link: {target}")
    return errors


def main() -> int:
    errors: list[str] = []
    checked_paths: set[Path] = set()
    base_dir = Path(__file__).resolve().parent.parent

    for russian_name, english_name in DOCUMENT_PAIRS:
        russian_path = base_dir / russian_name
        english_path = base_dir / english_name
        if not russian_path.is_file() or not english_path.is_file():
            errors.append(f"missing documentation pair: {russian_path} <-> {english_path}")
            continue
        russian = russian_path.read_text(encoding="utf-8")
        english = english_path.read_text(encoding="utf-8")
        checked_paths.update((russian_path, english_path))

        for label, russian_value, english_value in (
            ("structure", structure(russian), structure(english)),
            ("fenced code blocks", fenced_blocks(russian), fenced_blocks(english)),
        ):
            if russian_value != english_value:
                errors.append(f"{russian_name} <-> {english_name}: {label} differs")
        if CYRILLIC.search(english):
            errors.append(f"{english_name}: Cyrillic text remains in the English document")

    for path in checked_paths:
        errors.extend(check_local_links(path, path.read_text(encoding="utf-8")))

    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    print(f"OK: {len(DOCUMENT_PAIRS)} Russian/English documentation pairs are in sync")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
