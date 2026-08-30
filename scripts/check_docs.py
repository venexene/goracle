#!/usr/bin/env python3
"""Проверка локальных ссылок, блоков кода, заголовков и меню MkDocs."""

from __future__ import annotations

import re
import sys
from collections import Counter
from pathlib import Path
from urllib.parse import unquote

ROOT = Path(__file__).resolve().parents[1]
DOCS = ROOT / "docs"

LINK = re.compile(r"(?<!!)\[[^]]*]\(([^)]+)\)")
NAV = re.compile(r"^\s*-\s+[^:]+:\s+([^#'\"{}][^#]*)$")
HEADING = re.compile(r"^(#{1,6})\s+(.+?)\s*$")


def local_target(source: Path, raw: str) -> Path | None:
    target = raw.strip()
    if target.startswith("<") and ">" in target:
        target = target[1:target.index(">")]
    else:
        target = re.split(r"\s+['\"]", target, maxsplit=1)[0]
    if not target or target.startswith(("#", "http://", "https://", "mailto:")):
        return None
    path = unquote(target.split("#", 1)[0])
    if not path or Path(path).suffix.casefold() not in {
        ".md", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp"
    }:
        return None
    return (source.parent / path).resolve()


def main() -> int:
    errors: list[str] = []
    warnings: list[str] = []
    markdown = sorted(DOCS.rglob("*.md"))
    for path in markdown:
        text = path.read_text(encoding="utf-8")
        relative = path.relative_to(ROOT)

        fences = sum(line.lstrip().startswith("```") for line in text.splitlines())
        if fences % 2:
            errors.append(f"{relative}: непарный блок кода")

        headings = [match.group(2).strip().casefold() for line in text.splitlines()
                    if (match := HEADING.match(line))]
        duplicates = [heading for heading, count in Counter(headings).items() if count > 1]
        if duplicates:
            warnings.append(f"{relative}: повтор заголовка: {', '.join(duplicates)}")

        for match in LINK.finditer(text):
            target = local_target(path, match.group(1))
            if target is not None and not target.exists():
                errors.append(f"{relative}: нет локальной цели {match.group(1)!r}")

    config = (ROOT / "mkdocs.yml").read_text(encoding="utf-8")
    nav_files: set[Path] = set()
    in_nav = False
    for line in config.splitlines():
        if line == "nav:":
            in_nav = True
            continue
        if in_nav and line and not line.startswith(" "):
            break
        if in_nav and (match := NAV.match(line)):
            value = match.group(1).strip().strip("'\"")
            if value.endswith(".md"):
                target = DOCS / value
                nav_files.add(target.resolve())
                if not target.exists():
                    errors.append(f"mkdocs.yml: нет файла меню {value!r}")

    missing = [path.relative_to(ROOT) for path in markdown if path.resolve() not in nav_files]
    if missing:
        errors.append("в меню отсутствуют: " + ", ".join(map(str, missing)))

    if warnings:
        print("Предупреждения документации:")
        for warning in warnings:
            print(f"- {warning}")
    if errors:
        print("Ошибки документации:", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1
    print(f"Проверено файлов Markdown: {len(markdown)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
