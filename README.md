# Goracle

[![GitHub Pages](https://img.shields.io/github/actions/workflow/status/venexene/goracle/deploy.yml?label=deploy&logo=github)](https://venexene.github.io/goracle)
[![MkDocs Material](https://img.shields.io/badge/mkdocs-material-526CFE?logo=materialformkdocs&logoColor=white)](https://squidfunk.github.io/mkdocs-material/)
[![Go](https://img.shields.io/badge/go-reference-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Deepseek V4 Pro](https://img.shields.io/badge/ai-deepseek--v4-blueviolet)](https://copilot.github.com)
[![License: MIT](https://img.shields.io/badge/license-MIT-green)](LICENSE)

Русскоязычная база знаний по Go - от устройства рантайма до архитектурных паттернов. **29 тем** в 4 разделах, с диаграммами, сниппетами и поиском. Deepseek V4 Pro применялся для редактирования текстов.

🌐 **[venexene.github.io/goracle](https://venexene.github.io/goracle)**

---

## Tech Stack

**MkDocs Material** · **GitHub Pages** · **GitHub Actions** · **KaTeX** · **Go** · **Deepseek V4 Pro**

---

## Темы

| Раздел | Содержание |
|--------|-----------|
| **Computer Science** | Алгоритмы, конкурентность и паттерны, компьютерные сети и системы - теоретический фундамент |
| **Архитектура и дизайн** | SOLID в Go, ООП без классов, чистая и микросервисная архитектура, gRPC |
| **Детали языка** | Типы, массивы и срезы, мапы, строки, интерфейсы, каналы, defer, замыкания, ошибки, контекст, синхронизация, дженерики - от синтаксиса до рантайма |
| **Устройство Go** | Планировщик (GMP), управление памятью, escape analysis, сборщик мусора, инлайн, выравнивание |

---

## Структура

```
goracle/
├── mkdocs.yml                  # конфигурация MkDocs Material
├── docs/                       # всё содержимое (markdown + изображения)
│   ├── index.md                # главная страница с картой тем
│   ├── computer-science/       # Computer Science (5 тем)
│   ├── architecture/           # Архитектура и дизайн (5 тем)
│   ├── go-details/             # Детали языка (13 тем)
│   ├── go-fundamentals/        # Устройство Go (6 тем)
│   └── javascripts/            # KaTeX-рендеринг для формул
├── Agents/                     # инструкции для AI-агентов
│   ├── planner_writer.md
│   └── implementer_writer.md
├── .github/workflows/          # CI/CD — деплой на GitHub Pages
└── IMPROVEMENTS.md             # план развития (16 тем в очереди)
```

---

## Локальный запуск

```bash
pip install mkdocs-material pymdown-extensions
mkdocs serve
```

Открыть `http://127.0.0.1:8000`.

## Сборка

```bash
mkdocs build
```

Результат в `site/`.