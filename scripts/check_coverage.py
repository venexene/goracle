#!/usr/bin/env python3
"""Проверка соответствия тематических глав исполняемым примерам."""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
EXAMPLES = ROOT / "examples"
COVERAGE = EXAMPLES / "coverage.json"
PRACTICE = ROOT / "docs" / "practice.md"
FUNCTION = re.compile(r"^func\s+((?:Test|Benchmark|Example)\w*)\s*\(", re.MULTILINE)
PRACTICE_LINK = re.compile(r"\]\((?:\.\./)*practice\.md#([^)]+)\)")
SUMMARY = re.compile(r"^##+ (?:Краткий итог|Итоги|Итог|Выводы)", re.MULTILINE)


def available_checks() -> set[str]:
    checks: set[str] = set()
    for path in sorted(EXAMPLES.glob("*/*_test.go")):
        package = path.parent.name
        text = path.read_text(encoding="utf-8")
        checks.update(f"{package}.{name}" for name in FUNCTION.findall(text))
    return checks


def main() -> int:
    errors: list[str] = []
    entries = json.loads(COVERAGE.read_text(encoding="utf-8"))
    checks = available_checks()
    practice_text = PRACTICE.read_text(encoding="utf-8")
    practice_ids = set(re.findall(r'<a\s+id="([^"]+)"', practice_text))

    if len(entries) != 32:
        errors.append(f"ожидалось 32 темы, найдено {len(entries)}")

    topics = [entry.get("topic") for entry in entries]
    documents = [entry.get("document") for entry in entries]
    if len(topics) != len(set(topics)):
        errors.append("названия тем должны быть уникальными")
    if len(documents) != len(set(documents)):
        errors.append("основные документы тем должны быть уникальными")

    for entry in entries:
        topic = entry.get("topic", "<без названия>")
        document = ROOT / entry.get("document", "")
        linked = entry.get("checks", [])
        if not document.is_file():
            errors.append(f"{topic}: нет документа {document.relative_to(ROOT)}")
        else:
            text = document.read_text(encoding="utf-8")
            if "**Перед чтением:**" not in text:
                errors.append(f"{topic}: не указаны предварительные знания")
            if not re.search(r"\*\*После\s+[^*]+:\*\*", text):
                errors.append(f"{topic}: не указан проверяемый результат изучения")
            if not SUMMARY.search(text):
                errors.append(f"{topic}: нет краткого итога")

            linked_practice = (set(PRACTICE_LINK.findall(text)) & practice_ids) - {"capstone"}
            if not linked_practice:
                errors.append(f"{topic}: нет ссылки на свой раздел практикума")
            else:
                for practice_id in linked_practice:
                    start = practice_text.index(f'<a id="{practice_id}"></a>')
                    end = practice_text.find('<a id="', start + 1)
                    section = practice_text[start : end if end != -1 else None]
                    for level in ("Основа", "Изменение", "Сценарий"):
                        if f"| {level} |" not in section:
                            errors.append(
                                f"{topic}: в разделе практикума нет уровня {level.lower()}"
                            )
        if not 2 <= len(linked) <= 5:
            errors.append(f"{topic}: нужно от 2 до 5 проверок, указано {len(linked)}")
        if len(linked) != len(set(linked)):
            errors.append(f"{topic}: одна проверка указана несколько раз")
        for check in linked:
            if check not in checks:
                errors.append(f"{topic}: проверка {check!r} не найдена")

    if errors:
        print("Ошибки покрытия примерами:", file=sys.stderr)
        print("\n".join(f"- {error}" for error in errors), file=sys.stderr)
        return 1

    used = {check for entry in entries for check in entry["checks"]}
    print(f"Проверено тем: {len(entries)}; связанных тестов и замеров: {len(used)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
