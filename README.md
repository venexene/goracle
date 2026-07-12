# goracle

Russian-language knowledge base on Go internals. Each topic goes from fundamentals to practical takeaways.

Built with [MkDocs Material](https://squidfunk.github.io/mkdocs-material/). Text editing assisted by Deepseek V4 Pro.

## Topics

**Computer Science** — algorithms and data structures, concurrency, computer networks and systems.

**Architecture & Design** — SOLID principles in Go, OOP without classes, clean and microservice architecture.

**Language Details** — types, arrays and slices, maps, strings, interfaces, channels — from syntax to internals.

**Go Internals** — scheduler (GMP model), memory management, escape analysis, garbage collector.

## Project structure

```
goracle/
├── mkdocs.yml              # MkDocs Material config
├── docs/                   # all content (markdown + assets)
├── Agents/                 # AI agent configs for content editing
│   ├── planner_writer.md   # planning agent (Deepseek V4 Pro)
│   └── implementer_writer.md # implementation agent (Deepseek V4 Pro)
├── .github/workflows/      # GitHub Pages deploy
└── site/                   # built output (gitignored)
```

## Run locally

```bash
pip install mkdocs-material pymdown-extensions
mkdocs serve
```

Open `http://127.0.0.1:8000`.

## Build

```bash
mkdocs build
```

Output in `site/`.

## Deploy

Push to `main` triggers GitHub Actions — builds and deploys to GitHub Pages. Settings: Pages → Source → GitHub Actions.