# Логирование в Go

## 1. Введение

Логи — единственный способ узнать, что происходит внутри работающей программы. Без логов production-сервис — чёрный ящик: падает, но неизвестно почему и на каком запросе.

Go прошёл путь от простого `log.Println` до стандартного структурированного логирования в `log/slog` (Go 1.21). До `slog` экосистема была фрагментирована: zap для скорости, zerolog для zero-allocation, logrus для привычного API. Теперь `slog` — стандарт, встроенный в язык, с единым интерфейсом и поддержкой уровней, атрибутов и handler'ов.

В этой статье разобраны:

* зачем нужно логирование: уровни, структурированные vs неструктурированные логи
* стандартный пакет `log`: `Print`, `Fatal`, `Panic`, флаги, кастомные логеры
* `log/slog`: ключевые концепции, атрибуты, `Logger` и `Handler`
* handlers: `TextHandler` и `JSONHandler`, настройка через `HandlerOptions`
* уровни логирования: `Debug`, `Info`, `Warn`, `Error`
* контекстное логирование: `InfoContext`, проброс request ID и trace ID
* группировка и вложенные атрибуты: `Group`, `With`
* логирование ошибок: атрибуты, stack trace
* производительность: `LogAttrs`, сравнение с zap/zerolog/logrus
* интеграция с OpenTelemetry: связь логов и трейсов
* логирование в production: 12 Factor App, ротация, централизованный сбор
* тестирование логирования: проверка структуры и содержимого логов

> **Зачем это Go-разработчику.** Логирование — не фича, а инфраструктура. Без структурированных логов невозможно настроить алерты, искать ошибки и понимать, что произошло в 3 часа ночи. `log/slog` — стандарт с Go 1.21: никаких внешних зависимостей, единый API для всех проектов. Понимание `slog` от атрибутов до handler'ов — обязательный навык для Go-разработчика в 2025 году.

***

## 2. Зачем нужно логирование

`fmt.Println` работает для отладки на локальной машине. В production он бесполезен: нет временных меток, нет уровней, вывод перемешан из разных горутин, нельзя выключить debug-сообщения.

Логирование решает три задачи:

1. **Диагностика.** Что произошло перед падением? Какой запрос вызвал ошибку?
2. **Мониторинг.** Сколько ошибок в минуту? Растёт ли время ответа?
3. **Аудит.** Кто и когда менял данные? Откуда пришёл подозрительный запрос?

***

### Observability: логи, метрики, трейсы

Логирование — один из трёх столпов observability (наблюдаемости):

| Столп | Вопрос | Инструмент |
|---|---|---|
| **Логи** | Что произошло? | `slog`, Loki, ELK |
| **Метрики** | Сколько и как быстро? | Prometheus, Grafana |
| **Трейсы** | Где узкое место в цепочке вызовов? | Jaeger, Tempo, OpenTelemetry |

Логи — самый детальный источник. Метрики показывают агрегированную картину, трейсы — путь одного запроса через сервисы. Вместе они дают полную картину.

> **Зачем это Go-разработчику.** `fmt.Println` — не логирование. Без временной метки, уровня и контекста запроса сообщение в логе бесполезно. Observability — не DevOps-задача, а часть разработки: вы добавляете логи, метрики и трейсы в код, а не настраиваете их снаружи.

***

## 3. Стандартный пакет `log`

Пакет `log` — встроенный логер Go. Простой, без уровней, без структурирования. Подходит для маленьких утилит и быстрого прототипирования, но не для production-сервисов.

***

### `Print`, `Fatal`, `Panic`

```go
package main

import "log"

func main() {
    log.Print("сервер запускается")           // просто сообщение
    log.Println("сервер запущен на :8080")    // с переводом строки
    log.Printf("порт: %d", 8080)              // форматирование

    // Fatal = Print + os.Exit(1)
    log.Fatal("не могу подключиться к БД")    // программа завершается

    // Panic = Print + panic()
    log.Panic("необратимая ошибка")           // паника, которую можно recover
}
```

Ключевое различие:
- `Print` и `Printf` — просто выводят сообщение
- `Fatal` и `Fatalf` — выводят и вызывают `os.Exit(1)`. `defer`-ы при этом НЕ выполняются
- `Panic` и `Panicf` — выводят и вызывают `panic()`. `defer`-ы выполняются, панику можно поймать через `recover`

> В production-коде избегайте `log.Fatal` вне `main`: он убивает программу без возможности очистки. Возвращайте ошибку и дайте вызывающему коду решить, что делать.

***

### Настройка вывода и флагов

```go
log.SetOutput(os.Stderr) // по умолчанию stderr
log.SetFlags(log.LstdFlags | log.Lshortfile)
```

Флаги управляют префиксом сообщения:

| Флаг | Пример префикса |
|---|---|
| `Ldate` | `2025/07/21` |
| `Ltime` | `14:05:30` |
| `LstdFlags` | `2025/07/21 14:05:30` (Ldate + Ltime) |
| `Lshortfile` | `main.go:25:` |
| `Llongfile` | `/home/user/project/main.go:25:` |
| `LUTC` | UTC вместо локального времени |
| `Lmsgprefix` | Префикс перед сообщением, а не после временной метки |

### Кастомный логер

Вместо глобального логера — свой экземпляр с отдельным выводом:

```go
var (
    infoLogger  = log.New(os.Stdout, "INFO: ", log.LstdFlags)
    errorLogger = log.New(os.Stderr, "ERROR: ", log.LstdFlags|log.Lshortfile)
)

func main() {
    infoLogger.Println("сервер запущен")
    errorLogger.Println("не удалось открыть файл")
}
```

Кастомные логеры позволяют направлять разные уровни в разные писатели (stdout для Info, stderr для Error). Но уровней как таковых в `log` нет — INFO/ERROR/DEBUG только в префиксе, фильтровать их нельзя.

> **Зачем это Go-разработчику.** Стандартный `log` — минимальный инструмент. Подходит для утилит и прототипов. Для production-сервиса используйте `log/slog`: уровни, структурирование, контекст. `log.Fatal` убивает программу без `defer` — используйте только в `main`, когда дальнейшая работа невозможна.

***

## 4. `log/slog`: ключевые концепции

`log/slog` появился в Go 1.21 и решил главную проблему Go-логирования: фрагментацию. До него каждый проект выбирал между zap, zerolog, logrus — с несовместимыми API. Теперь `slog` — стандарт, встроенный в язык.

Три главные концепции:

1. **Logger** — фасад, через который вы пишете логи: `slog.Info("message")`
2. **Handler** — стратегия вывода: в текст, JSON, файл, внешний сервис. Logger делегирует Handler'у форматирование и запись
3. **Record** — структура, представляющая одно лог-сообщение: время, уровень, сообщение, атрибуты. Создаётся Logger'ом и передаётся Handler'у

```
Ваш код → slog.Logger → slog.Handler → io.Writer (os.Stdout, файл, сеть)
```

***

### Базовое использование

```go
import "log/slog"

func main() {
    slog.Info("сервер запущен", "port", 8080)
    slog.Debug("загружена конфигурация", "path", "/etc/app.yaml")
    slog.Warn("медленный запрос", "duration_ms", 1500)
    slog.Error("не удалось подключиться к БД", "error", err)
}
```

Вывод (TextHandler по умолчанию):

```
2025/07/21 14:05:30 INFO сервер запущен port=8080
2025/07/21 14:05:30 WARN медленный запрос duration_ms=1500
2025/07/21 14:05:30 ERROR не удалось подключиться к БД error="connection refused"
```

Каждое сообщение — пары ключ-значение. Ключи — строки, значения — любой тип. `slog.Info` и `slog.Error` работают на глобальном логере по умолчанию (который можно заменить через `slog.SetDefault`).

***

### Атрибуты: типизированные значения

`slog` принимает аргументы парами: ключ-строка, значение-any. Но для производительности есть типизированные атрибуты:

```go
slog.Info("пользователь создан",
    slog.String("name", "Alice"),
    slog.Int("id", 42),
    slog.Bool("active", true),
    slog.Float64("balance", 100.50),
    slog.Duration("elapsed", 250*time.Millisecond),
    slog.Time("created_at", time.Now()),
    slog.Any("metadata", map[string]any{"role": "admin"}),
)
```

Типизированные атрибуты (`slog.String`, `slog.Int`, ...) не аллоцируют память, в отличие от `"key", val`. Для горячих путей используйте `LogAttrs` (раздел 9).

### `slog.Logger` vs функции верхнего уровня

Функции `slog.Info`, `slog.Error` и т.д. пишут в глобальный логер. Это удобно для маленьких программ:

```go
slog.Info("сообщение") // глобальный логер
```

Для больших проектов — создавайте свой экземпляр и передавайте как зависимость:

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

logger.Info("сообщение") // именованный логер
```

Именованный логер можно обогатить постоянными атрибутами:

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).
    With("service", "user-api", "version", "1.2.3")

// Все сообщения этого логера будут содержать service и version
logger.Info("запрос обработан")
// {"time":"...","level":"INFO","msg":"запрос обработан","service":"user-api","version":"1.2.3"}
```

> **Зачем это Go-разработчику.** `slog` — единый стандарт. Не нужно выбирать между zap и zerolog на старте проекта. `slog.New` + `With` дают именованный логер с постоянными атрибутами — передавайте его через конструктор как зависимость. Функции верхнего уровня (`slog.Info`) — для утилит и примеров, не для production-кода.

***

## 5. Handlers: Text и JSON

Handler определяет, КАК лог записывается. `slog` поставляется с двумя: `TextHandler` (для людей) и `JSONHandler` (для машин). Вы можете написать свой handler для отправки в Loki, Elasticsearch или любой другой системы.

***

### `TextHandler`

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
logger.Info("запрос обработан",
    "method", "GET",
    "path", "/users",
    "status", 200,
    "duration_ms", 42,
)
```

Вывод:

```
time=2025-07-21T14:05:30.123+03:00 level=INFO msg="запрос обработан" method=GET path=/users status=200 duration_ms=42
```

Человеко-читаемый формат: `key=value`, подходит для разработки и небольших сервисов.

### `JSONHandler`

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger.Info("запрос обработан",
    "method", "GET",
    "path", "/users",
    "status", 200,
)
```

Вывод:

```json
{"time":"2025-07-21T14:05:30.123+03:00","level":"INFO","msg":"запрос обработан","method":"GET","path":"/users","status":200}
```

JSON — стандарт для production: легко парсить, индексировать и искать.

***

### `HandlerOptions`

```go
opts := &slog.HandlerOptions{
    Level:     slog.LevelDebug,       // минимальный уровень (по умолчанию Info)
    AddSource: true,                   // добавлять файл и строку вызова
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        if a.Key == "time" {
            a.Value = slog.StringValue(a.Value.Time().UTC().Format(time.RFC3339))
        }
        return a
    },
}

logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
```

| Опция | Описание |
|---|---|
| `Level` | Минимальный уровень. `Debug` — всё, `Info` — без Debug, `Warn` — Warn+Error |
| `AddSource` | Добавляет `source: "main.go:42"` в каждое сообщение |
| `ReplaceAttr` | Функция для модификации/удаления атрибутов перед записью |

`ReplaceAttr` — мощный механизм кастомизации:

```go
ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
    // Удалить все атрибуты с ключом "password"
    if a.Key == "password" {
        return slog.Attr{} // пустой атрибут удаляется
    }
    // Переименовать ключ
    if a.Key == "msg" {
        a.Key = "message"
    }
    return a
},
```

***

### Кастомные handlers

Handler — интерфейс:

```go
type Handler interface {
    Enabled(context.Context, Level) bool
    Handle(context.Context, Record) error
    WithAttrs(attrs []Attr) Handler
    WithGroup(name string) Handler
}
```

Кастомный handler для отправки логов в Loki/Elasticsearch должен реализовать эти 4 метода. На практике используют готовые адаптеры (`slogzap`, `slogzerolog`) или пишут обёртку над `JSONHandler`.

### Логер по умолчанию

```go
// Заменить глобальный логер
slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
})))
```

После `SetDefault` все вызовы `slog.Info`, `slog.Error` и т.д. используют новый логер.

> **Зачем это Go-разработчику.** `JSONHandler` в production, `TextHandler` в разработке. Переключайте через переменную окружения: `if os.Getenv("ENV") == "production" { JSONHandler } else { TextHandler }`. `AddSource: true` добавляет файл и строку — бесценно при отладке, но добавляет аллокации. `ReplaceAttr` — для фильтрации секретов и форматирования времени.

***

## 6. Уровни логирования

Уровень — фильтр важности. Правило: в development видно всё (Debug), в production — Info и выше.

`slog` использует 4 уровня:

| Уровень | Значение | Когда использовать |
|---|---|---|
| `Debug` | -4 | Детали, нужные только при отладке |
| `Info` | 0 | Штатные события: старт, остановка, обработка запроса |
| `Warn` | 4 | Нештатная ситуация, но сервис работает |
| `Error` | 8 | Ошибка, требующая внимания |

***

### Настройка уровня

```go
opts := &slog.HandlerOptions{
    Level: slog.LevelDebug, // показывать всё
}
logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
```

Программное изменение уровня:

```go
var levelVar slog.LevelVar
levelVar.Set(slog.LevelInfo)

opts := &slog.HandlerOptions{
    Level: &levelVar, // динамический уровень!
}
logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))

// Позже: включить Debug для диагностики
levelVar.Set(slog.LevelDebug)
```

`LevelVar` — атомарный переменный уровень. Можно менять на лету без пересоздания логера. Полезно для временного включения Debug в production через HTTP-эндпоинт:

```go
http.HandleFunc("/debug/loglevel", func(w http.ResponseWriter, r *http.Request) {
    levelVar.Set(slog.LevelDebug)
    w.Write([]byte("Debug enabled"))
})
```

### Соглашения по уровням

- **Debug** — значения переменных, содержимое запросов, SQL-запросы с параметрами. То, что нужно разработчику при отладке конкретной проблемы.
- **Info** — «пользователь создан», «платёж проведён», «сервер запущен». Штатный поток событий.
- **Warn** — «повторная попытка подключения к БД», «медленный запрос > 500ms». Не ошибка, но сигнал.
- **Error** — «не удалось создать пользователя», «БД недоступна». Требует реакции (алерт, тикет).

> **Зачем это Go-разработчику.** Уровень `Debug` в production — это гигабайты мусора, в котором тонут настоящие ошибки. `LevelVar` позволяет включить Debug точечно для диагностики без перезапуска. Правило: `Info` для штатных событий, `Error` для алертов, `Debug` для расследования инцидентов.

***

## 7. Контекстное логирование

Лог без контекста бесполезен: «ошибка подключения к БД» — но на каком запросе? От какого пользователя? Контекстное логирование привязывает лог-сообщение к конкретному запросу или операции через `context.Context`.

***

### `slog.InfoContext`, `slog.ErrorContext`

Методы с суффиксом `Context` принимают `context.Context`:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    slog.InfoContext(ctx, "запрос обработан", "method", r.Method, "path", r.URL.Path)
}
```

Сам по себе `InfoContext` не извлекает данные из контекста — для этого нужен middleware, который добавляет атрибуты в контекст.

### Паттерн: атрибуты через middleware + контекст

```go
type contextKey string
const requestIDKey contextKey = "requestID"

// Middleware: добавляет request ID в контекст
func requestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := uuid.New().String()
        w.Header().Set("X-Request-ID", requestID)

        ctx := context.WithValue(r.Context(), requestIDKey, requestID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Handler: извлекает request ID из контекста и логирует
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    requestID, _ := ctx.Value(requestIDKey).(string)

    slog.InfoContext(ctx, "обработка запроса",
        "request_id", requestID,
        "path", r.URL.Path,
    )
}
```

Результат:

```
time=... level=INFO msg="обработка запроса" request_id=a1b2c3d4 path=/users
```

Теперь по `request_id` можно найти все логи одного запроса среди тысяч других.

### Обогащение логера через `With`

Альтернатива ручному добавлению `request_id` в каждый вызов — создать дочерний логер с постоянными атрибутами:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    requestID := r.Context().Value(requestIDKey).(string)
    logger := slog.With("request_id", requestID)

    logger.InfoContext(r.Context(), "обработка запроса", "path", r.URL.Path)
    // ... дальше везде использовать logger вместо slog
}
```

`slog.With` возвращает новый `*slog.Logger` с добавленными атрибутами. Все сообщения через этот логер будут содержать `request_id`.

### Trace ID и OpenTelemetry

В системах с трейсингом request ID — это Span ID из OpenTelemetry:

```go
import "go.opentelemetry.io/otel/trace"

func handler(w http.ResponseWriter, r *http.Request) {
    span := trace.SpanFromContext(r.Context())
    traceID := span.SpanContext().TraceID().String()

    slog.InfoContext(r.Context(), "запрос обработан",
        "trace_id", traceID,
        "path", r.URL.Path,
    )
}
```

Trace ID связывает логи с трейсами: нашли ошибку в трейсе → нашли все логи этого запроса по `trace_id`.

> **Зачем это Go-разработчику.** Лог без request ID — иголка в стоге сена. Один проблемный запрос на тысячу нормальных — без контекста вы не отличите их. `slog.InfoContext` + `context.WithValue` + middleware = все логи запроса связаны. Trace ID замыкает треугольник observability: логи → метрики → трейсы.

***

## 8. Группировка и вложенные атрибуты

Когда атрибутов много, логи становятся плоской «лапшой»:

```
time=... level=INFO msg="запрос" method=GET path=/users user_id=42 user_name=Alice user_email=alice@example.com status=200 duration_ms=12
```

Группировка собирает связанные атрибуты в логический блок — проще читать и парсить.

***

### `slog.Group` — группировка в сообщении

```go
slog.Info("запрос обработан",
    slog.Group("request",
        "method", "GET",
        "path", "/users",
    ),
    slog.Group("response",
        "status", 200,
        "duration_ms", 42,
    ),
    slog.Group("user",
        "id", 42,
        "name", "Alice",
    ),
)
```

JSON-вывод:

```json
{
    "time": "...",
    "level": "INFO",
    "msg": "запрос обработан",
    "request": {
        "method": "GET",
        "path": "/users"
    },
    "response": {
        "status": 200,
        "duration_ms": 42
    },
    "user": {
        "id": 42,
        "name": "Alice"
    }
}
```

Группы в JSON становятся вложенными объектами. В текстовом формате — префиксы:

```
time=... level=INFO msg="запрос обработан" request.method=GET request.path=/users response.status=200 response.duration_ms=42 user.id=42 user.name=Alice
```

### `With` — постоянные атрибуты на логере

`With` добавляет атрибуты, которые будут в КАЖДОМ сообщении этого логера:

```go
serviceLogger := slog.Default().With(
    "service", "user-api",
    "version", "1.2.3",
)

// service и version появятся во всех сообщениях
serviceLogger.Info("запуск")
serviceLogger.Info("запрос обработан", "path", "/users")
```

В отличие от `Group`, `With` не создаёт вложенную структуру — атрибуты добавляются на верхнем уровне:

```json
{"time":"...","level":"INFO","msg":"запуск","service":"user-api","version":"1.2.3"}
```

### `WithGroup` — постоянная группа на логере

```go
dbLogger := slog.Default().WithGroup("database")
dbLogger.Info("подключение установлено", "host", "localhost", "port", 5432)
```

```json
{
    "database": {
        "msg": "подключение установлено",
        "host": "localhost",
        "port": 5432
    }
}
```

Полезно для выделения подсистем: все логи БД — в группе `database`, все HTTP — в группе `http`.

### Когда группировать, а когда нет

**Группируйте**, когда:
- Атрибутов > 5 и они логически связаны (request, response, user)
- Данные вложенных структур: `user: { id, name, email }`
- Выделение подсистем: `db`, `cache`, `queue`

**Не группируйте**, когда:
- Атрибутов 1–2 — группа из одного поля бессмысленна
- Атрибут нужен для поиска на верхнем уровне (`request_id`, `trace_id`)
- Чрезмерная вложенность (больше 2 уровней) усложняет поиск

> **Зачем это Go-разработчику.** Группировка превращает плоскую лапшу в читаемый JSON. `slog.Group` для разовых группировок в сообщении, `With` для постоянных атрибутов на логере, `WithGroup` для выделения подсистем. Не перегружайте: лог должен оставаться удобным для поиска, а глубокая вложенность мешает grep и jq.

***

## 9. Логирование ошибок

Ошибка — самое важное, что попадает в лог. Как её логировать — влияет на скорость диагностики.

***

### `slog.Error` — сообщение + ошибка

```go
user, err := createUser(req)
if err != nil {
    slog.Error("не удалось создать пользователя", "error", err)
    return
}
```

Вывод с `JSONHandler`:

```json
{"time":"...","level":"ERROR","msg":"не удалось создать пользователя","error":"duplicate email: alice@example.com"}
```

> Не логируйте ошибку на каждом уровне. Если функция возвращает ошибку наверх — логирует самый верхний обработчик (HTTP handler, main). Иначе одна ошибка породит 5 одинаковых логов.

### Атрибуты ошибки

```go
slog.Error("запрос к БД не удался",
    "error", err,
    "query", query,
    "duration_ms", elapsed.Milliseconds(),
    "retry", retryCount,
)
```

Контекст вокруг ошибки важнее самой ошибки. «connection refused» без параметров подключения — гадание. «connection refused» с `host`, `port`, `user`, `duration_ms` — диагноз за секунду.

### Stack trace

`slog` не добавляет stack trace автоматически. Для отладочных целей:

```go
import "runtime/debug"

slog.Error("паника в обработчике",
    "error", err,
    "stack", string(debug.Stack()),
)
```

`debug.Stack()` возвращает stack trace текущей горутины. В production логируйте stack trace только для паник и критических ошибок — это много текста.

### Соглашения

- **Сообщение** — что произошло, человеческим языком: «не удалось создать пользователя»
- **error** — всегда включайте оригинальную ошибку: `"error", err`
- **Контекст** — параметры операции, которые помогут воспроизвести ошибку
- **Не логируйте и не возвращайте.** Если функция логирует ошибку и возвращает её же — вызывающий код тоже залогирует. Двойное логирование путает.

> Правило: либо логируйте, либо возвращайте. Не делайте оба.

> **Зачем это Go-разработчику.** Хороший лог ошибки отвечает на три вопроса: что случилось, на каких данных, в каком контексте. `slog.Error("не удалось", "error", err)` — недостаточно. Добавьте `"user_id"`, `"query"`, `"duration_ms"`. `debug.Stack()` — крайнее средство, не сыпьте stack trace на каждую ошибку валидации.

***

## 10. Производительность

Логирование не должно тормозить приложение. В высоконагруженных системах (100K+ запросов/с) цена логирования становится заметной.

***

### Цена аллокаций

`slog.Info("msg", "key", val)` — `val` упаковывается в `any` (интерфейс), что вызывает аллокацию на куче. Для горячих путей — `LogAttrs`:

```go
// Аллокации: каждый "key", val — упаковка в any
slog.Info("запрос обработан", "method", "GET", "path", "/users", "status", 200)

// Без аллокаций: типизированные Attr
slog.LogAttrs(context.Background(), slog.LevelInfo, "запрос обработан",
    slog.String("method", "GET"),
    slog.String("path", "/users"),
    slog.Int("status", 200),
)
```

`LogAttrs` принимает `...slog.Attr` вместо `...any` — никакой упаковки, никаких аллокаций.

### Сравнение библиотек

Ориентировочные бенчмарки (меньше — лучше):

| Библиотека | ns/op | B/op | allocs/op |
|---|---|---|---|
| `slog` (LogAttrs) | ~100 | 0 | 0 |
| `zerolog` | ~80 | 0 | 0 |
| `zap` (sugared) | ~400 | ~64 | ~3 |
| `slog` (key-value) | ~500 | ~120 | ~5 |
| `logrus` | ~2000 | ~200 | ~10 |

- `zerolog` и `slog.LogAttrs` — zero-allocation
- `zap` (без sugared) — тоже zero-allocation, но многословнее
- `logrus` — медленный, использует reflection
- `slog` с key-value парами — аллокации на упаковку аргументов

### Асинхронное логирование

Для максимальной пропускной способности — писать логи в буфер и сбрасывать пакетами:

```go
type AsyncHandler struct {
    inner slog.Handler
    ch    chan func()
}

func NewAsyncHandler(inner slog.Handler, bufferSize int) *AsyncHandler {
    h := &AsyncHandler{
        inner: inner,
        ch:    make(chan func(), bufferSize),
    }
    go h.run()
    return h
}

func (h *AsyncHandler) Handle(ctx context.Context, r slog.Record) error {
    // Копируем record и отправляем в канал
    r2 := r.Clone()
    done := make(chan error, 1)
    h.ch <- func() {
        done <- h.inner.Handle(ctx, r2)
    }
    return <-done
}

func (h *AsyncHandler) run() {
    for fn := range h.ch {
        fn()
    }
}
```

На практике используйте готовые решения: `slog-sampling`, `slog-dedup`, `slog-batch`.

### Когда производительность критична

- **> 50 000 запросов/с** — используйте `LogAttrs` и асинхронный handler
- **10 000–50 000** — `LogAttrs` достаточно
- **< 10 000** — `slog.Info("msg", "key", val)` ок, аллокации незаметны

> **Зачем это Go-разработчику.** Не оптимизируйте логирование преждевременно. Начните с `slog.Info("msg", "key", val)`. Если бенчмарк показывает, что логирование — узкое место, переходите на `LogAttrs`. Async-логер — последний шаг для систем с 100K+ RPS. Самое медленное логирование — то, что пишется на диск без буферизации.

***

## 11. Интеграция с OpenTelemetry

OpenTelemetry — стандарт сбора телеметрии: трейсов, метрик и логов. `slog` интегрируется с OTel через log bridge: лог-сообщения автоматически получают Trace ID и Span ID и могут экспортироваться в OTel collector.

***

### Бридж `slog` ↔ OpenTelemetry

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/sdk/resource"
    otelslog "go.opentelemetry.io/contrib/bridges/otelslog"
)

func main() {
    // Создать OTel логер
    provider := // ... настройка OTel provider
    otelLogger := otelslog.NewLogger("my-service",
        otelslog.WithResource(resource.NewWithAttributes(/* ... */)),
    )

    // Установить как глобальный slog
    slog.SetDefault(otelLogger)

    // Все логи теперь содержат Trace ID и Span ID
    slog.InfoContext(ctx, "запрос обработан")
}
```

После настройки каждое лог-сообщение автоматически содержит:

```json
{
    "time": "...",
    "level": "INFO",
    "msg": "запрос обработан",
    "trace_id": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4",
    "span_id": "1a2b3c4d5e6f7a8b"
}
```

### Связь логов и трейсов

Без OTel trace ID и span ID нужно добавлять вручную. С OTel — автоматически:

- Нашли медленный трейс в Jaeger → скопировали `trace_id` → нашли все логи в Loki/Elasticsearch
- Нашли ошибку в логах → скопировали `trace_id` → открыли трейс → увидели всю цепочку вызовов

### Экспорт в OTel Collector

```go
import (
    "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
    sdklog "go.opentelemetry.io/otel/sdk/log"
)

func setupLogging() *sdklog.LoggerProvider {
    exporter, _ := otlploghttp.New(ctx, otlploghttp.WithEndpoint("localhost:4318"))
    provider := sdklog.NewLoggerProvider(
        sdklog.WithProcessor(
            sdklog.NewBatchProcessor(exporter),
        ),
    )
    otel.SetLoggerProvider(provider)
    return provider
}
```

Логи отправляются в OTel Collector → оттуда в Loki, Elasticsearch или любую другую систему.

> **Зачем это Go-разработчику.** OpenTelemetry — важный инструмент. Trace ID в логах превращает разрозненные сообщения в связный рассказ о запросе. OTel bridge для `slog` — 10 строк кода, которые экономят часы расследования инцидентов. Настройте OTel один раз и получайте сквозную наблюдаемость: логи + метрики + трейсы.

***

## 12. Логирование в production

Логирование в production отличается от разработки: это не отладка, а наблюдаемость и аудит.

***

### 12 Factor App

[12 Factor App](https://12factor.net/logs) — логи как поток событий. Приложение пишет в `stdout`, среда выполнения собирает и маршрутизирует:

```
Приложение → stdout → Docker/Systemd → Fluentd/Vector → Loki/ELK → Grafana
```

Приложение не знает, куда уходят логи — оно просто пишет в `stdout`. Это разделение ответственности: разработчик добавляет логи, DevOps настраивает сбор и хранение.

```go
// Правильно: всё в stdout
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

// Неправильно: приложение пишет в файл
file, _ := os.OpenFile("/var/log/app.log", ...)
logger := slog.New(slog.NewJSONHandler(file, nil))
```

***

### Ротация логов

Если приложение всё же пишет в файл — нужна ротация. Библиотека `lumberjack`:

```go
import "gopkg.in/natefinch/lumberjack.v2"

logger := slog.New(slog.NewJSONHandler(&lumberjack.Logger{
    Filename:   "/var/log/app.log",
    MaxSize:    100, // мегабайт
    MaxBackups: 3,
    MaxAge:     7,   // дней
    Compress:   true,
}, nil))
```

`lumberjack` автоматически ротирует файлы: при достижении MaxSize создаёт новый, старые архивирует, лишние удаляет.

### Централизованный сбор

Один сервер — можно читать логи из файла. Десять серверов — нужен центральный сбор:

| Система | Тип |
|---|---|
| **Loki** + **Grafana** | Легковесный, горизонтально масштабируемый, интегрирован с Grafana |
| **ELK** (Elasticsearch + Logstash + Kibana) | Тяжелее, мощный полнотекстовый поиск |
| **Datadog / New Relic** | SaaS, платно, минимум настройки |

### Что НЕ логировать

- **Пароли и токены.** Даже в замаскированном виде — не рискуйте.
- **Персональные данные.** Email, телефон, паспорт — подпадают под GDPR/152-ФЗ.
- **Тела запросов/ответов.** Могут содержать чувствительные данные. Если нужно — маскируйте.
- **Ключи шифрования.** Даже в Debug.

Используйте `ReplaceAttr` для фильтрации:

```go
ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
    sensitive := map[string]bool{
        "password": true, "token": true, "secret": true,
    }
    if sensitive[a.Key] {
        return slog.String(a.Key, "***REDACTED***")
    }
    return a
}
```

### Уровни в разных окружениях

| Окружение | Уровень | Почему |
|---|---|---|
| Development | Debug | Видно всё для отладки |
| Staging | Info | Как production, но можно включить Debug |
| Production | Info | Debug забивает диски и маскирует ошибки |

> **Зачем это Go-разработчику.** Логи в production — не `fmt.Println`, а структурированный JSON в stdout. Никаких файлов в коде приложения — только stdout, сбор и ротация на стороне инфраструктуры. `ReplaceAttr` для маскирования секретов — не опция, а требование безопасности. Уровень Debug в production — частая причина отказа системы логирования под нагрузкой.

***

## 13. Тестирование логирования

Логи — часть поведения программы, их можно и нужно тестировать. Проверяйте, что нужное сообщение появилось с правильным уровнем и атрибутами.

***

### Тест через буфер

```go
func TestHandler(t *testing.T) {
    var buf bytes.Buffer
    logger := slog.New(slog.NewJSONHandler(&buf, nil))

    handler := NewUserHandler(logger)
    req := httptest.NewRequest("POST", "/users", strings.NewReader(`{"name":"Alice"}`))
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)

    // Парсим JSON-логи
    var logEntry map[string]any
    json.Unmarshal(buf.Bytes(), &logEntry)

    if logEntry["level"] != "INFO" {
        t.Errorf("expected INFO, got %v", logEntry["level"])
    }
    if logEntry["msg"] != "пользователь создан" {
        t.Errorf("expected 'пользователь создан', got %v", logEntry["msg"])
    }
}
```

Паттерн: `bytes.Buffer` + `JSONHandler` → парсим JSON → проверяем поля. Проще и надёжнее, чем сравнивать строки.

### Что тестировать в логах

1. **Наличие сообщения.** Ошибка залогирована? Старт сервиса залогирован?
2. **Уровень.** Ошибка — Error? Штатное событие — Info?
3. **Атрибуты.** `request_id` присутствует? `user_id` совпадает с созданным?
4. **Контекст.** При ошибке есть параметры, помогающие диагностике?

### Интеграция с тестовым логером

```go
func TestService(t *testing.T) {
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))

    svc := NewService(logger)

    // Логи идут в вывод теста: go test -v покажет их
    err := svc.DoSomething()
    if err != nil {
        t.Fatal(err)
    }
}
```

При `go test -v` логи видны в выводе — удобно для отладки упавшего теста.

### Чего НЕ тестировать

- **Точный текст сообщения.** «не удалось создать пользователя: duplicate email» vs «duplicate email: не удалось создать пользователя». Проверяйте наличие ключевых слов, а не точную строку.
- **Временные метки.** Время всегда разное.
- **Формат вывода.** JSON или Text — зона ответственности handler'а, не бизнес-логики.

> **Зачем это Go-разработчику.** Без тестов вы узнаете об отсутствии лога в 3 часа ночи при падении production. `bytes.Buffer` + `JSONHandler` — 5 строк для проверки. Проверяйте наличие, уровень и атрибуты — не точный текст.

***

## Источники

### Официальная документация

* [pkg.go.dev/log/slog](https://pkg.go.dev/log/slog) — полная документация пакета `log/slog`
* [Go Blog: Structured Logging with slog](https://go.dev/blog/slog) — официальный блог-пост от команды Go
* [slog: Proposal](https://go.googlesource.com/proposal/+/master/design/56345-structured-logging.md) — дизайн-документ `slog`
* [pkg.go.dev/log](https://pkg.go.dev/log) — документация стандартного пакета `log`

### Туториалы и статьи

* [A Comprehensive Guide to Structured Logging in Go](https://betterstack.com/community/guides/logging/logging-in-go/) — Better Stack: от `log` до `slog`
* [Using slog for Structured Logging in Go](https://www.alexedwards.net/blog/using-slog) — Alex Edwards
* [Go slog: The Ultimate Guide](https://blog.jetbrains.com/go/2023/10/10/go-slog-ultimate-guide/) — JetBrains

### Библиотеки

* [uber-go/zap](https://github.com/uber-go/zap) — быстрый структурированный логер (Uber)
* [rs/zerolog](https://github.com/rs/zerolog) — zero-allocation JSON-логер
* [sirupsen/logrus](https://github.com/sirupsen/logrus) — популярный структурированный логер (до `slog`)
* [lumberjack](https://github.com/natefinch/lumberjack) — ротация лог-файлов

### OpenTelemetry

* [OpenTelemetry Go: Logs](https://opentelemetry.io/docs/languages/go/) — интеграция с OpenTelemetry
* [otelslog](https://pkg.go.dev/go.opentelemetry.io/contrib/bridges/otelslog) — бридж OTel ↔ `slog`

### Лучшие практики

* [12 Factor App: Logs](https://12factor.net/logs) — логи как поток событий
* [Google SRE Book: Monitoring Distributed Systems](https://sre.google/sre-book/monitoring-distributed-systems/) — логирование и мониторинг
