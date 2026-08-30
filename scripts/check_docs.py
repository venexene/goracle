#!/usr/bin/env python3
"""Проверка локальных ссылок, блоков кода, заголовков и меню MkDocs."""

from __future__ import annotations

import re
import sys
from collections import Counter
from pathlib import Path
from urllib.parse import unquote

from markdown.extensions.toc import slugify_unicode

ROOT = Path(__file__).resolve().parents[1]
DOCS = ROOT / "docs"

LINK = re.compile(r"(?<!!)\[[^]]*]\(([^)]+)\)")
NAV = re.compile(r"^\s*-\s+[^:]+:\s+([^#'\"{}][^#]*)$")
HEADING = re.compile(r"^(#{1,6})\s+(.+?)\s*$")
EXPLICIT_ID = re.compile(r"\bid=[\"']([^\"']+)[\"']")


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


def fragment(raw: str) -> str:
    target = raw.strip()
    if target.startswith("<") and ">" in target:
        target = target[1:target.index(">")]
    else:
        target = re.split(r"\s+['\"]", target, maxsplit=1)[0]
    return unquote(target.split("#", 1)[1]) if "#" in target else ""


def markdown_anchors(path: Path) -> set[str]:
    anchors: set[str] = set()
    counts: Counter[str] = Counter()
    for line in path.read_text(encoding="utf-8").splitlines():
        anchors.update(EXPLICIT_ID.findall(line))
        match = HEADING.match(line)
        if match is None:
            continue
        title = re.sub(r"[`*_]", "", match.group(2))
        title = re.sub(r"\s+\{[^}]*}\s*$", "", title)
        base = slugify_unicode(title, "-")
        suffix = counts[base]
        anchors.add(base if suffix == 0 else f"{base}_{suffix}")
        counts[base] += 1
    return anchors


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

        parents: list[str] = []
        heading_paths: list[tuple[str, ...]] = []
        for line in text.splitlines():
            if not (match := HEADING.match(line)):
                continue
            level = len(match.group(1))
            title = match.group(2).strip().casefold()
            parents = parents[:level - 1]
            heading_paths.append((*parents, title))
            parents.append(title)
        duplicates = [path[-1] for path, count in Counter(heading_paths).items() if count > 1]
        if duplicates:
            warnings.append(f"{relative}: повтор заголовка: {', '.join(duplicates)}")

        for match in LINK.finditer(text):
            target = local_target(path, match.group(1))
            if target is not None and not target.exists():
                errors.append(f"{relative}: нет локальной цели {match.group(1)!r}")
            elif (anchor := fragment(match.group(1))) and target is not None \
                    and target.suffix.casefold() == ".md" \
                    and anchor not in markdown_anchors(target):
                errors.append(f"{relative}: нет якоря #{anchor} в {target.relative_to(ROOT)}")

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
