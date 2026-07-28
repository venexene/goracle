# Goracle

[![GitHub Pages](https://img.shields.io/github/actions/workflow/status/venexene/goracle/deploy.yml?label=deploy&logo=github)](https://venexene.github.io/goracle)
[![MkDocs Material](https://img.shields.io/badge/mkdocs-material-526CFE?logo=materialformkdocs&logoColor=white)](https://squidfunk.github.io/mkdocs-material/)
[![Go](https://img.shields.io/badge/go-reference-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Deepseek V4 Pro](https://img.shields.io/badge/ai-deepseek--v4-blueviolet)](https://copilot.github.com)
[![License: MIT](https://img.shields.io/badge/license-MIT-green)](LICENSE)

A Russian-language knowledge base on Go - from runtime internals to architectural patterns. **30+ topics** across 5 sections, with diagrams, code snippets, and search. Deepseek V4 Pro was used for text editing.

**[venexene.github.io/goracle](https://venexene.github.io/goracle)**

---

## Tech Stack

**MkDocs Material** · **GitHub Pages** · **GitHub Actions** · **KaTeX** · **Go** · **Deepseek V4 Pro**

---

## Topics

| Section | Contents |
|---------|----------|
| **Computer Science** | Algorithms, concurrency patterns, computer networks and systems - the theoretical foundation |
| **Architecture & Design** | SOLID in Go, OOP without classes, clean and microservice architecture, HTTP, gRPC |
| **Language Details** | Types, arrays and slices, maps, strings, interfaces, channels, defer, closures, errors, context, synchronization, generics - the language itself |
| **Go Internals** | Scheduler (GMP model), memory management, escape analysis, garbage collector, inlining, alignment |
| **Tools & Practice** | Testing, databases, logging - real-world tools and practices for production Go |

### Computer Science

| Topic | Description |
|-------|-------------|
| [Algorithms & Data Structures](<docs/computer-science/Алгоритмы и структуры данных/algorithms-and-data-structures.md>) | Big O, arrays, linked lists, trees, hash tables, greedy algorithms, DP |
| [Parallelism & Concurrency](<docs/computer-science/Параллельность и конкрентность/concurrency.md>) | Goroutines, channels, synchronization primitives, data races, memory model |
| [Concurrency Patterns](<docs/computer-science/Конкурентные паттерны в Go/concurrency-patterns.md>) | Worker Pool, Fan-in, Fan-out, Pipeline, Cancellation |
| [Computer Networks](<docs/computer-science/Компьютерные сети/computer-networks.md>) | OSI, TCP/IP, HTTP, gRPC - networking fundamentals |
| [Computer Systems](<docs/computer-science/Компьютерные системы/computer-systems.md>) | CPU, memory, operating systems, interrupts |

### Architecture & Design

| Topic | Description |
|-------|-------------|
| [Programming Principles](<docs/architecture/Принципы программирования/programming-principles.md>) | SOLID in Go, KISS, DRY, YAGNI |
| [OOP in Go](docs/architecture/ООП/oop.md) | Structs, interfaces, embedding, polymorphism without inheritance |
| [Clean Architecture](<docs/architecture/Чистая архитектура/clean-architecture.md>) | Layers, dependencies, repositories, dependency injection in Go |
| [Microservice Architecture](<docs/architecture/Микросервисная архитектура/microservices.md>) | Decomposition, communication, database per service, Conway's Law |
| [HTTP](docs/architecture/HTTP/http.md) | HTTP/1.1–HTTP/3 and `net/http`: server, client, routing, concurrency |
| [gRPC](docs/architecture/gRPC/index.md) | Protobuf, HTTP/2, servers, clients, streaming, interceptors |

### Language Details

| Topic | Description |
|-------|-------------|
| [Data Types](<docs/go-details/Типы данных в Go/types.md>) | Static typing, basic and composite types, zero values |
| [Arrays & Slices](<docs/go-details/Массивы и срезы в Go/arrays-and-slices.md>) | Stack arrays, slice-header, append, escape analysis, memory leaks |
| [Maps](<docs/go-details/Мапы в Go/maps.md>) | hmap and buckets, evacuation, Swiss Table (Go 1.24+), thread safety |
| [Strings](<docs/go-details/Строки в Go/strings.md>) | String structure, UTF-8, immutability, rune iteration |
| [Interfaces](<docs/go-details/Интерфейсы в Go/interfaces.md>) | Implicit implementation, type assertion/switch, iface/eface, nil traps |
| [Channels](<docs/go-details/Каналы в Go/channels.md>) | Buffered and unbuffered, select, hchan internals, blocking/unblocking |
| [Defer](<docs/go-details/Defer в Go/defer.md>) | Deferred execution, LIFO, arguments vs closures, panic/recover |
| [Closures](<docs/go-details/Замыкания в Go/closures.md>) | Variable capture, function factories, state in closures |
| [Errors](<docs/go-details/Ошибки в Go/errors.md>) | The `error` interface, wrapping, `errors.Is`/`As`, custom errors |
| [Context](<docs/go-details/Контекст в Go/context.md>) | Background, WithCancel/Timeout, Value, internals |
| [Synchronization](<docs/go-details/Синхронизация в Go/synchronization.md>) | Mutex, RWMutex, WaitGroup, atomic, Cond, Pool, Map |
| [Generics](<docs/go-details/Дженерики в Go/generics.md>) | Type parameters, constraints, type inference |

### Tools & Practice

| Topic | Description |
|-------|-------------|
| [Testing](<docs/tools-and-practice/Тестирование в Go/testing.md>) | `go test`, table-driven tests, benchmarks, fuzzing, mocking, `httptest` |
| [Databases in Go](<docs/tools-and-practice/Базы данных в Go/db.md>) | SQL basics, ACID, `database/sql`, `sqlx`, `pgx`, migrations, ORM, `sqlc` |
| [Logging](<docs/tools-and-practice/Логирование в Go/logging.md>) | `log/slog`, levels, attributes, handlers, contextual logging, OpenTelemetry |

### Go Internals

| Topic | Description |
|-------|-------------|
| [Go Scheduler](<docs/go-fundamentals/Планировщик Go/scheduler.md>) | OS scheduling to GMP model: goroutines, machines, processors, work-stealing |
| [Memory in Go](<docs/go-fundamentals/Память в Go/memory.md>) | Virtual memory, paging, stack and heap, Go allocator |
| [Escape Analysis](<docs/go-fundamentals/Escape Analysis в Go/escape-analysis.md>) | How the compiler decides stack vs heap allocation |
| [Garbage Collector](<docs/go-fundamentals/Сбощик мусора в Go/garbage-collector.md>) | Tricolor marking, write barriers, GC phases, GOGC, tracing |
| [Inlining](<docs/go-fundamentals/Инлайн в Go/inlining.md>) | Function inlining by the compiler, conditions, impact on stack traces |
| [Alignment](<docs/go-fundamentals/Выравнивание в Go/alignment.md>) | Struct field alignment, padding, field order optimization |

---

## Structure

```
goracle/
├── mkdocs.yml                  # MkDocs Material configuration
├── docs/                       # all content
│   ├── index.md                # home page with topic map
│   ├── computer-science/       # Computer Science
│   ├── architecture/           # Architecture & Design
│   ├── go-details/             # Language Details (12 topics)
│   ├── go-fundamentals/        # Go Internals (6 topics)
│   ├── tools-and-practice/     # Tools & Practice (3 topics)
│   └── javascripts/            # KaTeX rendering for math formulas
├── Agents/                     # AI agent instructions
│   ├── planner_writer.md
│   └── implementer_writer.md
└── .github/workflows/          # CI/CD - deploy to GitHub Pages
```

---

## Local Development

```bash
pip install mkdocs-material pymdown-extensions
mkdocs serve
```

Open `http://127.0.0.1:8000`.

## Build

```bash
mkdocs build
```

Output goes to `site/`.