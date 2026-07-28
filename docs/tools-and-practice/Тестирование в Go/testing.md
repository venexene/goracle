# Тестирование в Go

## 1. Введение

Go поставляется с собственной системой тестирования: пакет `testing` и инструмент `go test`. Никаких внешних библиотек для запуска тестов не нужно — всё в стандартной поставке. Тесты живут в файлах с суффиксом `_test.go` рядом с тестируемым кодом, а функции тестов начинаются с `Test`.

Философия тестирования в Go — минимализм. Нет встроенных assert-функций: условие проверяется обычным `if` с `t.Error` или `t.Fatal`. Нет хуков для настройки/очистки — вместо них `TestMain` и `t.Cleanup`. Это намеренное решение: тесты на Go выглядят как обычный код, а не как DSL фреймворка.

В этой статье разобраны:

* базовые тесты: `TestXxx`, `t.Error`, `t.Fatal`, `go test`
* табличные тесты — основной паттерн организации тестов в Go
* `t.Helper()` и вспомогательные функции для тестов
* параллельные тесты через `t.Parallel()`
* `TestMain` и `t.Cleanup` для настройки и очистки
* бенчмарки: `BenchmarkXxx`, `b.N`, `-benchmem`
* фаззинг: `FuzzXxx`, начальный корпус, словари (Go 1.18+)
* примеры-как-тесты: `ExampleXxx` и `// Output:`
* покрытие кода: `go test -cover`, `-coverprofile`, `go tool cover`
* мокирование через интерфейсы и генерацию моков
* `httptest` для тестирования HTTP-обработчиков
* интеграционные тесты с Docker через `testcontainers-go`
* паттерны и лучшие практики: Arrange/Act/Assert, эталонные файлы

> **Зачем это Go-разработчику.** Тестирование в Go — не дополнительный навык, а базовая часть языка. `go test` встроен в цепочку инструментов наравне с `go build`. Стандартный пакет `testing` покрывает модульные тесты, бенчмарки, фаззинг и примеры — без внешних зависимостей. Понимание этой системы позволяет писать тесты, которые не тормозят разработку и не ломаются при рефакторинге.

***

## 2. Базовый тест

Тест в Go — функция с сигнатурой `func TestXxx(t *testing.T)` в файле с суффиксом `_test.go`. `Xxx` — любое имя, начинающееся с заглавной буквы (экспортируемое). Пакет теста может быть тем же (`package math`) или с суффиксом `_test` (`package math_test`). Суффикс `_test` даёт доступ только к экспортируемым идентификаторам — тестирование как внешний потребитель.

Запуск: `go test` (текущий пакет), `go test ./...` (все подпакеты), `go test -v` (подробный вывод).

***

### Первый тест

```go
package math_test

import (
    "math"
    "testing"
)

func TestAbs(t *testing.T) {
    got := math.Abs(-1)
    if got != 1 {
        t.Errorf("Abs(-1) = %f; want 1", got)
    }
}
```

`t.Errorf` сообщает об ошибке, но **не останавливает** тест — остальной код функции выполняется. `t.Fatalf` сообщает и немедленно завершает тест (вызывает `runtime.Goexit`). Выбирайте `Error`, если можно продолжать проверки; `Fatal`, если дальнейшие проверки бессмысленны.

```go
func TestDiv(t *testing.T) {
    result, err := Div(10, 0)
    if err == nil {
        t.Fatal("Div(10, 0) should return an error")  // останавливаемся
    }
    // result невалиден, проверять его бессмысленно
}
```

***

### Соглашения об именах

- Файл: `foo_test.go` — тесты для `foo.go`
- Функция: `Test` + имя тестируемой функции/типа: `TestAbs`, `TestUserService_Create`
- Подтесты через `t.Run`: имя описывает сценарий: `"negative number"`, `"zero value"`

```go
func TestAbs(t *testing.T) {
    t.Run("negative number", func(t *testing.T) {
        if got := math.Abs(-5); got != 5 {
            t.Errorf("Abs(-5) = %f; want 5", got)
        }
    })
    t.Run("positive number", func(t *testing.T) {
        if got := math.Abs(5); got != 5 {
            t.Errorf("Abs(5) = %f; want 5", got)
        }
    })
}
```

Подтесты можно запускать выборочно: `go test -run TestAbs/negative`.

***

### Пакет теста: внутренний vs внешний

```go
// math_test.go — внутренний пакет: доступ к неэкспортируемым идентификаторам
package math

// math_test.go — внешний пакет: только экспортируемый API
package math_test
```

Внешний пакет (`_test`) проверяет поведение как внешний потребитель. Внутренний — для тестирования деталей реализации. Правило: если тест проверяет публичный API, используйте `_test`. Если нужен доступ к приватным полям — без суффикса.

> **Зачем это Go-разработчику.** Базовый тест — это функция и `if`. Никакой магии, никакого DSL. `t.Errorf` vs `t.Fatalf` — ключевое различие: Error для продолжения проверок, Fatal для немедленной остановки. Имена тестов — часть документации: `TestAbs_negative` говорит больше, чем `TestAbs1`.

***

## 3. Табличные тесты (table-driven)

Табличные тесты — основной паттерн организации тестов в Go. Идея: определить слайс тест-кейсов (каждый — имя, входные данные и ожидаемый результат) и прогнать их в цикле. Go Wiki официально рекомендует этот подход.

Преимущества: легко добавить новый кейс (одна строка в слайсе), наглядно видно покрытие граничных случаев, при падении теста видно имя кейса.

***

### Базовая структура

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name string
        a, b int
        want int
    }{
        {"positive", 2, 3, 5},
        {"zero", 0, 0, 0},
        {"negative", -1, -2, -3},
        {"mixed", -5, 10, 5},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Add(tt.a, tt.b)
            if got != tt.want {
                t.Errorf("Add(%d, %d) = %d; want %d", tt.a, tt.b, got, tt.want)
            }
        })
    }
}
```

Анонимная структура описывает контракт: `name`, входные данные (`a`, `b`), ожидаемый результат (`want`). `t.Run(tt.name, ...)` создаёт подтест с именем кейса — в выводе будет видно, какой именно кейс упал.

***

### С `t.Parallel()` внутри подтестов

Каждый подтест можно запустить параллельно, захватив переменную цикла:

```go
for _, tt := range tests {
    tt := tt  // захват переменной — обязательно до Go 1.22
    t.Run(tt.name, func(t *testing.T) {
        t.Parallel()
        got := Add(tt.a, tt.b)
        if got != tt.want {
            t.Errorf("Add(%d, %d) = %d; want %d", tt.a, tt.b, got, tt.want)
        }
    })
}
```

До Go 1.22 переменная цикла `tt` переиспользовалась — все горутины видели последнее значение. `tt := tt` — идиома захвата. С Go 1.22 переменная цикла создаётся заново на каждой итерации — `tt := tt` больше не нужен.

***

### С `t.Cleanup` для общих ресурсов

```go
func TestDatabase(t *testing.T) {
    tests := []struct {
        name    string
        query   string
        want    []Row
    }{
        {"all users", "SELECT * FROM users", allUsers},
        {"by id", "SELECT * FROM users WHERE id = 1", user1},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            db := setupDB(t)
            t.Cleanup(func() { db.Close() })  // очистка после подтеста

            got := db.Query(tt.query)
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("Query(%q) returned unexpected result", tt.query)
            }
        })
    }
}
```

`t.Cleanup` регистрирует функцию очистки — она выполнится, когда тест и все его подтесты завершатся. Удобнее `defer`, потому что работает и при параллельных подтестах.

> **Зачем это Go-разработчику.** Табличные тесты — главный паттерн тестирования в Go. Если вы пишете отдельную `TestXxx`-функцию на каждый случай, вы делаете неправильно. Слайс структур + `t.Run` — один шаблон для всех тестов. Добавить кейс — одна строка. Увидеть, что тестируется — один взгляд на таблицу.

***

## 4. `t.Helper()` и вспомогательные функции

Повторяющийся код настройки (создание БД, поднятие сервера, подготовка фикстур) выносится в отдельные функции — test helpers. Но если такая функция вызывает `t.Fatal`, Go выводит номер строки **внутри помощника**, а не в тесте, где он был вызван. `t.Helper()` решает эту проблему.

***

### Проблема без `t.Helper()`

```go
func createTestDB(t *testing.T) *sql.DB {
    db, err := sql.Open("postgres", "host=localhost")
    if err != nil {
        t.Fatal(err)  // ошибка → строка в createTestDB, а не в тесте
    }
    return db
}

func TestCreateUser(t *testing.T) {
    db := createTestDB(t)  // при ошибке: testing.go:XXX (createTestDB) — неочевидно
    // ...
}
```

Вывод: `helper_test.go:5: connection refused` — строка 5 в `createTestDB`, а не строка вызова. Разработчик не понимает, какой тест упал.

### Решение: `t.Helper()`

```go
func createTestDB(t *testing.T) *sql.DB {
    t.Helper()  // ← эта функция — test helper, не показывать её в ошибках
    db, err := sql.Open("postgres", "host=localhost")
    if err != nil {
        t.Fatal(err)
    }
    return db
}
```

Теперь `t.Fatal` выводит **строку вызова** `createTestDB(t)`, а не строку внутри помощника. `t.Helper()` можно вызывать многократно — в цепочке вложенных помощников ошибка покажет самую внешнюю точку вызова в тесте.

***

### Вспомогательная функция с возвратом ошибки

Альтернатива `t.Fatal` — возврат ошибки и проверка на стороне теста:

```go
func createTestDB(t *testing.T) (*sql.DB, error) {
    t.Helper()
    return sql.Open("postgres", "host=localhost")
}

func TestCreateUser(t *testing.T) {
    db, err := createTestDB(t)
    if err != nil {
        t.Fatalf("failed to create test DB: %v", err)
    }
    defer db.Close()
    // ...
}
```

Этот подход даёт тесту больше контроля: он сам решает, `Fatal` или `Error`. Помощник только создаёт ресурс.

***

### `t.TempDir()` — встроенный помощник

Для временных файлов не нужно писать свой helper:

```go
func TestProcessFile(t *testing.T) {
    dir := t.TempDir()  // создаёт временную директорию и удаляет после теста
    path := filepath.Join(dir, "input.txt")
    os.WriteFile(path, []byte("hello"), 0644)

    result := ProcessFile(path)
    if result != "HELLO" {
        t.Errorf("ProcessFile = %q; want %q", result, "HELLO")
    }
}
```

`t.TempDir()` эквивалентен `os.MkdirTemp` + `t.Cleanup(func() { os.RemoveAll(dir) })` — создаёт директорию и автоматически удаляет после теста.

> **Зачем это Go-разработчику.** `t.Helper()` — не опциональный декоратор, а обязательная строка в каждой вспомогательной функции. Без него trace при падении уводит внутрь помощника. С ним — показывает строку в тесте, где помощник был вызван. `t.TempDir()` избавляет от рутины с временными файлами.

***

## 5. `t.Parallel()` и параллельные тесты

`t.Parallel()` помечает тест или подтест как параллельный. Go-раннер запускает такие тесты одновременно в нескольких горутинах. По умолчанию `-parallel` равен `GOMAXPROCS`, но можно явно задать: `go test -parallel 8`.

Параллельность особенно полезна с табличными тестами: каждый кейс в таблице — кандидат на независимый запуск.

***

### Параллельные тесты верхнего уровня

```go
func TestA(t *testing.T) {
    t.Parallel()
    time.Sleep(100 * time.Millisecond)
    // проверки...
}

func TestB(t *testing.T) {
    t.Parallel()
    time.Sleep(100 * time.Millisecond)
    // проверки...
}
```

Оба теста выполняются одновременно, общее время ≈ 100ms вместо 200ms.

### Параллельные подтесты

Внутри табличного теста — `t.Parallel()` в каждом `t.Run`:

```go
func TestAdd(t *testing.T) {
    for _, tt := range tests {
        tt := tt  // захват до Go 1.22
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            got := Add(tt.a, tt.b)
            if got != tt.want {
                t.Errorf("Add(%d, %d) = %d; want %d", tt.a, tt.b, got, tt.want)
            }
        })
    }
}
```

Важно: родительский тест ждёт завершения всех параллельных подтестов перед выходом. `t.Cleanup` работает корректно при параллельных подтестах.

***

### Гонки в тестах

Параллельные тесты используют общий `*testing.T` — но это безопасно, потому что каждый подтест получает свой `t`. Проблема — в разделяемых данных:

```go
var globalDB *sql.DB  // ГОНКА при параллельных тестах!

func TestA(t *testing.T) {
    t.Parallel()
    globalDB.Exec("INSERT INTO users ...")  // data race
}

func TestB(t *testing.T) {
    t.Parallel()
    globalDB.Exec("INSERT INTO users ...")  // data race
}
```

Решение: каждый тест создаёт свои ресурсы или использует `sync.Mutex` для общего.

### Совместимость с `TestMain`

`TestMain` выполняется в единственной горутине до всех тестов. Параллельные тесты запускаются после `m.Run()`. Порядок:

```
TestMain: setup → m.Run() { параллельные тесты } → teardown
```

```go
func TestMain(m *testing.M) {
    db := setupGlobalDB()
    code := m.Run()     // здесь TestA, TestB, TestC — параллельно
    db.Close()
    os.Exit(code)
}
```

> **Зачем это Go-разработчику.** `t.Parallel()` — простой способ ускорить тесты без изменения их структуры. Добавили одну строку — тесты поехали параллельно. Главная ловушка: разделяемое состояние (глобальные переменные, shared БД). Перед включением `t.Parallel()` убедитесь, что тесты независимы. Запуск с `-race` обязателен.

***

## 6. `TestMain` и настройка/очистка

`TestMain` — опциональная функция, выполняющая глобальную настройку и очистку для всего пакета тестов. Определяется один раз на пакет. При наличии `TestMain` Go не запускает тесты напрямую — вместо этого вызывает `TestMain`, а та через `m.Run()` запускает все тесты пакета.

***

### `TestMain` для глобального состояния

```go
var testDB *sql.DB

func TestMain(m *testing.M) {
    // Настройка: выполняется один раз перед всеми тестами
    var err error
    testDB, err = sql.Open("postgres", "host=localhost dbname=test")
    if err != nil {
        log.Fatal(err)
    }
    defer testDB.Close()

    // Миграции
    if err := migrate(testDB); err != nil {
        log.Fatal(err)
    }

    // Запуск всех тестов
    code := m.Run()

    // Очистка: после всех тестов
    testDB.Exec("TRUNCATE users")  // очистка после всех тестов
    os.Exit(code)
}
```

`m.Run()` возвращает код выхода: 0 — успех, 1 — есть упавшие тесты. Его нужно передать в `os.Exit`. Без `os.Exit` тесты отработают, но процесс не завершится с правильным кодом.

> **Зачем это Go-разработчику.** `TestMain` — для дорогой настройки: поднятие тестовой БД, миграции, запуск Docker-контейнера. Не используйте `TestMain` ради пары переменных — для этого есть `t.Cleanup` внутри теста.

***

### `t.Cleanup` vs `defer`

`t.Cleanup` — современная альтернатива `defer` для тестов. Регистрирует функцию очистки, которая выполнится, когда тест и все его подтесты завершатся:

```go
func TestWithCleanup(t *testing.T) {
    db := setupDB(t)
    t.Cleanup(func() {
        db.Close()
    })
    // ... тест ...
}
```

Преимущества `t.Cleanup` над `defer`:

| `t.Cleanup` | `defer` |
|---|---|
| Работает во вспомогательных функциях (очистка привязана к вызывающему тесту) | Очистка привязана к функции, где вызван |
| Корректно работает с параллельными подтестами | Может выполниться до завершения подтестов |
| Можно зарегистрировать несколько — выполняются в обратном порядке (LIFO) | Аналогично |

Типичный паттерн — помощник создаёт ресурс и регистрирует очистку:

```go
func createTestDB(t *testing.T) *sql.DB {
    t.Helper()
    db, err := sql.Open("postgres", "...")
    if err != nil {
        t.Fatal(err)
    }
    t.Cleanup(func() { db.Close() })
    return db
}

// В тесте: вызов createTestDB(t) — и забыли об очистке
```

> **Зачем это Go-разработчику.** `t.Cleanup` — предпочтительный способ очистки в тестах. Он решает проблему `defer` в помощниках (очистка живёт до конца теста, а не до выхода из функции-помощника) и безопасен при параллельных подтестах. Используйте `t.Cleanup` по умолчанию, `TestMain` — только для глобального состояния.

***

## 7. Бенчмарки

Бенчмарк — функция с сигнатурой `func BenchmarkXxx(b *testing.B)` в `_test.go`. Go запускает её многократно, увеличивая `b.N` до стабилизации времени. Запуск: `go test -bench=.` (все бенчмарки) или `go test -bench=BenchmarkFoo`.

Бенчмарки не запускаются при обычном `go test` — только с флагом `-bench`.

***

### Минимальный бенчмарк

```go
func BenchmarkAdd(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Add(10, 20)
    }
}
```

`b.N` — количество итераций. Go сам подбирает его: начинает с 1, удваивает, пока время не стабилизируется (минимум 1 секунда). Вывод:

```
BenchmarkAdd-8    100000000    10.5 ns/op
```

`-8` — количество потоков (`GOMAXPROCS`). `100000000` — `b.N`. `10.5 ns/op` — время одной итерации.

***

### `b.ResetTimer` и настройка

Если настройка занимает время, но не должна учитываться в результате — `b.ResetTimer()`:

```go
func BenchmarkSearch(b *testing.B) {
    data := generateHugeDataset()  // долгая настройка
    b.ResetTimer()                 // сбрасываем таймер

    for i := 0; i < b.N; i++ {
        Search(data, "needle")
    }
}
```

`b.StopTimer()` / `b.StartTimer()` для пауз:

```go
func BenchmarkWithSetup(b *testing.B) {
    for i := 0; i < b.N; i++ {
        b.StopTimer()
        data := loadFromFile()  // не учитываем в бенчмарке
        b.StartTimer()

        Process(data)
    }
}
```

***

### Полезные флаги

| Флаг | Что делает |
|---|---|
| `-bench=.` | Запустить все бенчмарки |
| `-bench=Add` | Только бенчмарки, содержащие "Add" в имени |
| `-benchmem` | Показывать аллокации (`allocs/op`) и байты (`B/op`) |
| `-benchtime=5s` | Минимальное время бенчмарка (по умолчанию 1s) |
| `-count=5` | Повторить бенчмарк 5 раз — для стабильности результатов |

С `-benchmem`:

```go
func BenchmarkStringConcat(b *testing.B) {
    for i := 0; i < b.N; i++ {
        _ = "hello" + " " + "world"
    }
}
```

```
BenchmarkStringConcat-8    500000000    3.2 ns/op    0 B/op    0 allocs/op
```

`0 B/op, 0 allocs/op` — компилятор оптимизировал конкатенацию констант (escape analysis).

***

### Сравнение бенчмарков

Типичный сценарий — сравнить два подхода:

```go
func BenchmarkConcatPlus(b *testing.B) {
    for i := 0; i < b.N; i++ {
        s := ""
        for j := 0; j < 100; j++ {
            s += "x"
        }
    }
}

func BenchmarkConcatBuilder(b *testing.B) {
    for i := 0; i < b.N; i++ {
        var sb strings.Builder
        for j := 0; j < 100; j++ {
            sb.WriteString("x")
        }
        _ = sb.String()
    }
}
```

Запуск: `go test -bench=. -benchmem`. Результат покажет, какой подход быстрее и сколько памяти тратит.

> **Зачем это Go-разработчику.** Бенчмарк — не микрооптимизация, а способ принимать обоснованные решения. `strings.Builder` или `+=`? `sync.Mutex` или `sync.RWMutex`? Бенчмарк отвечает числом, а не догадкой. `-benchmem` обязателен: время может быть одинаковым, а аллокации — разными на порядок.

***

## 8. Фаззинг (Go 1.18+)

Фаззинг — тестирование случайными входными данными. Вместо конкретных значений вы описываете **типы** аргументов, а Go генерирует значения и ищет такие, на которых тест падает. Особенно эффективен для парсеров, кодировщиков и обработки пользовательского ввода.

Запуск: `go test -fuzz=.` или `go test -fuzz=FuzzFoo`.

***

### Фазз-тест

```go
func FuzzDiv(f *testing.F) {
    // Начальный корпус — значения, с которых фаззер начинает поиск
    f.Add(10, 2)
    f.Add(0, 5)

    f.Fuzz(func(t *testing.T, a, b int) {
        _, err := Div(a, b)
        if err != nil && b != 0 {
            t.Errorf("Div(%d, %d) unexpected error: %v", a, b, err)
        }
        if err == nil && b == 0 {
            t.Error("Div should return error when dividing by zero")
        }
    })
}
```

Сигнатура: `func FuzzXxx(f *testing.F)`. Типы аргументов `f.Fuzz` ограничены: `int`, `int8`–`int64`, `uint`-варианты, `float32`, `float64`, `string`, `[]byte`, `bool`.

`f.Add` добавляет значения в начальный корпус — фаззер стартует с них и мутирует, пытаясь найти падающие комбинации.

***

### Как работает фаззер

1. Берёт значение из корпуса
2. Мутирует его (меняет байты, добавляет, удаляет)
3. Прогоняет через `f.Fuzz`
4. Если тест упал — сохраняет значение в `testdata/fuzz/FuzzXxx/` и продолжает поиск
5. Если тест прошёл — добавляет значение в корпус и мутирует дальше

Фаззер работает, пока не найдёт ошибку или пока его не остановят (`-fuzztime=30s`). Найденные значения можно закоммитить — они станут частью начального корпуса и будут прогоняться при каждом `go test` (без флага `-fuzz`).

***

### Кеширование и корпус

Без флага `-fuzz` фазз-тест прогоняет только начальный корпус — как обычный тест. С `-fuzz=.` запускается полный фаззинг.

```bash
go test -fuzz=. -fuzztime=30s   # фаззинг 30 секунд
go test -fuzz=FuzzDiv           # один конкретный фазз-тест
```

Найденные падающие значения сохраняются в `testdata/fuzz/`. При следующем запуске они будут протестированы — регрессия не пройдёт.

> **Зачем это Go-разработчику.** Фаззинг находит баги, которые вы не придумали бы вручную: невалидный UTF-8, деление на ноль с неожиданными комбинациями, переполнения. Один `FuzzParser` может найти больше крайних случаев, чем сотня ручных тестов. Добавьте фазз-тест к любому коду, принимающему внешние данные: парсерам, валидаторам, сериализаторам.

***

## 9. Примеры как тесты

`ExampleXxx` — функция, которая одновременно служит документацией и тестом. Выводит данные в `stdout`, а Go сверяет вывод с комментарием `// Output:`. Если вывод не совпадает — тест падает.

Примеры отображаются в `pkg.go.dev` как документация к пакету, функции или типу.

***

### Example-функция

```go
func ExampleAdd() {
    fmt.Println(Add(2, 3))
    // Output: 5
}
```

Запуск: `go test -v` (вместе с остальными тестами). Если вывод не совпадает — ошибка с diff.

Несколько примеров на одну функцию:

```go
func ExampleAdd_positive() {
    fmt.Println(Add(2, 3))
    // Output: 5
}

func ExampleAdd_negative() {
    fmt.Println(Add(-2, -3))
    // Output: -5
}
```

Суффикс после `_` — имя варианта: `ExampleAdd_positive`, `ExampleAdd_negative`.

***

### Неупорядоченный вывод

Если порядок строк не гарантирован — `// Unordered output:`:

```go
func ExampleMap() {
    m := map[string]int{"a": 1, "b": 2}
    for k, v := range m {
        fmt.Println(k, v)
    }
    // Unordered output:
    // a 1
    // b 2
}
```

Go проверяет, что каждая строка присутствует в выводе, но не сверяет порядок.

***

### Example без проверки вывода

Если не указать `// Output:`, пример компилируется, но не тестируется — только документация:

```go
func ExampleServer() {
    srv := &http.Server{Addr: ":8080"}
    log.Println(srv.ListenAndServe())
}
```

Такие примеры отображаются в `pkg.go.dev` с кнопкой «Run», но не проверяются автоматически.

> **Зачем это Go-разработчику.** Примеры решают две задачи разом: документируют API для пользователей пакета и гарантируют, что документация не устарела. Если `// Output:` изменился при рефакторинге — тест упадёт, и вы обновите документацию сразу, а не через месяц по баг-репорту.

***

## 10. Покрытие кода

Go измеряет покрытие на уровне инструкций: процент строк, выполненных хотя бы раз во время тестов. Запуск: `go test -cover`.

Покрытие — не цель, а инструмент. 100% покрытие не гарантирует отсутствия багов, но 0% — гарантирует, что код никто не проверял.

***

### Базовое покрытие

```bash
go test -cover
# ok   example/math   0.123s   coverage: 85.7% of statements
```

Вывод показывает процент покрытых инструкций. Каждый `if`, `return`, вызов функции считается инструкцией.

### Профиль покрытия

`-coverprofile` сохраняет подробные данные в файл:

```bash
go test -coverprofile=coverage.out
go tool cover -html=coverage.out    # открыть в браузере
go tool cover -func=coverage.out    # таблица покрытия по функциям
```

HTML-отчёт подсвечивает строки: зелёный — покрыто, красный — не покрыто, серый — неинструментировано (объявления, импорты).

```bash
go tool cover -func=coverage.out
# example/math/add.go:5:   Add      100.0%
# example/math/div.go:3:   Div       75.0%
# total:                   (statements)  87.5%
```

Видно, какая функция покрыта меньше — туда и добавлять тесты.

***

### Какой процент считать достаточным

Не гонитесь за 100%. Покрытие измеряет **выполнение**, а не **корректность**. Тест может покрывать строку, но не проверять граничные случаи.

Прагматичный подход:
- **Критический код** (платежи, аутентификация) — 90%+, с проверкой всех ветвлений
- **Бизнес-логика** — 70–80%, с табличными тестами на основные сценарии
- **Связующий код** (контроллеры, middleware) — 50–70%, интеграционные тесты
- **Геттеры, сеттеры, логирование, конфигурация** — не покрывать, если тривиально

`go test -cover` должен быть частью CI — падение при снижении покрытия ниже порога:

```bash
go test -cover -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//'
# сравниваем с порогом, например 70%
```

> **Зачем это Go-разработчику.** Покрытие отвечает на вопрос «какой код НЕ тестируется вообще». Это карта слепых зон, а не показатель качества. Смотрите на покрытие в диффе PR: если новая функция имеет 0% покрытия — задумайтесь. Если рефакторинг существующего кода без тестов — покройте тестами в том же PR. `go tool cover -html` — быстрый способ найти, что именно не покрыто.

***

## 11. Мокирование и интерфейсы

Мокирование в Go строится на интерфейсах. Зависимость (БД, HTTP-клиент, файловая система) описывается интерфейсом. В тестах реальная реализация заменяется моком — объектом, который возвращает предопределённые данные и записывает вызовы.

Без моков тесты становятся интеграционными: подними PostgreSQL, настрой Redis, дождись Kafka. Моки превращают их в модульные — быстро, изолированно, без сети.

***

### Ручной мок

Минимальный мок — структура, реализующая интерфейс:

```go
type UserRepo interface {
    GetByID(id int) (*User, error)
    Save(user *User) error
}

// Ручной мок
type mockUserRepo struct {
    users map[int]*User
    saveCalled int
}

func (m *mockUserRepo) GetByID(id int) (*User, error) {
    u, ok := m.users[id]
    if !ok {
        return nil, fmt.Errorf("user %d not found", id)
    }
    return u, nil
}

func (m *mockUserRepo) Save(user *User) error {
    m.saveCalled++
    m.users[user.ID] = user
    return nil
}

func TestGetUser(t *testing.T) {
    repo := &mockUserRepo{
        users: map[int]*User{1: {ID: 1, Name: "Alice"}},
    }
    svc := NewUserService(repo)

    user, err := svc.GetUser(1)
    if err != nil {
        t.Fatal(err)
    }
    if user.Name != "Alice" {
        t.Errorf("got %q; want Alice", user.Name)
    }
    if repo.saveCalled != 0 {
        t.Error("Save should not be called")
    }
}
```

Плюсы: полный контроль, никаких зависимостей. Минусы: много кода для больших интерфейсов.

***

### Генерация моков через `gomock`

`gomock` (uber-go/mock) генерирует мок из интерфейса автоматически:

```go
// user_repo.go
type UserRepo interface {
    GetByID(id int) (*User, error)
    Save(user *User) error
}
```

```bash
go install go.uber.org/mock/mockgen@latest
mockgen -source=user_repo.go -destination=mock_user_repo.go -package=main
```

В тесте:

```go
func TestGetUser(t *testing.T) {
    ctrl := gomock.NewController(t)
    repo := NewMockUserRepo(ctrl)

    repo.EXPECT().GetByID(1).Return(&User{Name: "Alice"}, nil)

    svc := NewUserService(repo)
    user, err := svc.GetUser(1)
    if err != nil {
        t.Fatal(err)
    }
    if user.Name != "Alice" {
        t.Errorf("got %q; want Alice", user.Name)
    }
}
```

`EXPECT()` задаёт ожидания: какие методы будут вызваны, с какими аргументами и что вернуть. Если ожидание не выполнилось — `gomock.NewController` упадёт в `t.Cleanup`.

`gomock.InOrder` проверяет порядок вызовов. `gomock.Any()` — любой аргумент. `Times(3)` — ровно 3 вызова.

> **Зачем это Go-разработчику.** Без интерфейсов мокирование в Go невозможно — это не недостаток, а архитектурное ограничение, заставляющее проектировать код с явными зависимостями. Ручной мок — для маленьких интерфейсов (1–3 метода). `gomock` — для больших, когда генерация экономит часы. Правило: интерфейс должен быть маленьким и принадлежать потребителю, а не реализации.

***

## 12. `httptest` для HTTP

Пакет `net/http/httptest` — два инструмента для тестирования HTTP. Подробный разбор — в [статье про HTTP](../architecture/HTTP/http.md) (раздел 15). Здесь — краткий обзор для контекста тестирования.

### `httptest.NewServer`

Запускает реальный HTTP-сервер на случайном порту. Для тестов, где нужен полный HTTP-стек:

```go
func TestGetUser(t *testing.T) {
    mux := http.NewServeMux()
    mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, `{"id": %s}`, r.PathValue("id"))
    })

    server := httptest.NewServer(mux)
    defer server.Close()

    resp, err := http.Get(server.URL + "/users/42")
    if err != nil {
        t.Fatal(err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Errorf("want 200, got %d", resp.StatusCode)
    }
}
```

### `httptest.NewRecorder`

Тестирует обработчик без сети — быстрее `NewServer`:

```go
func TestHandler(t *testing.T) {
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusCreated)
        w.Write([]byte("created"))
    })

    req := httptest.NewRequest("POST", "/users", nil)
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)

    if rec.Code != http.StatusCreated {
        t.Errorf("want 201, got %d", rec.Code)
    }
    if rec.Body.String() != "created" {
        t.Errorf("want 'created', got %q", rec.Body.String())
    }
}
```

> **Зачем это Go-разработчику.** `NewRecorder` для модульных тестов обработчиков, `NewServer` для интеграционных и тестирования middleware. Оба — часть stdlib, не требуют внешних библиотек. `rec.Result()` возвращает `*http.Response` — можно проверять заголовки, статус и тело через стандартный интерфейс.

***

## 13. Интеграционные тесты

Интеграционные тесты проверяют взаимодействие с реальными зависимостями: базой данных, брокером сообщений, внешним API. В отличие от модульных тестов с моками, здесь поднимается настоящая инфраструктура — обычно в Docker-контейнерах.

Запуск медленнее модульных тестов — их стоит вынести в отдельный этап CI или помечать build tag:

```go
//go:build integration

package db_test
```

Запуск: `go test -tags=integration ./...`.

***

### `testcontainers-go`

`testcontainers-go` управляет Docker-контейнерами из тестов: запускает PostgreSQL, Redis, Kafka и автоматически останавливает после теста:

```go
func TestCreateUser(t *testing.T) {
    ctx := context.Background()

    // Запускаем PostgreSQL в контейнере
    req := testcontainers.ContainerRequest{
        Image:        "postgres:16-alpine",
        ExposedPorts: []string{"5432/tcp"},
        Env: map[string]string{
            "POSTGRES_PASSWORD": "test",
            "POSTGRES_DB":       "testdb",
        },
        WaitingFor: wait.ForListeningPort("5432/tcp"),
    }
    postgres, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started: true,
    })
    if err != nil {
        t.Fatal(err)
    }
    defer postgres.Terminate(ctx)

    // Получаем порт и подключаемся
    port, _ := postgres.MappedPort(ctx, "5432")
    dsn := fmt.Sprintf("postgres://postgres:test@localhost:%s/testdb?sslmode=disable", port.Port())
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()

    // Миграции и тест
    migrate(db)
    repo := NewUserRepo(db)
    err = repo.Create(&User{Name: "Alice"})
    if err != nil {
        t.Fatal(err)
    }
}
```

Плюсы: точная копия продакшен-окружения, ловит проблемы с SQL-диалектами и совместимостью версий. Минусы: медленный запуск (секунды на контейнер), требует Docker.

Альтернатива: `dockertest` (ory/dockertest) — легковеснее, меньше зависимостей.

***

### Фикстуры и эталонные файлы

Go игнорирует директорию `testdata/` при сборке. Храните там фикстуры: SQL-дампы, JSON-файлы, эталонные файлы.

**Фикстура** — заранее подготовленные данные, которые приводят систему в известное состояние перед тестом. Это может быть дамп базы данных (`fixtures.sql`), набор JSON-файлов, тестовые изображения. Фикстуры гарантируют, что тест запускается на одних и тех же данных, независимо от окружения.

**Эталонный файл** (golden file) — сохранённый ожидаемый вывод, с которым тест сравнивает текущий результат. Вместо того чтобы вручную прописывать `want := "..."` в коде, вы сохраняете вывод в файл и сравниваете с ним. Флаг `-update` перезаписывает эталон при изменении поведения:

```
db_test.go
testdata/
    fixtures.sql       # схема и тестовые данные
    users_golden.json  # ожидаемый вывод API
```

Загрузка фикстуры:

```go
func loadFixture(t *testing.T, name string) []byte {
    t.Helper()
    data, err := os.ReadFile(filepath.Join("testdata", name))
    if err != nil {
        t.Fatal(err)
    }
    return data
}

func TestExport(t *testing.T) {
    users := loadFixture(t, "users_golden.json")
    var want []User
    json.Unmarshal(users, &want)

    got := ExportUsers(db)
    if !reflect.DeepEqual(got, want) {
        t.Errorf("export mismatch")
    }
}
```

> **Зачем это Go-разработчику.** Интеграционные тесты — последняя линия обороны перед деплоем. Модульные тесты с моками говорят «код корректен». Интеграционные — «код работает с настоящей БД». Оба нужны. `testcontainers-go` убирает главную боль интеграционных тестов: настройку окружения.

***

## 14. Паттерны и лучшие практики

Тестирование в Go — не только про `t.Error` и `t.Run`. Это про дисциплину: как структурировать тест, что тестировать, а что нет, и как не дать тестам превратиться в обузу.

***

### Arrange/Act/Assert

Структурируйте тест в три блока:

```go
func TestWithdraw(t *testing.T) {
    // Arrange: подготовка
    account := &Account{Balance: 100}

    // Act: действие
    err := account.Withdraw(30)

    // Assert: проверки
    if err != nil {
        t.Fatal(err)
    }
    if account.Balance != 70 {
        t.Errorf("balance = %d; want 70", account.Balance)
    }
}
```

Разделяйте блоки пустыми строками или комментариями. Это делает тест читаемым: сразу видно, что готовим, что делаем и что проверяем.

***

### Что тестировать

Не нужно покрывать тестами всё. Приоритет:

1. **Бизнес-логика** — расчёты, валидация, правила. То, что приносит деньги или может их потерять.
2. **Крайние случаи** — пустой ввод, отрицательные числа, деление на ноль, обрыв соединения.
3. **Публичный API** — тесты через `_test`-пакет гарантируют, что контракт не сломался.
4. **Интеграционные точки** — БД-запросы, вызовы внешних API. Тестировать с реальными зависимостями хотя бы в одном тесте.

Что НЕ тестировать:
- Геттеры и сеттеры — если в них нет логики
- Стандартную библиотеку — `json.Marshal` уже протестирован
- Сгенерированный код — если генератор проверен отдельно
- Тривиальные обёртки — `return repo.Find(id)` без дополнительной логики

***

### Тесты как документация

Хороший тест объясняет поведение лучше комментария. Имена тестов должны читаться как спецификация:

```go
// Плохо
func TestUser1(t *testing.T) {}
func TestUser2(t *testing.T) {}

// Хорошо
func TestUser_Activate(t *testing.T) {}
func TestUser_Deactivate_AlreadyInactive(t *testing.T) {}
func TestUser_Delete_WithOrders(t *testing.T) {}
```

Подтесты уточняют сценарий: `t.Run("negative balance", ...)`, `t.Run("empty name", ...)`. Разработчик, читающий вывод `go test -v`, понимает поведение без чтения кода.

***

### Белый и чёрный ящик

```go
package user        // белый ящик: доступ к приватным полям
package user_test   // чёрный ящик: только экспортируемый API
```

Тесты чёрного ящика (`_test`) проверяют поведение как внешний потребитель — они не сломаются при изменении внутренностей. Тесты белого ящика — для сложной внутренней логики, которую нельзя проверить через публичный API. Правило: начинайте с чёрного ящика, переходите к белому только при необходимости.

***

### Запуск тестов в CI

Минимальный набор:

```bash
go test -v -race -cover -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

- `-race` — детектор гонок (замедляет тесты, но обязателен)
- `-count=1` — отключает кеширование (в CI тесты всегда свежие)
- `-shuffle=on` — случайный порядок тестов (ловит зависимости между тестами)
- `-timeout=5m` — защита от зависших тестов

```bash
go test -race -count=1 -shuffle=on -timeout=5m ./...
```

### Чего избегать

- **assert-библиотек.** `testify/assert` популярен, но скрывает детали: `assert.Equal(t, want, got)` менее информативен, чем `if got != want { t.Errorf(...) }`. Плюс добавляет зависимость.
- **Моков всего подряд.** Если мок повторяет реальную логику — это двойная работа. Мокайте только внешние границы (БД, HTTP, файлы).
- **Слишком больших тестов.** Тест на 200 строк — признак, что тестируемый код делает слишком много.
- **Тестов, зависящих от порядка.** Каждый тест должен работать изолированно. `-shuffle=on` помогает это проверить.

> **Зачем это Go-разработчику.** Тесты — это код, который нужно поддерживать. Плохие тесты хуже отсутствия тестов: они дают ложное чувство безопасности и замедляют рефакторинг. Хорошие — документируют поведение, ловят регрессии и позволяют менять код без страха. Три правила: чёрный ящик по умолчанию, табличные тесты для набора случаев, никаких assert-библиотек без крайней необходимости.

***

## Источники

### Официальная документация

* [pkg.go.dev/testing](https://pkg.go.dev/testing) — полная документация пакета `testing`
* [Go Blog: Using Subtests and Sub-benchmarks](https://go.dev/blog/subtests) — `t.Run`, `t.Parallel`, `TestMain`
* [Go Blog: Fuzzing in Go](https://go.dev/blog/fuzz-beta) — фаззинг: `f.Add`, `f.Fuzz`, корпус
* [Go Blog: The cover story](https://go.dev/blog/cover) — `go test -cover`, `-coverprofile`, `go tool cover`
* [Go Wiki: TableDrivenTests](https://github.com/golang/go/wiki/TableDrivenTests) — канонический пример табличных тестов

### Статьи

* [Prefer table driven tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests) — Dave Cheney о табличных тестах
* [How to write benchmarks in Go](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go) — Dave Cheney: `b.N`, `-benchmem`
* [5 simple tips for writing unit tests in Go](https://medium.com/@matryer/5-simple-tips-and-tricks-for-writing-unit-tests-in-golang-619653f90742) — Mat Ryer: `t.Helper`, test helpers
* [Testing in Go: Subtests](https://ieftimov.com/posts/testing-in-go-subtests/) — Ilija Eftimov: глубокий разбор `t.Run`
* [Testing in Go: Clean Tests Using t.Cleanup](https://ieftimov.com/posts/testing-in-go-clean-tests-using-t-cleanup/) — Ilija Eftimov: `t.Cleanup` vs `defer`

### Книги

* [Learn Go with Tests](https://quii.gitbook.io/learn-go-with-tests) — Chris James, бесплатная онлайн-книга: TDD на Go от основ до моков
* [100 Go Mistakes and How to Avoid Them](https://100go.co/) — Teiva Harsanyi, глава 11: типичные ошибки в тестах

### Инструменты

* [gomock](https://github.com/uber-go/mock) — генерация моков из интерфейсов
* [testcontainers-go](https://golang.testcontainers.org/) — Docker-контейнеры в тестах
* [dockertest](https://github.com/ory/dockertest) — легковесная альтернатива testcontainers
