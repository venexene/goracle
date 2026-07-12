# goracle

Russian-language knowledge base on Go internals. Each topic goes from fundamentals to practical takeaways.

## Topics

**Go Fundamentals** — scheduler (GMP model), memory management, escape analysis, garbage collector.

**General** — algorithms and data structures, SOLID principles in Go, OOP without classes, clean and microservice architecture, concurrency, computer networks and systems.

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