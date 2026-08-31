#!/usr/bin/env python3
"""Минимальная автоматическая проверка опечаток и лишних англицизмов в прозе."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PATTERNS = {
    r"\bboilerplate\b": "шаблонный код",
    r"\bmainstream(?:-язык\w*)?\b": "распространённый язык",
    r"\bproduction-ready\b": "готовый к эксплуатации",
    r"\breal-time\b": "реального времени",
    r"\bproduction\b": "рабочая среда",
    r"\bretry\b": "повтор",
    r"\bgraceful shutdown\b": "мягкое завершение",
    r"\bтрасс(?:а|ы|е|у|ой|ам|ами|ах)\b": "трассировка",
    r"\bтрейс\w*\b": "трассировка",
    r"\bтакж\b": "также",
    r"\bвкомпилиру\w*\b": "встраивается при компиляции",
    r"\bв рамкахх\b": "в рамках",
}

# В словаре английский эквивалент нужен для поиска документации. Точные значения
# протоколов и конфигурации исключаются функцией prose_lines вместе с кодом.
ALLOWED_BY_FILE = {
    "glossary.md": {r"\bretry\b"},
}


def prose_lines(text: str):
    fenced = False
    for number, line in enumerate(text.splitlines(), 1):
        if line.lstrip().startswith("```"):
            fenced = not fenced
            continue
        if fenced or "Real-time Transport Control Protocol" in line:
            continue
        line = re.sub(r"`[^`]*`", "", line)
        line = re.sub(r"https?://\S+", "", line)
        yield number, line


def main() -> int:
    errors: list[str] = []
    for path in sorted((ROOT / "docs").rglob("*.md")):
        allowed = ALLOWED_BY_FILE.get(path.name, set())
        for number, line in prose_lines(path.read_text(encoding="utf-8")):
            for pattern, replacement in PATTERNS.items():
                if pattern in allowed:
                    continue
                if re.search(pattern, line, flags=re.IGNORECASE):
                    errors.append(
                        f"{path.relative_to(ROOT)}:{number}: замените на «{replacement}»"
                    )
    if errors:
        print("Замечания к языку:", file=sys.stderr)
        print("\n".join(f"- {error}" for error in errors), file=sys.stderr)
        return 1
    print("Проверка частых опечаток и лишних англицизмов пройдена")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
