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
FUNDAMENTALS = DOCS / "go-fundamentals"
FUNDAMENTALS_ROUTES = {
    FUNDAMENTALS / "Компиляция и оптимизации Go" / "compilation.md",
    FUNDAMENTALS / "Планировщик Go" / "scheduler.md",
    FUNDAMENTALS / "Память в Go" / "memory.md",
    FUNDAMENTALS / "Сборщик мусора в Go" / "garbage-collector.md",
}


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

        if path == FUNDAMENTALS or FUNDAMENTALS in path.parents:
            if "**Уровень:**" in text:
                errors.append(f"{relative}: в статье указана аудитория или уровень")

            first_section = text.find("\n## ")
            upper = text[:first_section] if first_section != -1 else text
            if first_section == -1 or first_section == 0 or text[first_section - 1] != "\n":
                errors.append(f"{relative}: после верхнего блока нужна пустая строка")
            upper_fields = [
                "**Проверено:**",
                "**Перед чтением:**",
                "**После чтения:**",
                "**Практика:**",
            ]
            positions = [upper.find(field) for field in upper_fields]
            if any(position == -1 for position in positions):
                errors.append(f"{relative}: неполный верхний блок")
            elif positions != sorted(positions):
                errors.append(f"{relative}: нарушен порядок полей верхнего блока")
            if "practice.md" not in upper:
                errors.append(f"{relative}: верхний блок не ведёт в общий практикум")

            lower_sections = ["Краткий итог", "Продолжение", "Источники"]
            lower_positions = [text.find(f"\n## {section}\n") for section in lower_sections]
            if any(position == -1 for position in lower_positions):
                errors.append(f"{relative}: неполный нижний блок")
            elif lower_positions != sorted(lower_positions):
                errors.append(f"{relative}: нарушен порядок разделов нижнего блока")
            h2 = re.findall(r"^## (.+)$", text, re.MULTILINE)
            for section in lower_sections:
                if h2.count(section) > 1:
                    errors.append(f"{relative}: раздел «{section}» указан несколько раз")
            if h2 and h2[-1] != "Источники":
                errors.append(f"{relative}: последний раздел должен называться «Источники»")
            if re.search(r"^## (?:Практика|Практические задания|Самопроверка)$", text, re.MULTILINE):
                errors.append(f"{relative}: задания должны находиться в общем практикуме")

            if re.search(r"^#{2,4} \d+\. ", text, re.MULTILINE):
                errors.append(f"{relative}: заголовки не должны нумероваться вручную")

            if path in FUNDAMENTALS_ROUTES:
                route_sections = ["О разделе", "Материалы", "Рекомендуемый порядок"]
                route_positions = [text.find(f"\n## {section}\n") for section in route_sections]
                if any(position == -1 for position in route_positions):
                    errors.append(f"{relative}: неполная структура учебного маршрута")
                elif h2[:3] != route_sections:
                    errors.append(f"{relative}: маршрут должен начинаться с единых разделов")

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
