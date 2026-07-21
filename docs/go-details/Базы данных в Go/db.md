# Базы данных в Go

## 1. Введение

Любое веб-приложение, которое хранит данные между перезапусками, нуждается в базе данных. Пользователи, заказы, товары, логи — всё это должно пережить рестарт сервера. Файлы для этого не подходят: конкурентный доступ, поиск и целостность данных требуют специализированного инструмента.

Go работает с базами данных через пакет `database/sql` — минималистичный интерфейс, который не зависит от конкретной СУБД. Вы пишете один и тот же код для PostgreSQL, MySQL и SQLite — меняется только драйвер. Поверх `database/sql` существуют расширения (`sqlx`, `pgx`) и ORM (GORM), но ядро всегда одно.

Эта статья идёт от основ: сначала разбирается, что такое реляционная база данных и SQL, затем ACID и транзакции, и только потом — Go-специфичные инструменты. Такой порядок потому, что нельзя эффективно использовать `database/sql`, не понимая, что происходит на стороне базы данных.

В этой статье разобраны:

* реляционные базы данных: таблицы, ключи, отношения
* SQL: запросы, соединения, фильтрация
* ACID и транзакции: что гарантирует база данных
* пакет `database/sql`: подключение, запросы, prepared statements
* транзакции в Go: `Begin`, `Commit`, `Rollback`
* миграции: версионирование схемы через `golang-migrate`
* `sqlx`: сканирование в структуры, именованные параметры
* `pgx`: нативный драйвер PostgreSQL
* ORM на примере GORM: плюсы и минусы
* паттерн репозиторий: абстракция над БД
* SQL-инъекции и безопасность
* производительность: индексы, `EXPLAIN`, N+1
* тестирование с БД: `testcontainers-go`, транзакционные тесты
* SQLite: встраиваемая БД для тестов и локальных приложений

> **Зачем это Go-разработчику.** База данных — самый надёжный источник состояния в системе. Если вы теряете данные — вы теряете бизнес. А потерять их легко: забытая транзакция, неправильный уровень изоляции, N+1 запрос под нагрузкой. Понимание базы данных от SQL до пула соединений — обязательный навык бэкенд-разработчика, а не опция.

***

## 2. Что такое реляционная база данных

Реляционная база данных — это набор **таблиц**, связанных между собой отношениями (relations — отсюда и «реляционная»). Каждая таблица — прямоугольная сетка из **строк** (records) и **столбцов** (columns). Столбцы имеют фиксированный тип, строки добавляются и удаляются.

В отличие от файлов или NoSQL-хранилищ, реляционная БД гарантирует **целостность данных**: нельзя удалить пользователя, у которого остались заказы (если настроен внешний ключ), нельзя записать строку с невалидным email (если настроен constraint).

***

### Таблицы, строки, столбцы

Пример таблицы `users`:

```
 id |  name   |        email         | created_at
----+---------+----------------------+------------
  1 | Alice   | alice@example.com    | 2025-01-01
  2 | Bob     | bob@example.com      | 2025-06-15
  3 | Charlie | charlie@example.com  | 2025-12-31
```

Столбцы: `id` (число), `name` (строка), `email` (строка), `created_at` (дата). Строки — конкретные пользователи. Каждый столбец имеет тип, определённый при создании таблицы:

```sql
CREATE TABLE users (
    id         SERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    email      TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT now()
);
```

`SERIAL` — автоинкрементное число. `NOT NULL` — значение обязательно. `UNIQUE` — не может быть дубликатов. `DEFAULT` — значение по умолчанию.

***

### Первичный ключ

**Первичный ключ** (PRIMARY KEY) — столбец или комбинация столбцов, однозначно идентифицирующая строку. Обычно это `id` с автоинкрементом или UUID. Первичный ключ гарантирует, что каждая строка уникальна и не может быть `NULL`.

Без первичного ключа нельзя надёжно сослаться на строку из другой таблицы.

### Внешний ключ и отношения

**Внешний ключ** (FOREIGN KEY) — столбец, который ссылается на первичный ключ другой таблицы. Он создаёт **отношение** между таблицами:

```sql
CREATE TABLE orders (
    id      SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),  -- внешний ключ
    amount  DECIMAL NOT NULL,
    created_at TIMESTAMP DEFAULT now()
);
```

Типы отношений:

**Один-ко-многим.** Самый частый случай. Один пользователь → много заказов. Внешний ключ на стороне «многих»:

```
users:  id=1 (Alice)
orders: id=1, user_id=1, amount=100
        id=2, user_id=1, amount=250
```

**Один-к-одному.** Один пользователь → один профиль. Внешний ключ с `UNIQUE`:

```sql
CREATE TABLE profiles (
    id      SERIAL PRIMARY KEY,
    user_id INTEGER UNIQUE REFERENCES users(id),
    bio     TEXT
);
```

**Многие-ко-многим.** Студент ↔ курс. Промежуточная таблица:

```sql
CREATE TABLE enrollments (
    student_id INTEGER REFERENCES students(id),
    course_id  INTEGER REFERENCES courses(id),
    PRIMARY KEY (student_id, course_id)
);
```

***

### Реляционная БД vs NoSQL

| Критерий | Реляционная (PostgreSQL) | NoSQL (MongoDB) |
|---|---|---|
| Схема | Фиксированная, миграции | Гибкая, schema-less |
| Связи | Внешние ключи, JOIN | Вложенные документы, денормализация |
| Целостность | ACID, constraints | Обычно eventual consistency |
| Запросы | SQL | Свой язык (MQL, N1QL) |
| Когда выбрать | Данные структурированы и связаны | Слабоструктурированные, высокая скорость записи |

Правило: если данные имеют чёткую структуру и связи между собой (а большинство бизнес-данных именно такие) — берите реляционную БД. NoSQL — для специфических случаев: логи, кеш, временные ряды.

> **Зачем это Go-разработчику.** В Go вы работаете с БД через структуры: `type User struct { ID int; Name string }`. Столбцы таблицы отображаются на поля структуры. Понимание первичных и внешних ключей напрямую влияет на дизайн структур: `Order.UserID int` — это отражение `user_id INTEGER REFERENCES users(id)`. Отношения между таблицами становятся отношениями между структурами.

***

## 3. Основы SQL

SQL (Structured Query Language) — язык запросов к реляционным базам данных. Четыре основные операции: SELECT (чтение), INSERT (вставка), UPDATE (обновление), DELETE (удаление). Даже используя ORM, вы пишете SQL — просто в обёртке методов.

Все примеры ниже — на диалекте PostgreSQL. Другие базы (MySQL, SQLite) имеют тот же базовый синтаксис.

***

### SELECT: чтение данных

```sql
-- Все строки и столбцы
SELECT * FROM users;

-- Конкретные столбцы
SELECT id, name FROM users;

-- Фильтрация: WHERE
SELECT * FROM users WHERE name = 'Alice';

-- Несколько условий: AND, OR
SELECT * FROM orders WHERE amount > 100 AND created_at > '2025-01-01';

-- Сортировка: ORDER BY (ASC — по возрастанию, DESC — по убыванию)
SELECT * FROM users ORDER BY created_at DESC;

-- Ограничение количества: LIMIT
SELECT * FROM users ORDER BY created_at DESC LIMIT 10;

-- Пропуск строк: OFFSET (для пагинации)
SELECT * FROM users ORDER BY id LIMIT 10 OFFSET 20;  -- страница 3 по 10
```

***

### JOIN: соединение таблиц

JOIN связывает строки из разных таблиц по условию. Четыре основных типа:

| Тип | Что возвращает |
|---|---|
| `INNER JOIN` | Только строки с совпадениями в обеих таблицах |
| `LEFT JOIN` | Все строки из левой таблицы + совпадения из правой (NULL, если нет) |
| `RIGHT JOIN` | Все строки из правой таблицы + совпадения из левой (NULL, если нет) |
| `FULL OUTER JOIN` | Все строки из обеих таблиц (NULL, где нет совпадения) |

Плюс `CROSS JOIN` — декартово произведение (каждая строка левой × каждая строка правой), без условия.

Исходные данные для примеров:

```
users:                          orders:
 id | name                      id | user_id | amount
----+-------                   ----+---------+--------
 1  | Alice                      1 |    1    | 100.00
 2  | Bob                        2 |    1    | 250.00
 3  | Charlie                    3 |    2    |  75.00
                                 4 |  NULL   |  50.00   ← заказ без пользователя
```

**INNER JOIN** — только пересечение. Charlie и заказ #4 выпадают:

```sql
SELECT users.name, orders.amount
FROM users
INNER JOIN orders ON users.id = orders.user_id;
```

```
  name  | amount
--------+--------
 Alice  | 100.00
 Alice  | 250.00
 Bob    |  75.00
```

**LEFT JOIN** — все пользователи, даже без заказов:

```sql
SELECT users.name, orders.amount
FROM users
LEFT JOIN orders ON users.id = orders.user_id;
```

```
  name   | amount
---------+--------
 Alice   | 100.00
 Alice   | 250.00
 Bob     |  75.00
 Charlie | NULL      ← есть в users, нет в orders
```

**RIGHT JOIN** — все заказы, даже без пользователя:

```sql
SELECT users.name, orders.amount
FROM users
RIGHT JOIN orders ON users.id = orders.user_id;
```

```
  name  | amount
--------+--------
 Alice  | 100.00
 Alice  | 250.00
 Bob    |  75.00
 NULL   |  50.00   ← заказ #4: есть в orders, нет в users
```

> На практике `RIGHT JOIN` используют редко — тот же результат даёт `LEFT JOIN` с переставленными таблицами. Но в сложных запросах с множеством JOIN'ов он бывает удобнее.

**FULL OUTER JOIN** — всё из обеих таблиц. Charlie и заказ #4 — оба видны:

```sql
SELECT users.name, orders.amount
FROM users
FULL OUTER JOIN orders ON users.id = orders.user_id;
```

```
  name   | amount
---------+--------
 Alice   | 100.00
 Alice   | 250.00
 Bob     |  75.00
 Charlie | NULL      ← пользователь без заказов
 NULL    |  50.00    ← заказ без пользователя
```

**CROSS JOIN** — все комбинации. Для 3 пользователей × 4 заказа = 12 строк:

```sql
SELECT users.name, orders.id AS order_id
FROM users
CROSS JOIN orders;
```

Полезен для генерации сеток (например, все товары × все склады) и в тестах для заполнения данных.

> **Запомнить:** `LEFT JOIN` — сохраняем левую таблицу (основную). `RIGHT JOIN` — сохраняем правую. 99% случаев покрываются `INNER` и `LEFT`. `RIGHT` — редкость, `FULL OUTER` — для аудита и сверки данных.

***

### INSERT: вставка данных

```sql
-- Явное перечисление столбцов
INSERT INTO users (name, email) VALUES ('Diana', 'diana@example.com');

-- Несколько строк за раз
INSERT INTO users (name, email) VALUES
    ('Eve', 'eve@example.com'),
    ('Frank', 'frank@example.com');

-- С возвратом сгенерированного id (PostgreSQL)
INSERT INTO users (name, email) VALUES ('Grace', 'grace@example.com')
RETURNING id;
```

Всегда перечисляйте столбцы явно — это защищает от ошибок при изменении схемы таблицы.

***

### UPDATE: обновление данных

```sql
-- Обновить одну строку по id
UPDATE users SET name = 'Alice Johnson' WHERE id = 1;

-- Обновить несколько строк
UPDATE orders SET amount = amount * 1.1 WHERE created_at < '2025-01-01';
```

**Критично:** всегда используйте `WHERE`. `UPDATE users SET name = 'Oops'` без WHERE обновит ВСЕ строки.

***

### DELETE: удаление данных

```sql
-- Удалить по id
DELETE FROM users WHERE id = 99;

-- Удалить по условию
DELETE FROM orders WHERE created_at < '2024-01-01';
```

То же правило: `DELETE FROM users` без WHERE удалит всё.

### GROUP BY и агрегация

```sql
-- Сколько заказов у каждого пользователя
SELECT users.name, COUNT(orders.id) AS order_count
FROM users
LEFT JOIN orders ON users.id = orders.user_id
GROUP BY users.name;

-- Сумма заказов по пользователям, только те, у кого больше 500
SELECT users.name, SUM(orders.amount) AS total
FROM users
JOIN orders ON users.id = orders.user_id
GROUP BY users.name
HAVING SUM(orders.amount) > 500;
```

`WHERE` фильтрует строки ДО группировки, `HAVING` — ПОСЛЕ.

> **Зачем это Go-разработчику.** SQL — не «язык DBA», а ваш основной инструмент запросов к БД. ORM и `sqlx` не отменяют SQL — они лишь убирают рутину сканирования в структуры. `SELECT ... JOIN ... WHERE` вы будете писать руками в любом Go-проекте. `LEFT JOIN` вместо `INNER JOIN` — разница между «показать пользователя без заказов» и «потерять пользователя в выдаче».

***

## 4. ACID и транзакции

ACID — четыре свойства, гарантирующие надёжность операций в реляционной БД. Транзакция — набор SQL-команд, которые либо выполняются все вместе, либо не выполняются вообще.

***

### Atomicity — атомарность

Либо все операции транзакции применятся, либо ни одной. Если на середине произошла ошибка — БД откатывает уже сделанное.

```sql
-- Перевод денег: снять у Алисы, добавить Бобу. Обе операции или ни одной.
BEGIN;
UPDATE accounts SET balance = balance - 100 WHERE user_id = 1;
UPDATE accounts SET balance = balance + 100 WHERE user_id = 2;
COMMIT;
```

Упадёт соединение после первого UPDATE — ROLLBACK произойдёт автоматически.

### Consistency — согласованность

Транзакция переводит БД из одного согласованного состояния в другое. Ограничения (constraints), внешние ключи, триггеры — всё это обеспечивает согласованность на уровне схемы. Например, `balance` не может стать отрицательным, если есть `CHECK (balance >= 0)`.

### Isolation — изоляция

Одновременные транзакции не должны мешать друг другу. Чем выше уровень изоляции, тем меньше аномалий — но ниже производительность.

**Уровни изоляции** (от слабого к сильному):

| Уровень | Грязное чтение | Неповторяющееся чтение | Фантомы |
|---|---|---|---|
| Read Uncommitted | ✅ возможно | ✅ | ✅ |
| Read Committed | ❌ | ✅ возможно | ✅ |
| Repeatable Read | ❌ | ❌ | ✅ возможно |
| Serializable | ❌ | ❌ | ❌ |

**Аномалии:**

- **Грязное чтение** — транзакция A видит незакоммиченные изменения транзакции B. B откатывается — A работала с данными, которых «никогда не было».
- **Неповторяющееся чтение** — в рамках одной транзакции два одинаковых SELECT дают разный результат, потому что другая транзакция изменила и закоммитила строку между ними.
- **Фантомы** — транзакция A дважды читает по одному условию, но второй раз появляется НОВАЯ строка (другая транзакция вставила и закоммитила).

В PostgreSQL уровень по умолчанию — **Read Committed**. Для большинства приложений этого достаточно.

```sql
-- Установить уровень изоляции для текущей транзакции
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
-- ... операции ...
COMMIT;
```

### Durability — долговечность

Закоммиченная транзакция не теряется даже при сбое питания. Достигается через Write-Ahead Log (WAL) — изменения сначала пишутся в журнал на диск, и только потом применяются к данным.

***

### Управление транзакциями

```sql
BEGIN;                        -- начать транзакцию
UPDATE users SET name = 'Alice Johnson' WHERE id = 1;
-- ... другие операции ...
COMMIT;                       -- зафиксировать
-- или
ROLLBACK;                     -- откатить
```

**Savepoints** — точки отката внутри транзакции. Позволяют откатить часть изменений, не теряя всё:

```sql
BEGIN;
UPDATE users SET name = 'New Name' WHERE id = 1;
SAVEPOINT after_rename;       -- контрольная точка

UPDATE users SET email = 'bad' WHERE id = 1;
-- Ой, не то...
ROLLBACK TO after_rename;     -- откат только до savepoint

COMMIT;                       -- name изменён, email — нет
```

> **Зачем это Go-разработчику.** Транзакции — ваш главный инструмент целостности данных. Списываете деньги и не начисляете — без транзакции потеряете деньги или данные. `database/sql` даёт `db.Begin()`, дальше вы работаете с `*sql.Tx` как с `*sql.DB`. Уровни изоляции влияют на конкурентность: Serializable защитит от всего, но затормозит под нагрузкой — выбирайте осознанно. В финансовых операциях без транзакций не выжить.

***

## 5. Как программа подключается к базе данных

Подключение к БД — не просто «открыл сокет и посылаю SQL». Под капотом несколько слоёв, и каждый решает свою задачу.

***

### Путь запроса: от приложения до БД

```
Ваш код → database/sql → драйвер (pgx) → TCP :5432 → PostgreSQL
```

1. **Ваш код** — вы вызываете `db.Query("SELECT ...")` или метод ORM.
2. **database/sql** — стандартный пакет Go. Не знает SQL диалектов — предоставляет унифицированный интерфейс: открыть соединение, выполнить запрос, просканировать результат.
3. **Драйвер** — реализация интерфейса `database/sql/driver` под конкретную БД (`pgx`, `go-sql-driver/mysql`, `mattn/go-sqlite3`). Переводит вызовы `database/sql` в протокол конкретной базы.
4. **TCP-соединение** — драйвер открывает TCP-сокет до сервера БД (по умолчанию порт 5432 для PostgreSQL, 3306 для MySQL).
5. **СУБД** — принимает соединение, парсит SQL, выполняет, возвращает результат.

### Connection string

Строка подключения — URL или DSN (Data Source Name), из которого драйвер извлекает хост, порт, имя БД, пользователя и пароль.

PostgreSQL (драйвер `pgx`):

```
postgres://user:password@localhost:5432/mydb?sslmode=disable
```

Разбор по частям:

| Часть | Значение |
|---|---|
| `postgres://` | Схема (протокол), по ней database/sql выбирает драйвер |
| `user:password` | Учётные данные |
| `localhost:5432` | Хост и порт |
| `/mydb` | Имя базы данных |
| `?sslmode=disable` | Параметры (SSL, таймауты, пул) |

**SSL/TLS** (Secure Sockets Layer / Transport Layer Security) — протокол шифрования трафика между вашим приложением и БД. Без него SQL-запросы и ответы (включая пароль при аутентификации и сами данные) идут по сети открытым текстом — любой, кто прослушивает сеть, может их прочитать. SSL оборачивает TCP-соединение в шифрованный канал.

В PostgreSQL `sslmode` управляет поведением:

| sslmode | Поведение |
|---|---|
| `disable` | Без шифрования (только для локальной разработки) |
| `require` | Требовать SSL, но не проверять сертификат сервера |
| `verify-full` | SSL + проверка сертификата (production) |

> В production-окружении всегда используйте `sslmode=verify-full` или хотя бы `require`. `disable` допустим только на `localhost`.

MySQL:

```
user:password@tcp(localhost:3306)/mydb?parseTime=true
```

### Драйвер: мост между Go и БД

Драйвер — это пакет, реализующий низкоуровневый протокол общения с конкретной СУБД. Go не тащит все драйверы в стандартную библиотеку — вы подключаете нужный через blank import:

```go
import (
    "database/sql"
    _ "github.com/jackc/pgx/v5/stdlib" // blank import — регистрирует драйвер
)
```

`_` перед путём импорта означает: «загрузи пакет и выполни его `init()`, но сами функции пакета мне не нужны». Без `_` Go-компилятор откажется собирать код — неиспользуемый импорт это ошибка компиляции.

Как это работает под капотом:

1. При запуске программы Go проходит по всем импортам и для каждого пакета выполняет его `init()`-функции до вызова `main()`.
2. Внутри `pgx/stdlib` есть `init()`, который вызывает `sql.Register("pgx", &Driver{})` — кладёт драйвер в глобальный реестр `database/sql`.
3. Когда ваш код вызывает `sql.Open("pgx", "...")`, `database/sql` ищет в реестре драйвер с именем `"pgx"` и получает готовый `*Driver`.

Упрощённо, внутри пакета `pgx/stdlib` происходит это:

```go
package stdlib

import "database/sql"

func init() {
    sql.Register("pgx", &Driver{})
}

type Driver struct{ /* реализация database/sql/driver.Driver */ }
```

Аналогично работают и другие драйверы: `go-sql-driver/mysql` регистрирует `"mysql"`, `mattn/go-sqlite3` — `"sqlite3"`.

### Пул соединений

Открывать TCP-соединение на каждый запрос — дорого (TCP-handshake, TLS, аутентификация). `database/sql` держит **пул** — набор готовых соединений, которые переиспользуются между запросами.

```go
db, err := sql.Open("pgx", "postgres://user:pass@localhost:5432/mydb")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Настройка пула (подробнее в разделе 6)
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
```

`sql.Open` НЕ открывает соединение — только валидирует аргументы и инициализирует пул. Реальное соединение открывается лениво, при первом запросе. Чтобы проверить доступность сразу — `db.Ping()`.

> **Зачем это Go-разработчику.** Понимание слоёв экономит часы отладки. «Почему запрос висит?» — начните с `db.Stats()` (раздел 6). «Почему не подключается?» — проверьте connection string: пароль, хост, sslmode. `sql.Open` без `Ping` не падает при неверном адресе — популярная ловушка для новичков. Драйвер подключается через blank import: забудете `_ "..."` — получите `sql: unknown driver "pgx"`.

***

## 6. Пакет `database/sql`: подключение и пул

`sql.DB` — это не одно TCP-соединение, а пул. Он открывает, переиспользует и закрывает соединения автоматически. Ваша задача — правильно его настроить.

***

### Подключение: `sql.Open`

```go
import (
    "database/sql"
    _ "github.com/jackc/pgx/v5/stdlib"
)

db, err := sql.Open("pgx", "postgres://user:pass@localhost:5432/mydb?sslmode=disable")
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

Что происходит при `sql.Open`:
1. Ищет драйвер `"pgx"` в глобальном реестре (зарегистрирован через `init()`).
2. Создаёт `*sql.DB` — пул с нулевым количеством активных соединений.
3. **Не открывает TCP-соединение.** Только проверяет, что connection string не пустая.

Первое соединение откроется лениво — при первом `db.Query()`, `db.Exec()` или `db.Ping()`.

Проверка доступности сразу:

```go
if err := db.Ping(); err != nil {
    log.Fatalf("БД недоступна: %v", err)
}
```

> Всегда вызывайте `Ping` при старте приложения: неработающая БД обнаружится сразу, а не при первом запросе пользователя.

### Настройки пула

```go
db.SetMaxOpenConns(25)              // максимум открытых соединений
db.SetMaxIdleConns(10)              // максимум простаивающих (idle) соединений
db.SetConnMaxLifetime(5 * time.Minute)  // максимальное время жизни соединения
db.SetConnMaxIdleTime(1 * time.Minute)  // максимум простоя, после — закрыть
```

**SetMaxOpenConns** — верхняя граница. Когда все 25 заняты, следующий запрос ЖДЁТ. По умолчанию `0` = без ограничений — под нагрузкой можно открыть тысячи соединений и убить БД.

> Всегда выставляйте `SetMaxOpenConns`. Нет причин оставлять 0 в production.

**SetMaxIdleConns** — сколько соединений держать открытыми в простое. Когда `idle > MaxIdleConns`, лишние закрываются. По умолчанию `2`. Увеличьте, если приложение часто ходит в БД — переиспользование idle-соединения быстрее, чем открытие нового.

**SetConnMaxLifetime** — максимальный возраст соединения от момента открытия. После этого соединение закрывается и заменяется новым. Нужно для:
- Ротации подключений перед restart БД
- Обхода проблем с долгоживущими TCP-соединениями (балансировщики, NAT)

**SetConnMaxIdleTime** — максимальное время простоя. Соединение, не использовавшееся дольше этого времени, закрывается.

Рекомендации для среднего веб-сервиса (PostgreSQL):

| Параметр | Значение | Почему |
|---|---|---|
| `MaxOpenConns` | 20–50 | PostgreSQL эффективен с небольшим числом соединений |
| `MaxIdleConns` | 10–25 | Половина или меньше от MaxOpenConns |
| `ConnMaxLifetime` | 30–60m | Меньше таймаута балансировщика |
| `ConnMaxIdleTime` | 5–10m | Чтобы не копить idle-соединения |

### Мониторинг: `db.Stats()`

```go
stats := db.Stats()
fmt.Printf("Open: %d, Idle: %d, InUse: %d, WaitCount: %d\n",
    stats.OpenConnections, stats.Idle, stats.InUse, stats.WaitCount)
```

| Поле | Что значит |
|---|---|
| `OpenConnections` | Открыто сейчас |
| `Idle` | Простаивают |
| `InUse` | Заняты запросами |
| `WaitCount` | Сколько запросов ждали освобождения соединения |
| `WaitDuration` | Суммарное время ожидания |
| `MaxOpenConnections` | Лимит (из `SetMaxOpenConns`) |

Если `WaitCount` растёт — пул перегружен, увеличьте `MaxOpenConns` или оптимизируйте запросы.

### Жизненный цикл соединения в пуле

```
sql.Open() → пул пуст
               ↓
db.Query()  → открыть TCP-соединение → выполнить запрос → вернуть в idle
               ↓
след. Query → взять из idle (если есть) → выполнить запрос → вернуть в idle
               ↓
idle > MaxIdleConns → закрыть лишние
               ↓
возраст > ConnMaxLifetime → закрыть → следующий запрос откроет новое
```

### `db.Close()` и утечки

```go
defer db.Close()
```

`Close` закрывает ВСЕ соединения в пуле. Без него при завершении программы соединения повиснут до таймаута сервера БД.

> `*sql.DB` создаётся ОДИН раз на всё приложение. Не открывайте новый пул на каждый запрос — это создаёт новый пул соединений, который никогда не закроется.

> **Зачем это Go-разработчику.** `SetMaxOpenConns` без значения (0) — самая популярная причина падения БД под нагрузкой. `WaitCount` в `Stats()` — ваш главный индикатор: если растёт, пулу не хватает соединений. `Ping()` при старте спасает от дебага в 3 часа ночи. И никогда не создавайте `*sql.DB` в обработчике HTTP — это утечка соединений.

***

## 7. Запросы: `Query`, `QueryRow`, `Exec`

Три метода `*sql.DB` покрывают все SQL-операции: чтение многих строк, чтение одной строки, изменение данных.

***

### `Query` — множество строк

```go
rows, err := db.Query("SELECT id, name, email FROM users WHERE active = $1", true)
if err != nil {
    return err
}
defer rows.Close() // всегда закрывать!

for rows.Next() {
    var id int
    var name, email string
    if err := rows.Scan(&id, &name, &email); err != nil {
        return err
    }
    fmt.Printf("%d: %s <%s>\n", id, name, email)
}

if err := rows.Err(); err != nil { // проверить ошибки итерации
    return err
}
```

Ключевые моменты:
- `defer rows.Close()` — освобождает соединение обратно в пул. Без Close соединение утекает навсегда.
- `rows.Next()` возвращает `false`, когда строки кончились.
- `rows.Scan()` копирует значения столбцов в переменные по указателям. Порядок столбцов в Scan должен совпадать с порядком в SELECT.
- `rows.Err()` — после цикла проверяет, не было ли ошибок при итерации (разрыв соединения, неконвертируемый тип).

### `QueryRow` — одна строка

```go
var name, email string
err := db.QueryRow("SELECT name, email FROM users WHERE id = $1", 42).Scan(&name, &email)
if err == sql.ErrNoRows {
    // строки нет — это нормально, не ошибка
    return nil
}
if err != nil {
    return err
}
```

`QueryRow` возвращает `*sql.Row`. У него нет `Close()` — соединение освобождается автоматически при вызове `Scan`. Если `Scan` не вызван — соединение утекает.

> `sql.ErrNoRows` — не ошибка приложения. Это сигнал «данных нет», и обрабатывать его нужно отдельно от настоящих ошибок.

### `Exec` — изменение данных

```go
result, err := db.Exec(
    "INSERT INTO users (name, email) VALUES ($1, $2)",
    "Alice", "alice@example.com",
)
if err != nil {
    return err
}

id, _ := result.LastInsertId()     // сгенерированный id (не все драйверы поддерживают)
n, _ := result.RowsAffected()      // сколько строк затронуто
```

`Exec` — для INSERT, UPDATE, DELETE и DDL. Возвращает `sql.Result` с двумя полезными методами:
- `LastInsertId()` — работает с `SERIAL`/`AUTO_INCREMENT`, но **не поддерживается** PostgreSQL через `database/sql`. Для PostgreSQL используйте `RETURNING id` через `QueryRow`.
- `RowsAffected()` — всегда работает. Полезно проверить, что UPDATE/DELETE действительно что-то затронули.

Для PostgreSQL — возврат id через `RETURNING`:

```go
var id int
err := db.QueryRow(
    "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id",
    "Alice", "alice@example.com",
).Scan(&id)
```

### Плейсхолдеры: `$1` vs `?`

PostgreSQL использует нумерованные плейсхолдеры `$1, $2, $3`. MySQL и SQLite — вопросительные знаки `?, ?, ?`. Драйвер сам преобразует плейсхолдеры в безопасные значения — **никогда не подставляйте значения через `fmt.Sprintf`** (SQL-инъекции — раздел 13).

### NULL и nullable-типы

Колонка в БД может быть `NULL`. Go-типы вроде `string` и `int` не умеют хранить «отсутствие значения». Решение — nullable-типы из `database/sql`:

```go
var name sql.NullString
var age sql.NullInt64

db.QueryRow("SELECT name, age FROM users WHERE id = $1", 1).Scan(&name, &age)

if name.Valid {
    fmt.Println(name.String) // "Alice"
} else {
    fmt.Println("имя не указано")
}
```

| Тип | Поле значения |
|---|---|
| `sql.NullString` | `.String` |
| `sql.NullInt64` | `.Int64` |
| `sql.NullInt32` | `.Int32` |
| `sql.NullFloat64` | `.Float64` |
| `sql.NullBool` | `.Bool` |
| `sql.NullTime` | `.Time` |

Каждый имеет поле `Valid bool`. Если `Valid == false` — значение в БД было `NULL`.

Альтернатива — использовать указатели: `var name *string`. `Scan` запишет `nil`, если значение `NULL`. Но указатели размазывают nil-проверки по всему коду — nullable-типы понятнее.

### Сканирование в структуру

Вручную писать `Scan(&u.ID, &u.Name, &u.Email, ...)` для 15 полей — боль. `sqlx` (раздел 11) делает это автоматически, но на `database/sql` можно обойтись ручным маппингом или вспомогательной функцией:

```go
type User struct {
    ID    int
    Name  string
    Email string
}

func scanUser(rows *sql.Rows) (*User, error) {
    var u User
    return &u, rows.Scan(&u.ID, &u.Name, &u.Email)
}
```

> **Зачем это Go-разработчику.** `defer rows.Close()` — мелочь, без которой соединения утекают. `ErrNoRows` — не ошибка, а штатная ситуация. `LastInsertId` не работает в PostgreSQL — используйте `RETURNING`. И никогда не передавайте пользовательский ввод в SQL через `fmt.Sprintf` — только через плейсхолдеры `$1`, `$2`.

***

***

## 8. Подготовленные выражения

Подготовленное выражение (prepared statement) — SQL-запрос, заранее скомпилированный на сервере БД. Вы отправляете SQL один раз, получаете «ручку» (`*sql.Stmt`) и дёргаете её многократно с разными параметрами.

***

### `Prepare`: как это работает

```go
stmt, err := db.Prepare("SELECT name, email FROM users WHERE id = $1")
if err != nil {
    return err
}
defer stmt.Close()

// Многократное исполнение — запрос уже разобран и оптимизирован
rows, _ := stmt.Query(1)   // пользователь 1
rows, _ = stmt.Query(42)    // пользователь 42
rows, _ = stmt.Query(100)   // пользователь 100
```

Путь запроса без `Prepare`:

```
SQL-строка → парсинг → план запроса → выполнение → результат
```

Путь с `Prepare`:

```
SQL-строка → парсинг → план запроса (сохраняется!)
         ↓
параметры → выполнение плана → результат    ← без повторного парсинга
         ↓
параметры → выполнение плана → результат    ← ещё раз
```

БД парсит SQL и строит план запроса ОДИН раз. Дальше только подставляет параметры и выполняет.

### Методы `*sql.Stmt`

```go
stmt.Query(args...)       // множество строк
stmt.QueryRow(args...)    // одна строка
stmt.Exec(args...)        // изменение данных
```

Те же, что у `*sql.DB`, но SQL уже зафиксирован — передаёте только параметры.

### Когда `Prepare` полезен

- **Массовые вставки** — один `Prepare("INSERT INTO ...")` на тысячи строк. Без него каждая вставка парсится заново.
- **Повторяющиеся запросы** — `SELECT ... WHERE id = $1` для разных id в цикле.
- **Защита от инъекций** — параметры всегда передаются отдельно от SQL, их невозможно интерпретировать как часть запроса.

### Защита от SQL-инъекций

Главное правило: **никогда не подставляйте значения в SQL через `fmt.Sprintf`**.

```go
// УЯЗВИМО — так делать НЕЛЬЗЯ
userInput := "1; DROP TABLE users;"
query := fmt.Sprintf("SELECT * FROM users WHERE id = %s", userInput)
db.Query(query) // выполняет ДВА запроса: SELECT и DROP

// БЕЗОПАСНО — плейсхолдеры
db.Query("SELECT * FROM users WHERE id = $1", userInput)
// БД ищет id = '1; DROP TABLE users;' — строку, а не SQL-команду
```

Подготовленное выражение усиливает защиту: SQL и параметры передаются серверу БД **раздельно**. Сервер уже знает структуру запроса — параметр может быть только значением, не командой.

### Когда `Prepare` НЕ нужен

`Prepare` — это два сетевых round-trip: сначала `PREPARE`, потом `EXECUTE`. Для однократного запроса это медленнее, чем прямой `db.Query()`.

```go
// Для однократного запроса — проще и быстрее
db.Query("SELECT * FROM users WHERE id = $1", 42)

// Для многократного — Prepare
stmt, _ := db.Prepare("SELECT * FROM users WHERE id = $1")
for _, id := range ids {
    stmt.Query(id)
}
```

### `Prepare` на уровне `database/sql`

`database/sql` может эмулировать `Prepare` на стороне клиента, если драйвер не поддерживает серверные подготовленные выражения. Гарантий нет — но для `pgx` и `go-sql-driver/mysql` это настоящий серверный Prepare.

> **Зачем это Go-разработчику.** `Prepare` — ваш инструмент для пакетных операций. Тысяча вставок через `Prepare` в 5–10 раз быстрее, чем тысяча отдельных `db.Exec`. Никакой `fmt.Sprintf` для SQL — даже если вы «знаете, что вход безопасный». Завтра код поменяется, и безопасный вход станет опасным. Плейсхолдеры всегда.

***

## 9. Транзакции в Go

Теорию ACID разобрали в разделе 4. Здесь — как работать с транзакциями в коде.

***

### `Begin`, `Commit`, `Rollback`

```go
tx, err := db.Begin()
if err != nil {
    return err
}

_, err = tx.Exec("UPDATE accounts SET balance = balance - 100 WHERE user_id = $1", 1)
if err != nil {
    tx.Rollback()
    return err
}

_, err = tx.Exec("UPDATE accounts SET balance = balance + 100 WHERE user_id = $1", 2)
if err != nil {
    tx.Rollback()
    return err
}

tx.Commit()
```

`*sql.Tx` имеет те же методы, что и `*sql.DB`: `Query`, `QueryRow`, `Exec`, `Prepare`. Разница — все операции идут в рамках одной транзакции.

### Паттерн с `defer`

Писать `tx.Rollback()` перед каждым `return` — многословно и легко забыть. Стандартный паттерн:

```go
tx, err := db.Begin()
if err != nil {
    return err
}
defer tx.Rollback() // откат, если Commit не был вызван

// ... операции с tx ...

return tx.Commit() // Commit отменяет Rollback
```

Как это работает: `Rollback` после `Commit` — no-op (ничего не делает). Если до `Commit` случилась ошибка — `defer` гарантирует откат.

### Контекст: `BeginTx`

```go
tx, err := db.BeginTx(ctx, nil) // с контекстом для таймаутов и отмены
```

Опции через `*sql.TxOptions`:

```go
tx, err := db.BeginTx(ctx, &sql.TxOptions{
    Isolation: sql.LevelSerializable, // уровень изоляции
    ReadOnly:  true,                  // только чтение
})
```

### Транзакция и `Prepare`

Подготовленное выражение внутри транзакции привязано к ней:

```go
tx, _ := db.Begin()
defer tx.Rollback()

stmt, _ := tx.Prepare("INSERT INTO logs (msg) VALUES ($1)")
defer stmt.Close()

stmt.Exec("транзакция началась")
// ... другие операции ...
stmt.Exec("транзакция завершается")

tx.Commit() // обе вставки либо применятся, либо нет
```

`stmt` создан из `tx`, а не из `db` — его операции часть транзакции.

### Распространённые ошибки

**Использовать `db` вместо `tx` внутри транзакции:**

```go
tx, _ := db.Begin()
defer tx.Rollback()

db.Exec("UPDATE users SET name = $1 WHERE id = $2", "Alice", 1) // ОШИБКА: не в транзакции!
tx.Exec("UPDATE users SET name = $1 WHERE id = $2", "Bob", 2)

tx.Commit() // Bob обновился, Alice — мимо транзакции
```

Запрос через `db` выполняется в отдельном соединении, вне транзакции.

**Забыть `Commit`:**

```go
tx, _ := db.Begin()
defer tx.Rollback()

tx.Exec("INSERT INTO users (name) VALUES ($1)", "Alice")
// return без Commit → defer делает Rollback → Алиса не сохранилась
```

**Долгая транзакция** — блокировки держатся до `Commit`/`Rollback`. Не делайте HTTP-запросы или вызовы внешних API внутри транзакции.

> **Зачем это Go-разработчику.** `defer tx.Rollback()` — идиома, с которой начинается любая транзакция в Go. Не забывайте её — потеря данных стоит дороже двух строк кода. Транзакция через `db.BeginTx(ctx, ...)` позволяет отменить долгий запрос по таймауту контекста. И главное: внутри транзакции используйте `tx`, а не `db`.

***

## 10. Миграции

Миграции — способ версионировать схему базы данных. Вместо «в prod выкатились, а колонку забыли добавить» — SQL-файлы с изменениями, которые применяются последовательно.

Стандартный инструмент для Go — [`golang-migrate`](https://github.com/golang-migrate/migrate). Поддерживает PostgreSQL, MySQL, SQLite и десятки других БД.

***

### Как это работает

Каждая миграция — пара файлов: `up` (применить) и `down` (откатить):

```
migrations/
    000001_create_users.up.sql
    000001_create_users.down.sql
    000002_add_email_column.up.sql
    000002_add_email_column.down.sql
```

Номер версии — порядковый. `golang-migrate` хранит в БД таблицу `schema_migrations` с текущей версией и признаком dirty (незавершённая миграция).

```sql
-- 000001_create_users.up.sql
CREATE TABLE users (
    id   SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

-- 000001_create_users.down.sql
DROP TABLE users;
```

### CLI: создание и применение

```bash
# Создать новую пару миграций
migrate create -ext sql -dir migrations -seq create_users

# Применить все неприменённые
migrate -path migrations -database "postgres://user:pass@localhost:5432/mydb?sslmode=disable" up

# Откатить последнюю
migrate -path migrations -database "..." down 1

# Принудительно выставить версию (если миграция сломалась)
migrate -path migrations -database "..." force 5
```

### Программный API в Go

Миграции можно встроить прямо в код приложения — удобно для запуска при старте:

```go
import (
    "embed"
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    "github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func runMigrations(dbURL string) error {
    source, err := iofs.New(migrationsFS, "migrations")
    if err != nil {
        return err
    }

    m, err := migrate.NewWithSourceInstance("iofs", source, dbURL)
    if err != nil {
        return err
    }
    defer m.Close()

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return err
    }
    return nil
}
```

Директива `//go:embed migrations/*.sql` вкомпилирует SQL-файлы прямо в бинарник. Не надо таскать папку `migrations/` рядом с исполняемым файлом — один бинарник содержит всё.

### Правила хороших миграций

1. **Одна миграция — одно изменение.** Не мешайте создание таблицы и добавление колонки в другой таблице в одном файле.
2. **Пишите `down` всегда.** Даже если «мы никогда не откатываемся». Откат случится в самый неподходящий момент.
3. **Идемпотентность.** `up` должен падать, если изменения уже есть (или проверять `IF NOT EXISTS`). `down` — аналогично с `IF EXISTS`.
4. **`NOT NULL` с `DEFAULT`.** Добавление `NOT NULL` колонки без значения по умолчанию сломает существующие строки.
5. **Делайте миграции обратимыми.** `down` не обязан восстанавливать данные, но обязан восстанавливать схему.

### Dirty-состояние

Если `up`-миграция упала посреди выполнения, БД помечается как **dirty**. Повторный `up` невозможен — сначала нужно `force` до предыдущей версии и вручную откатить частично применённые изменения.

> Всегда тестируйте `up` и `down` локально перед применением в production.

> **Зачем это Go-разработчику.** Миграции — не роскошь, а необходимость. Без них схема БД живёт в головах разработчиков, и «я забыл выполнить SQL перед деплоем» становится нормой. `embed` + `iofs` — стандартный способ вкомпилировать миграции в бинарник на Go 1.16+. И всегда пишите `down`: откат без down-миграции — это ручная починка схемы в 4 утра.

***

***

## 11. `sqlx`: расширение `database/sql`

[`sqlx`](https://github.com/jmoiron/sqlx) — тонкая надстройка над `database/sql`. Не заменяет его, а убирает рутину: сканирование в структуры, именованные параметры, `IN` со срезами. Тот же пул соединений, те же драйверы — только меньше кода.

```go
import (
    "github.com/jmoiron/sqlx"
    _ "github.com/jackc/pgx/v5/stdlib"
)

db, _ := sqlx.Connect("pgx", "postgres://user:pass@localhost:5432/mydb?sslmode=disable")
```

`sqlx.Connect` = `sql.Open` + `Ping` в одном вызове.

***

### `StructScan` — сканирование в структуру

Главная боль `database/sql` — перечислять поля вручную. `sqlx` делает это по тегам `db`:

```go
type User struct {
    ID    int    `db:"id"`
    Name  string `db:"name"`
    Email string `db:"email"`
}

var users []User
err := db.Select(&users, "SELECT id, name, email FROM users WHERE active = $1", true)
```

Методы для сканирования:

| Метод | Когда |
|---|---|
| `db.Select(&slice, query, args...)` | Много строк → срез структур |
| `db.Get(&struct, query, args...)` | Одна строка → одна структура. Нет строк → ошибка `sql.ErrNoRows` |
| `db.QueryRowx(query, args...).StructScan(&struct)` | Как `Get`, но больше контроля |

Сложная структура — вложенные объекты из JOIN:

```go
type OrderWithUser struct {
    OrderID  int     `db:"order_id"`
    Amount   float64 `db:"amount"`
    UserName string  `db:"user_name"`
}

var orders []OrderWithUser
db.Select(&orders, `
    SELECT o.id AS order_id, o.amount, u.name AS user_name
    FROM orders o
    JOIN users u ON u.id = o.user_id
`)
```

Ключевой момент: алиасы в SQL (`AS order_id`) должны совпадать с тегами `db` в структуре.

***

### `Named` — именованные параметры

Вместо `$1, $2, $3` — параметры по именам из структуры или мапы:

```go
user := User{Name: "Alice", Email: "alice@example.com"}

// :name и :email подставятся из полей структуры
_, err := db.NamedExec(`
    INSERT INTO users (name, email) VALUES (:name, :email)
`, user)
```

С мапой:

```go
db.NamedExec(`UPDATE users SET email = :email WHERE id = :id`,
    map[string]interface{}{
        "id":    42,
        "email": "new@example.com",
    })
```

`NamedQuery` для чтения:

```go
rows, _ := db.NamedQuery(`SELECT * FROM users WHERE name = :name`, user)
defer rows.Close()
```

***

### `In` — `IN (...)` со срезами

`database/sql` не умеет подставлять срез в `IN ($1, $2, $3)`. `sqlx` — умеет:

```go
ids := []int{1, 2, 3, 4, 5}

// sqlx.In раскроет ? в ?, ?, ?, ?, ? и переставит аргументы
query, args, _ := sqlx.In("SELECT * FROM users WHERE id IN (?)", ids)

// query = "SELECT * FROM users WHERE id IN (?, ?, ?, ?, ?)"
// args  = [1, 2, 3, 4, 5]
db.Query(query, args...)
```

С PostgreSQL — плейсхолдеры `$1, $2` вместо `?`. `sqlx.In` всегда выдаёт `?`, но `db.Rebind()` преобразует их под нужный драйвер:

```go
ids := []int{1, 2, 3}

query, args, _ := sqlx.In("SELECT * FROM users WHERE id IN (?)", ids)

// query = "SELECT * FROM users WHERE id IN (?)"  ← пока ?
// args  = [1, 2, 3]

query = db.Rebind(query)

// query = "SELECT * FROM users WHERE id IN ($1, $2, $3)"  ← $1, $2, $3
// args  = [1, 2, 3]

db.Query(query, args...)
```

`Rebind` знает, с каким драйвером работает `*sqlx.DB`, и подставляет правильный синтаксис: `$1, $2` для PostgreSQL, `?` для MySQL. Вызывать `Rebind` нужно всегда после `sqlx.In`, если БД использует `$`-плейсхолдеры.

### `sqlx` vs `database/sql`

`sqlx` не скрывает `database/sql`. Пул соединений, транзакции, `Prepare` — всё на месте. Разница только в удобстве:

| Операция | `database/sql` | `sqlx` |
|---|---|---|
| SELECT много строк | `rows.Next()` + `rows.Scan(&a, &b, &c)` | `db.Select(&slice, ...)` |
| SELECT одна строка | `db.QueryRow(...).Scan(&a, &b, &c)` | `db.Get(&struct, ...)` |
| INSERT с параметрами из структуры | Перечислять поля вручную | `db.NamedExec(...)` |
| `IN (?, ?, ?)` | Строить вручную | `sqlx.In(...)` |

### `sqlx.Tx` — транзакции

```go
tx, _ := db.Beginx()          // *sqlx.Tx вместо *sql.Tx
defer tx.Rollback()

tx.NamedExec(`INSERT INTO users (name, email) VALUES (:name, :email)`, user)
tx.Select(&users, "SELECT * FROM users WHERE active = $1", true)

tx.Commit()
```

Те же `Select`, `Get`, `NamedExec` — но в транзакции.

> **Зачем это Go-разработчику.** `sqlx` — золотая середина между `database/sql` и ORM. Не прячет SQL, но убирает `Scan(&a, &b, &c, &d, &e)`. `Select` и `Get` сокращают рутину в 3 раза. `sqlx.In` решает проблему `IN`-клаузы, которая в чистом `database/sql` требует ручной сборки запроса. Если проект уже на `database/sql` — перейти на `sqlx` можно за час: методы совместимы.

***

## 12. `pgx`: нативный драйвер PostgreSQL

[`pgx`](https://github.com/jackc/pgx) — это не просто драйвер, а полноценный клиент PostgreSQL на чистом Go. В отличие от `database/sql` + `pgx/stdlib`, нативный `pgx` даёт прямой доступ к PostgreSQL-специфичным возможностям: `COPY`, `LISTEN`/`NOTIFY`, массивы, JSONB, уведомления.

`pgx` работает в двух режимах:

| Режим | Пакет | Совместимость |
|---|---|---|
| Нативный | `github.com/jackc/pgx/v5` | Только PostgreSQL. Быстрее, больше возможностей |
| Через `database/sql` | `github.com/jackc/pgx/v5/stdlib` | Совместим с `database/sql`, `sqlx`, ORM |

До этого раздела мы использовали `pgx/stdlib` через `database/sql`. Здесь — нативный режим.

***

### Подключение: `pgxpool`

Нативный `pgx` использует `pgxpool` — собственный пул соединений, оптимизированный под PostgreSQL:

```go
import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
)

ctx := context.Background()

pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost:5432/mydb?sslmode=disable")
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

if err := pool.Ping(ctx); err != nil {
    log.Fatal(err)
}
```

`pgxpool.New` = `sql.Open` + `Ping` + настройка пула. Пул уже готов к работе.

### Запросы: `Query`, `QueryRow`, `Exec`

Интерфейс похож на `database/sql`, но с контекстом первым аргументом:

```go
// Множество строк
rows, _ := pool.Query(ctx, "SELECT id, name FROM users WHERE active = $1", true)
defer rows.Close()

for rows.Next() {
    var id int
    var name string
    rows.Scan(&id, &name)
}

// Одна строка
var name string
pool.QueryRow(ctx, "SELECT name FROM users WHERE id = $1", 42).Scan(&name)

// Изменение данных
tag, _ := pool.Exec(ctx, "UPDATE users SET name = $1 WHERE id = $2", "Alice", 1)
fmt.Println(tag.RowsAffected()) // количество затронутых строк
```

### `CollectRows` — сбор строк в срез

Вместо ручного цикла `rows.Next()` + `Scan` — `pgx.CollectRows` собирает результат напрямую в срез:

```go
rows, _ := pool.Query(ctx, "SELECT id, name FROM users")

users, err := pgx.CollectRows(rows, pgx.RowToStructByName[User])
// users — []User, заполненный автоматически
```

`RowToStructByName` находит поля структуры по именам столбцов (без тегов `db`, просто `Name string` → колонка `name`). Для JOIN с алиасами:

```go
type OrderWithUser struct {
    OrderID int    // → колонка order_id (RowToStructByName)
    Amount  float64
    Name    string // → колонка name
}
```

### PostgreSQL-специфичные типы

`pgx` поддерживает типы, которых нет в `database/sql`:

```go
// Массивы
var tags []string
pool.QueryRow(ctx, "SELECT tags FROM users WHERE id = $1", 1).Scan(&tags)
// tags = ["go", "postgresql", "backend"]

// JSONB
var metadata map[string]interface{}
pool.QueryRow(ctx, "SELECT metadata FROM users WHERE id = $1", 1).Scan(&metadata)

// UUID
var id pgtype.UUID // или google/uuid
pool.QueryRow(ctx, "SELECT id FROM users").Scan(&id)

// INET (IP-адреса)
var ip net.IP
pool.QueryRow(ctx, "SELECT ip_address FROM sessions").Scan(&ip)
```

Стандартные `database/sql` типы (`sql.NullString`) работают, но pgx-типы (`pgtype.Text`, `pgtype.Int4`) эффективнее — без лишних аллокаций.

### `COPY` — массовая вставка

`COPY` — протокол PostgreSQL для сверхбыстрой пакетной загрузки. Быстрее `INSERT` в 5–10 раз:

```go
rows := [][]interface{}{
    {"Alice", "alice@example.com"},
    {"Bob", "bob@example.com"},
    {"Charlie", "charlie@example.com"},
}

_, err := pool.CopyFrom(
    ctx,
    pgx.Identifier{"users"},
    []string{"name", "email"},
    pgx.CopyFromRows(rows),
)
```

`CopyFromRows` принимает срез строк — каждая строка это `[]interface{}`. Для тысяч записей `COPY` — единственный правильный способ.

### `LISTEN`/`NOTIFY` — асинхронные уведомления

PostgreSQL умеет отправлять уведомления клиентам. `pgx` может их принимать:

```go
conn, _ := pool.Acquire(ctx)
defer conn.Release()

conn.Exec(ctx, "LISTEN user_created")

for {
    notification, _ := conn.Conn().WaitForNotification(ctx)
    fmt.Printf("Канал: %s, данные: %s\n",
        notification.Channel, notification.Payload)
}
```

Другое приложение делает `NOTIFY user_created, 'user_id=42'` — и ваш код получает уведомление в реальном времени. Полезно для инвалидации кеша между сервисами.

### `pgx` нативный vs `database/sql`

| Возможность | `database/sql` + `pgx/stdlib` | Нативный `pgx` |
|---|---|---|
| Базовые запросы | ✅ | ✅ |
| `COPY` | ❌ | ✅ |
| PostgreSQL-типы (UUID, INET, JSONB) | Частично, через string | ✅ нативно |
| `LISTEN`/`NOTIFY` | ❌ | ✅ |
| `CollectRows` | ❌ (используйте `sqlx`) | ✅ |
| Производительность | Базовая + оверхед `database/sql` | Быстрее, меньше аллокаций |
| Совместимость с ORM/`sqlx` | ✅ | ❌ |

> Правило: если проект только на PostgreSQL — берите нативный `pgx`. Если нужна совместимость с `sqlx`, GORM или потенциальная поддержка других БД — `pgx/stdlib` через `database/sql`.

> **Зачем это Go-разработчику.** `pgx` — стандарт de facto для работы с PostgreSQL в Go. `COPY` даёт 5–10x прирост на массовых вставках. `CollectRows` заменяет `sqlx.Select`. `LISTEN`/`NOTIFY` позволяет строить реактивные системы без Kafka для простых сценариев. Если вы точно на PostgreSQL — нет причин использовать `database/sql` вместо нативного `pgx`.

***

## 13. ORM на примере GORM

ORM (Object-Relational Mapping) — прослойка, которая отображает строки таблиц на Go-структуры, а SQL-запросы — на вызовы методов. [GORM](https://gorm.io) — самая популярная ORM для Go.

Главное обещание ORM: «пиши на Go, SQL сгенерируется сам». Реальность: для простых CRUD-операций это правда, для сложных запросов — ORM мешает.

***

### Модели и `AutoMigrate`

Модель — структура с GORM-тегами. `AutoMigrate` создаёт/обновляет таблицы автоматически:

```go
type User struct {
    ID        uint   `gorm:"primaryKey"`
    Name      string `gorm:"not null"`
    Email     string `gorm:"uniqueIndex;not null"`
    Orders    []Order `gorm:"foreignKey:UserID"`
    CreatedAt time.Time
}

type Order struct {
    ID     uint `gorm:"primaryKey"`
    UserID uint
    Amount float64
}

db.AutoMigrate(&User{}, &Order{})
```

`AutoMigrate` НЕ удаляет колонки и не меняет типы — только добавляет новые колонки и индексы. Для production-миграций используйте `golang-migrate`, а `AutoMigrate` — для разработки и тестов.

### CRUD: Create, Read, Update, Delete

```go
// Create
user := User{Name: "Alice", Email: "alice@example.com"}
db.Create(&user) // user.ID заполнен после вставки

// Read — одна запись
var user User
db.First(&user, 1)        // по первичному ключу
db.First(&user, "email = ?", "alice@example.com") // по условию

// Read — множество
var users []User
db.Where("name LIKE ?", "A%").Order("created_at desc").Limit(10).Find(&users)

// Update
db.Model(&user).Update("name", "Alice Johnson")
db.Model(&user).Updates(User{Name: "Alice J.", Email: "alice@new.com"})

// Delete
db.Delete(&user, 1)
```

### Ассоциации

GORM автоматически подгружает связанные записи через `Preload`:

```go
type User struct {
    ID     uint
    Name   string
    Orders []Order `gorm:"foreignKey:UserID"`
}

type Order struct {
    ID     uint
    UserID uint
    Amount float64
}

var user User
db.Preload("Orders").First(&user, 1)
// user.Orders заполнен: все заказы пользователя
```

Типы ассоциаций:

| Отношение | Теги |
|---|---|
| Has Many | `User.Orders []Order gorm:"foreignKey:UserID"` |
| Belongs To | `Order.User User gorm:"foreignKey:UserID"` |
| Has One | `User.Profile Profile gorm:"foreignKey:UserID"` |
| Many To Many | `User.Languages []Language gorm:"many2many:user_languages"` |

### Preload vs N+1

**N+1** — антипаттерн: сначала грузите пользователей, потом для каждого идёте в БД за заказами.

```go
// ПЛОХО: 1 запрос за пользователями + N запросов за заказами каждого
var users []User
db.Find(&users) // SELECT * FROM users

for i := range users {
    db.Model(&users[i]).Association("Orders").Find(&users[i].Orders)
    // SELECT * FROM orders WHERE user_id = ?
}
```

10 пользователей = 1 + 10 = 11 запросов. 1000 пользователей = 1001 запрос.

```go
// ХОРОШО: 2 запроса (JOIN или два SELECT с IN)
var users []User
db.Preload("Orders").Find(&users)
// 1. SELECT * FROM users
// 2. SELECT * FROM orders WHERE user_id IN (1, 2, 3, ..., 1000)
```

`Preload` собирает все ID пользователей и делает ОДИН запрос за всеми заказами.

> Всегда используйте `Preload` для ассоциаций. Без него связанные поля остаются пустыми до явного обращения — что и приводит к N+1.

### Транзакции в GORM

```go
db.Transaction(func(tx *gorm.DB) error {
    if err := tx.Create(&user).Error; err != nil {
        return err // автоматический Rollback
    }

    order := Order{UserID: user.ID, Amount: 100}
    if err := tx.Create(&order).Error; err != nil {
        return err // автоматический Rollback
    }

    return nil // Commit
})
```

Возврат ошибки из замыкания → автоматический `Rollback`. `return nil` → `Commit`. Удобно, но скрывает логику транзакции за замыканием.

### Когда ORM полезен, а когда мешает

**Полезен:**

- Простые CRUD-операции: Create, Find, Update, Delete без сложных JOIN
- Прототипирование и MVP — скорость разработки выше
- Команда боится SQL или нет выделенного DBA

**Мешает:**

- Сложные запросы с JOIN, подзапросами, оконными функциями
- Оптимизация — ORM генерирует неоптимальный SQL, а доступа к плану запроса нет
- Большие проекты — накопленные костыли вокруг ORM перевешивают начальную выгоду
- `AutoMigrate` в production — риск потерять данные

### GORM vs sqlx vs database/sql

| Подход | Когда |
|---|---|
| `database/sql` | Максимальный контроль. Микросервисы с простыми запросами |
| `sqlx` | Золотая середина. SQL пишете вы, маппинг — библиотека |
| GORM | Прототипы, CRUD-heavy приложения без сложной аналитики |

> **Зачем это Go-разработчику.** GORM экономит время на старте, но за сложность запросов приходится платить дважды: сначала обходом ограничений ORM, потом переписыванием на сырой SQL. Если пишете production-сервис на годы — начните с `sqlx` или `pgx`. Если прототип на неделю — GORM. И никогда не полагайтесь на `AutoMigrate` в production.

***

## 14. Паттерн репозиторий

Репозиторий — прослойка между бизнес-логикой и базой данных. Вместо того чтобы вызывать `db.Query` прямо из обработчика HTTP, вы прячете SQL за интерфейсом.

Зачем:
- **Тестируемость** — подменяете реальный репозиторий моком, никакой базы для тестов не нужно
- **Изоляция** — бизнес-логика не знает, откуда берутся данные: PostgreSQL, кеш, внешний API
- **Переиспользование** — один метод `GetUserByID` вместо разбросанных `db.QueryRow` по всему коду

***

### Интерфейс

```go
type User struct {
    ID    int
    Name  string
    Email string
}

type UserRepository interface {
    GetByID(ctx context.Context, id int) (*User, error)
    Create(ctx context.Context, user *User) error
    List(ctx context.Context, limit, offset int) ([]User, error)
}
```

Интерфейс описывает ЧТО делает репозиторий. Реализация — КАК.

### Реализация на `sqlx`

```go
type userRepository struct {
    db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) UserRepository {
    return &userRepository{db: db}
}

func (r *userRepository) GetByID(ctx context.Context, id int) (*User, error) {
    var user User
    err := r.db.GetContext(ctx, &user,
        "SELECT id, name, email FROM users WHERE id = $1", id)
    if err == sql.ErrNoRows {
        return nil, nil // нет пользователя — не ошибка
    }
    return &user, err
}

func (r *userRepository) Create(ctx context.Context, user *User) error {
    return r.db.QueryRowContext(ctx,
        "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id",
        user.Name, user.Email,
    ).Scan(&user.ID)
}

func (r *userRepository) List(ctx context.Context, limit, offset int) ([]User, error) {
    var users []User
    err := r.db.SelectContext(ctx, &users,
        "SELECT id, name, email FROM users ORDER BY id LIMIT $1 OFFSET $2",
        limit, offset)
    return users, err
}
```

Ключевые решения:
- Неэкспортируемая структура `userRepository` — снаружи только интерфейс
- Конструктор `NewUserRepository` возвращает интерфейс, а не структуру
- `*Context`-методы `sqlx` — поддержка таймаутов и отмены через контекст
- `ErrNoRows` → `nil, nil` — отсутствие записи не ошибка

### Внедрение зависимостей

```go
type UserService struct {
    repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
    return &UserService{repo: repo}
}

func (s *UserService) GetUser(ctx context.Context, id int) (*User, error) {
    user, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("get user: %w", err)
    }
    if user == nil {
        return nil, ErrNotFound
    }
    return user, nil
}
```

Сервис зависит от интерфейса, а не от конкретной реализации. В production передаётся `NewUserRepository(db)`, в тестах — мок.

### Тестирование с моком

```go
type mockUserRepository struct {
    users map[int]*User
}

func (m *mockUserRepository) GetByID(ctx context.Context, id int) (*User, error) {
    user, ok := m.users[id]
    if !ok {
        return nil, nil
    }
    return user, nil
}

func (m *mockUserRepository) Create(ctx context.Context, user *User) error {
    user.ID = len(m.users) + 1
    m.users[user.ID] = user
    return nil
}

func (m *mockUserRepository) List(ctx context.Context, limit, offset int) ([]User, error) {
    // ... реализация
}

func TestUserService_GetUser(t *testing.T) {
    repo := &mockUserRepository{
        users: map[int]*User{
            1: {ID: 1, Name: "Alice", Email: "alice@example.com"},
        },
    }
    svc := NewUserService(repo)

    user, err := svc.GetUser(context.Background(), 1)
    assert.NoError(t, err)
    assert.Equal(t, "Alice", user.Name)
}
```

Репозиторий-мок — обычная мапа. Никакой БД, никаких контейнеров. Тест проходит за микросекунды.

### Когда использовать

**Используйте репозиторий:**
- Сервис содержит бизнес-логику, которую нужно тестировать изолированно
- Один и тот же запрос вызывается из нескольких мест
- Вы меняли БД (MySQL → PostgreSQL) и хотите заменить только слой доступа

**Не используйте репозиторий:**
- Сервис — тонкая обёртка над одним SQL-запросом (YAGNI)
- Прототип из 3 эндпоинтов — интерфейс + мок займут больше кода, чем бизнес-логика

### Репозиторий + транзакции

Транзакции проходят сквозь репозиторий через передачу `*sql.Tx`:

```go
type UserRepository interface {
    GetByIDTx(ctx context.Context, tx *sqlx.Tx, id int) (*User, error)
    CreateTx(ctx context.Context, tx *sqlx.Tx, user *User) error
}

func (r *userRepository) CreateTx(ctx context.Context, tx *sqlx.Tx, user *User) error {
    return tx.QueryRowContext(ctx,
        "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id",
        user.Name, user.Email,
    ).Scan(&user.ID)
}

// В сервисе:
func (s *UserService) TransferMoney(ctx context.Context, fromID, toID int, amount float64) error {
    tx, _ := s.db.Beginx()
    defer tx.Rollback()

    // ... операции через s.userRepo.CreateTx(ctx, tx, ...) и s.accountRepo.UpdateBalanceTx(ctx, tx, ...)

    return tx.Commit()
}
```

Альтернатива — передавать `*sqlx.Tx` через `context.Context`, но явная передача читается лучше и безопаснее компилятором.

> **Зачем это Go-разработчику.** Репозиторий — паттерн номер один для тестируемого кода с БД. Без него тест бизнес-логики требует поднятия PostgreSQL в Docker. С ним — мок на мапе, тест за микросекунды. Интерфейс → мок — фундаментальный приём в Go, и репозиторий его идеально иллюстрирует.

***

## 15. SQL-инъекции и безопасность

SQL-инъекция — уязвимость номер один для приложений с БД. Злоумышленник передаёт SQL-код в строковом параметре, и приложение выполняет его. Один неосторожный `fmt.Sprintf` — и данные утекли, таблицы удалены, сервер скомпрометирован.

Механизм защиты раскрыт в разделе 8 (подготовленные выражения). Здесь — полная картина угроз и способов защиты.

***

### Как работает инъекция

Уязвимый код — подстановка пользовательского ввода напрямую в SQL:

```go
// УЯЗВИМО
userInput := r.URL.Query().Get("id")
query := fmt.Sprintf("SELECT * FROM users WHERE id = %s", userInput)
db.Query(query)
```

| Вход | Что выполнит БД |
|---|---|
| `42` | `SELECT * FROM users WHERE id = 42` — норма |
| `1 OR 1=1` | `SELECT * FROM users WHERE id = 1 OR 1=1` — **все строки** |
| `1; DROP TABLE users; --` | Два запроса: SELECT и **DROP TABLE** |
| `1 UNION SELECT email, password FROM admins` | UNION-инъекция: утечка паролей |

Безопасная альтернатива — плейсхолдеры:

```go
db.Query("SELECT * FROM users WHERE id = $1", userInput)
// userInput = "1; DROP TABLE users;" → БД ищет id = '1; DROP TABLE users;'
// Это просто строка, не SQL-команда
```

Плейсхолдер отделяет SQL-код от данных. Параметр **никогда** не интерпретируется как часть запроса.

### Типы инъекций

| Тип | Пример | Защита |
|---|---|---|
| **Classic SQLi** | `' OR '1'='1` в поле входа | Плейсхолдеры |
| **UNION-based** | `1 UNION SELECT ...` — кража данных из других таблиц | Плейсхолдеры |
| **Blind SQLi** | `1 AND (SELECT length(password) FROM users LIMIT 1) > 10` — угадывание по ответам true/false | Плейсхолдеры |
| **Second-order** | Злоумышленник сохраняет `'; DROP TABLE--` в поле имени, которое позже подставляется в другой запрос | Плейсхолдеры везде, даже при чтении собственных данных |
| **Имена таблиц/колонок** | `?orderBy=name; DROP TABLE--` → `ORDER BY name; DROP TABLE--` | Белые списки значений |

### Инъекции через имена таблиц и колонок

Плейсхолдеры защищают **значения**, но не имена таблиц/колонок. Для сортировки и фильтрации по имени колонки — белый список:

```go
var allowedColumns = map[string]bool{
    "id":   true,
    "name": true,
    "created_at": true,
}

func listUsers(db *sql.DB, orderBy string) ([]User, error) {
    if !allowedColumns[orderBy] {
        return nil, fmt.Errorf("invalid column: %s", orderBy)
    }
    // Безопасно: orderBy проверен по белому списку
    query := fmt.Sprintf("SELECT * FROM users ORDER BY %s", orderBy)
    return db.Query(query)
}
```

> Никогда не подставляйте пользовательский ввод в SQL, даже косвенно. Белый список — единственный безопасный способ для имён столбцов.

### Что ещё, кроме плейсхолдеров

Плейсхолдеры — основа, но не всё:

1. **Никаких секретов в логах.** Не логируйте SQL с подставленными параметрами — там могут быть пароли, токены, персональные данные.
2. **Минимальные привилегии.** Приложение должно подключаться к БД под пользователем с минимумом прав: `SELECT`, `INSERT`, `UPDATE`, `DELETE` на свои таблицы. Не `SUPERUSER`, не владелец базы.
3. **SSL/TLS между приложением и БД.** `sslmode=verify-full` в production (раздел 5). Без него SQL-запросы идут открытым текстом.
4. **Валидация на входе.** Длина строки, формат email, диапазон чисел — до того как данные попадут в SQL.
5. **`database/sql` экранирует.** Не изобретайте своё экранирование — используйте плейсхолдеры, они делают это правильно для каждого драйвера.

### Инъекция через LIKE

`LIKE` имеет свои спецсимволы: `%` (любая подстрока) и `_` (один символ). Если пользователь вводит `%`, поиск возвращает всё:

```go
// Опасность: поиск по '%%' вернёт все строки
db.Query("SELECT * FROM users WHERE name LIKE $1", "%"+userInput+"%")

// Безопасно: экранировать спецсимволы LIKE
escaped := strings.ReplaceAll(userInput, "%", "\\%")
escaped = strings.ReplaceAll(escaped, "_", "\\_")
db.Query("SELECT * FROM users WHERE name LIKE $1 ESCAPE '\\'", "%"+escaped+"%")
```

### Проверка в CI/CD

Автоматизируйте поиск уязвимостей:
- [`gosec`](https://github.com/securego/gosec) — находит `fmt.Sprintf` c SQL в коде
- [`sqlint`](https://github.com/srcclr/sqlint) — проверяет SQL-строки
- Правило в линтере: запретить `fmt.Sprintf` в комбинации с `Query`/`Exec`

> **Зачем это Go-разработчику.** SQL-инъекция — не теория из учебника. OWASP ставит её на 3-е место среди всех веб-уязвимостей. Одна забытая конкатенация строк — и база данных скомпрометирована. Плейсхолдеры решают 95% проблем, белые списки — оставшиеся 5% (имена колонок). Автоматизируйте проверки в CI: `gosec` найдёт `fmt.Sprintf` в SQL-контексте до того, как код попадёт в production.

***

## 16. Производительность

Медленные запросы убивают приложение быстрее, чем отсутствие фич. Три кита производительности БД: индексы, анализ плана запроса и мониторинг.

***

### Индексы

Индекс — структура данных, которая позволяет БД находить строки без полного перебора таблицы. Как алфавитный указатель в книге: вместо пролистывания всех страниц — открыл указатель, увидел номер страницы, перешёл.

```sql
-- Без индекса: последовательное сканирование (Seq Scan) всей таблицы
-- SELECT * FROM users WHERE email = 'alice@example.com' -- миллионы строк → минуты

-- С индексом: поиск по B-дереву
CREATE INDEX idx_users_email ON users(email);

-- SELECT * FROM users WHERE email = 'alice@example.com' -- миллионы строк → миллисекунды
```

**Типы индексов в PostgreSQL:**

| Тип | Для чего | Пример |
|---|---|---|
| B-tree (по умолчанию) | Равенство, диапазоны, сортировка | `WHERE id = 5`, `WHERE created_at > '...'` |
| GIN | Полнотекстовый поиск, массивы, JSONB | `WHERE tags @> ARRAY['go']` |
| GiST | Геометрия, полнотекстовый поиск | `WHERE geom && 'POINT(...)'` |
| BRIN | Очень большие таблицы с коррелированными данными | `WHERE created_at BETWEEN ...` |
| Hash | Только равенство (редко используется) | `WHERE email = '...'` |

**Когда индекс НЕ работает:**

- `WHERE LOWER(email) = '...'` — функция на колонке отключает индекс. Решение: индекс на выражение `CREATE INDEX idx ON users(LOWER(email))`
- `WHERE name LIKE '%Alice'` — поиск по подстроке с начала строки. B-tree работает только с префиксом: `LIKE 'Alice%'`
- Маленькие таблицы (<1000 строк) — Seq Scan быстрее индекса

**Составные индексы** для запросов с несколькими условиями:

```sql
-- Запрос: WHERE user_id = ? AND status = ? ORDER BY created_at
CREATE INDEX idx_orders_user_status_created ON orders(user_id, status, created_at);
```

Порядок колонок важен: сначала условия равенства (`user_id`, `status`), потом сортировка (`created_at`).

### `EXPLAIN ANALYZE` — план запроса

Показывает, КАК БД выполняет запрос и сколько это стоит:

```sql
EXPLAIN ANALYZE
SELECT u.name, COUNT(o.id) AS order_count
FROM users u
LEFT JOIN orders o ON o.user_id = u.id
WHERE u.active = true
GROUP BY u.name;
```

```
 HashAggregate  (cost=150.30..152.42 rows=200 width=36) (actual time=3.221..3.440 rows=180 loops=1)
   ->  Hash Left Join  (cost=25.15..125.55 rows=4950 width=32) (actual time=0.123..2.891 rows=4850 loops=1)
         ->  Seq Scan on users u  (cost=0.00..18.50 rows=200 width=10) (actual time=0.012..0.045 rows=200 loops=1)
               Filter: active
         ->  Hash  (cost=18.00..18.00 rows=400 width=14) (actual time=0.089..0.089 rows=400 loops=1)
               ->  Seq Scan on orders o  (cost=0.00..18.00 rows=400 width=14) (actual time=0.010..0.045 rows=400 loops=1)
```

Что смотреть:
- **Seq Scan** (последовательное сканирование) на большой таблице — плохо, нужен индекс
- **cost=0.00..18.50** — оценка стоимости. Первое число — старт, второе — общая. Чем меньше, тем лучше
- **actual time** — реальное время. Если сильно отличается от cost — статистика устарела, сделайте `ANALYZE`
- **rows** — оценка vs реальность. Большое расхождение → неверный план

> Перед добавлением индекса всегда проверяйте `EXPLAIN ANALYZE`. Не угадывайте — измеряйте.

### N+1: проблема и решение

N+1 — когда вместо одного запроса с JOIN выполняется 1 + N запросов. Детально разобрано в разделе 13 (GORM/Preload), здесь — общий случай.

```go
// N+1: 1 запрос за пользователями + N запросов за заказами
rows, _ := db.Query("SELECT id, name FROM users")
var users []User
for rows.Next() {
    var u User
    rows.Scan(&u.ID, &u.Name)
    
    // Для каждого пользователя — отдельный запрос в БД
    db.Query("SELECT * FROM orders WHERE user_id = $1", u.ID)
    users = append(users, u)
}
```

Решение — один запрос с JOIN или два запроса с `WHERE IN`:

```sql
-- Вариант 1: JOIN (возвращает денормализованные данные)
SELECT u.id, u.name, o.id AS order_id, o.amount
FROM users u
LEFT JOIN orders o ON o.user_id = u.id

-- Вариант 2: два запроса (данные собираются в коде)
SELECT id, name FROM users
SELECT id, user_id, amount FROM orders WHERE user_id = ANY($1)
```

### `pg_stat_statements` — мониторинг запросов

Расширение PostgreSQL, которое собирает статистику по всем запросам:

```sql
CREATE EXTENSION pg_stat_statements;

-- Самые медленные запросы (среднее время)
SELECT query, calls, mean_exec_time, total_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;

-- Самые частые запросы
SELECT query, calls, total_exec_time
FROM pg_stat_statements
ORDER BY calls DESC
LIMIT 10;
```

Интегрируйте в мониторинг: Grafana + Prometheus + `pg_stat_statements` дают полную картину по медленным запросам.

### Пул соединений и производительность

Настройки пула из раздела 6 напрямую влияют на производительность:

| Симптом | Причина | Решение |
|---|---|---|
| Растёт `WaitCount` в `db.Stats()` | `MaxOpenConns` слишком мал | Увеличить `MaxOpenConns` |
| `WaitDuration` > 100ms | Запросы долгие или пул мал | Оптимизация запросов + увеличить пул |
| Много idle-соединений | `MaxIdleConns` завышен | Снизить до 10–25 |
| Ошибки таймаута соединения | `ConnMaxLifetime` не настроен | Выставить 30–60m |

### Быстрые проверки

1. **Медленный запрос** → `EXPLAIN ANALYZE` → есть Seq Scan на большой таблице? → индекс
2. **Растёт нагрузка** → `db.Stats()` → `WaitCount` растёт? → увеличить пул или кешировать
3. **N+1 в логах** → много одинаковых запросов подряд → JOIN или batch-загрузка
4. **Индекс не используется** → функция на колонке или не тот тип индекса → индекс на выражение

> **Зачем это Go-разработчику.** Индексы и `EXPLAIN ANALYZE` — не «забота DBA». В стартапе DBA нет — есть вы. Медленный запрос в production обнаруживается не глазами, а через `pg_stat_statements` и алерты. N+1 — самая частая причина деградации под нагрузкой: на 10 пользователях незаметно, на 10000 — отказ. Индекс на поле, по которому ищете, — первое, что нужно сделать перед выкаткой фичи.

***

## 17. Тестирование с базой данных

Для unit-тестов бизнес-логики используйте моки репозиториев (раздел 14). Но иногда нужны тесты с НАСТОЯЩЕЙ базой: миграции, сложные SQL-запросы, триггеры, блокировки. Два основных подхода: testcontainers и транзакционные тесты.

***

### `testcontainers-go` — база в Docker

[testcontainers-go](https://golang.testcontainers.org) поднимает контейнер с PostgreSQL прямо в тесте. Никакой внешней БД, никаких «запусти docker-compose перед тестами»:

```go
import (
    "context"
    "testing"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestWithRealDB(t *testing.T) {
    ctx := context.Background()

    // Поднять PostgreSQL 16
    pg, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    if err != nil {
        t.Fatal(err)
    }
    defer pg.Terminate(ctx) // контейнер удалится после теста

    // Получить connection string
    connStr, _ := pg.ConnectionString(ctx)
    // connStr = "postgres://test:test@localhost:54321/testdb?sslmode=disable"

    db, _ := sql.Open("pgx", connStr)
    defer db.Close()

    // Применить миграции
    runMigrations(db)

    // Тестировать как с реальной БД
    _, err = db.Exec("INSERT INTO users (name, email) VALUES ($1, $2)", "Alice", "alice@example.com")
    if err != nil {
        t.Fatal(err)
    }
}
```

Плюсы:
- **Настоящая БД** — точь-в-точь как в production, включая особенности диалекта
- **Изоляция** — каждый тест (или пакет) получает чистую базу
- **CI/CD** — работает везде, где есть Docker

Минусы:
- **Медленно** — запуск контейнера 2–5 секунд. Для сотен тестов — минуты
- **Docker в CI** — нужен настроенный Docker-раннер

### Транзакционные тесты

Альтернатива для скорости: один раз поднять БД на весь пакет, каждый тест — в отдельной транзакции с откатом:

```go
var testDB *sql.DB

func TestMain(m *testing.M) {
    // Один раз подключиться к тестовой БД
    var err error
    testDB, err = sql.Open("pgx", "postgres://test:test@localhost:5432/testdb?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer testDB.Close()

    runMigrations(testDB) // применить миграции один раз
    os.Exit(m.Run())
}

func TestCreateUser(t *testing.T) {
    tx, err := testDB.Begin()
    if err != nil {
        t.Fatal(err)
    }
    defer tx.Rollback() // ОТКАТ в конце — БД чиста для следующего теста

    // Тест в транзакции
    _, err = tx.Exec("INSERT INTO users (name, email) VALUES ($1, $2)", "Alice", "alice@example.com")
    if err != nil {
        t.Fatal(err)
    }

    var name string
    tx.QueryRow("SELECT name FROM users WHERE email = $1", "alice@example.com").Scan(&name)
    if name != "Alice" {
        t.Errorf("expected Alice, got %s", name)
    }

    // Rollback при выходе — Алиса не останется в БД
}
```

Ключевые моменты:
- `TestMain` — один раз на весь пакет: подключиться, прогнать миграции
- Каждый тест: `Begin()` → `defer Rollback()` → тест → автоматический откат
- После теста БД чиста — другие тесты не видят чужих данных

> Будьте осторожны: код, который делает `Commit` внутри тестируемой функции, сломает изоляцию. Используйте транзакционные тесты только для кода, который не управляет транзакциями сам.

### Фикстуры — наполнение данными

Перед тестом нужно заполнить БД известными данными. Два способа:

**SQL-фикстуры** — файл с INSERT'ами:

```sql
-- fixtures/users.sql
INSERT INTO users (id, name, email) VALUES
    (1, 'Alice', 'alice@example.com'),
    (2, 'Bob',   'bob@example.com');
```

```go
func loadFixture(tx *sql.Tx, path string) {
    sql, _ := os.ReadFile(path)
    tx.Exec(string(sql))
}
```

**Хелперы** — Go-функции для создания тестовых данных:

```go
func createTestUser(tx *sql.Tx, name string) *User {
    var id int
    tx.QueryRow("INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id",
        name, name+"@test.com").Scan(&id)
    return &User{ID: id, Name: name, Email: name + "@test.com"}
}
```

### Тестирование миграций

Миграции нужно тестировать отдельно — это самый опасный код в проекте:

```go
func TestMigrations(t *testing.T) {
    ctx := context.Background()
    pg, _ := postgres.RunContainer(ctx, testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("testdb"), postgres.WithUsername("test"), postgres.WithPassword("test"))
    defer pg.Terminate(ctx)

    connStr, _ := pg.ConnectionString(ctx)

    // Применить все миграции
    m, _ := migrate.New("file://migrations", connStr)
    m.Up()

    // Откатить последнюю
    m.Steps(-1)

    // Применить снова (проверяем, что down реально откатил)
    m.Up()
}
```

### Мок vs реальная БД

| Подход | Когда |
|---|---|
| **Мок репозитория** (раздел 14) | Тест бизнес-логики. Быстро, детерминировано |
| **Транзакционный тест** | Тест SQL-запроса, миграции. Быстрее testcontainers |
| **testcontainers** | Полный интеграционный тест. Особенности диалекта, триггеры, расширения |

Правило: unit-тесты → мок. SQL-тесты → транзакции. Интеграционные тесты → testcontainers. Не используйте testcontainers там, где достаточно мока.

> **Зачем это Go-разработчику.** Тестирование с БД — не опция, а требование. Миграция с опечаткой в имени колонки не упадёт при компиляции — только в рантайме. Транзакционные тесты дают баланс скорости и достоверности: одна БД на пакет, откат после каждого теста. Testcontainers — золотой стандарт для интеграционных тестов: «в CI точно так же, как в production». Начните с транзакционных тестов для SQL, добавьте testcontainers для миграций и сложных сценариев.

***

## 18. SQLite — встраиваемая база данных

SQLite — реляционная БД, которая хранится в одном файле (или в памяти) и не требует отдельного сервера. Библиотека SQLite вкомпилирована в приложение — никаких портов, демонов и подключений по сети.

В Go SQLite используется через драйвер [`mattn/go-sqlite3`](https://github.com/mattn/go-sqlite3) (CGO, быстрее) или [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite) (чистый Go, без CGO):

```go
import (
    "database/sql"
    _ "modernc.org/sqlite" // чистый Go, работает без CGO
)

db, _ := sql.Open("sqlite", "file:app.db?_journal_mode=WAL")
```

Ключевое отличие: connection string — путь к файлу, а не сетевой адрес.

***

### Когда использовать SQLite

| Сценарий | Почему SQLite |
|---|---|
| Локальные приложения (CLI, desktop) | Не нужен сервер БД — один бинарник, один файл |
| Тесты | Мгновенный запуск, не нужен Docker |
| Встраиваемые системы, IoT | Минимальные ресурсы, нет сети |
| Прототипы и MVP | Ноль настройки — `:memory:` для временной БД |
| Кеширование и промежуточные данные | Файловая БД быстрее сетевого вызова |

**Когда НЕ использовать:**
- Многопользовательские веб-приложения (конкурентная запись ограничена)
- Большие объёмы данных (> 1 ТБ)
- Высокая частота конкурентных записей

### Режимы работы

**Файловая БД** — данные живут в одном файле:

```go
db, _ := sql.Open("sqlite", "file:myapp.db?_journal_mode=WAL&_foreign_keys=on")
```

Параметры в connection string:

| Параметр | Зачем |
|---|---|
| `_journal_mode=WAL` | Write-Ahead Log — конкурентное чтение при записи |
| `_foreign_keys=on` | Включить внешние ключи (по умолчанию выключены!) |
| `_busy_timeout=5000` | Ждать 5 секунд при блокировке вместо мгновенной ошибки |

**WAL-режим.** В SQLite есть два режима журналирования: rollback journal (по умолчанию) и WAL.

В rollback-режиме читатели БЛОКИРУЮТ писателя и наоборот: пока один читает, никто не может писать. Для однопоточного приложения это нормально, для веб-сервера с конкурентными запросами — нет.

WAL (Write-Ahead Log) решает это: изменения пишутся в отдельный WAL-файл, а не напрямую в основную БД. Читатели работают с основной БД, писатель — с WAL-файлом. Результат: читатели и писатель не блокируют друг друга.

```
Без WAL:  читатель → [БД заблокирована] ← писатель ждёт
С WAL:    читатель → [основной файл БД]
          писатель → [WAL-файл]       ← параллельно!
```

Периодически WAL-файл сливается с основной БД (checkpoint). Для 99% приложений на SQLite WAL — обязательный режим. Всегда добавляйте `_journal_mode=WAL` в connection string.

**В памяти** — БД исчезает при закрытии соединения:

```go
db, _ := sql.Open("sqlite", ":memory:")
```

Идеально для тестов: не нужно чистить файлы после прогона.

### SQLite и `database/sql`

SQLite работает через `database/sql` — те же `db.Query`, `db.Exec`, транзакции. Но есть нюансы:

**Плейсхолдеры: `?` вместо `$1`.** SQLite использует вопросительные знаки:

```go
db.Query("SELECT * FROM users WHERE id = ?", 42) // правильно
db.Query("SELECT * FROM users WHERE id = $1", 42) // ОШИБКА в SQLite
```

**Типы данных: гибкая типизация.** SQLite не требует указывать тип колонки жёстко — но тип хранится для каждого значения. Основные mapped-типы:

| SQLite-тип | Go-тип |
|---|---|
| INTEGER | `int64` |
| REAL | `float64` |
| TEXT | `string` |
| BLOB | `[]byte` |
| NULL | `nil` (указатели или `sql.NullString`) |

### Тесты с SQLite

SQLite в памяти — быстрая альтернатива testcontainers для тестов, не требующих специфики PostgreSQL:

```go
func TestWithSQLite(t *testing.T) {
    db, _ := sql.Open("sqlite", ":memory:")
    defer db.Close()

    // Создать схему
    db.Exec(`CREATE TABLE users (
        id INTEGER PRIMARY KEY,
        name TEXT NOT NULL,
        email TEXT UNIQUE NOT NULL
    )`)

    // Тестировать
    db.Exec("INSERT INTO users (name, email) VALUES (?, ?)", "Alice", "alice@example.com")

    var name string
    db.QueryRow("SELECT name FROM users WHERE email = ?", "alice@example.com").Scan(&name)

    if name != "Alice" {
        t.Errorf("expected Alice, got %s", name)
    }
    // Никакой очистки — :memory: исчезает с Close()
}
```

Плюсы против testcontainers:
- **Скорость** — запуск мгновенный, без Docker
- **Простота** — не нужен Docker в CI
- **Детерминированность** — изолированное состояние на каждый тест

Минусы:
- Не PostgreSQL — не проверить специфичные для PG фичи (JSONB, `COPY`, `LISTEN/NOTIFY`)
- Нет проверки реального диалекта — можно случайно использовать синтаксис PostgreSQL, который в production упадёт

### Ограничения SQLite

1. **Конкурентная запись.** Только один пишущий в один момент. WAL-режим смягчает это (читатели не блокируют писателя), но для высоконагруженных систем с множеством параллельных записей SQLite не подходит.
2. **Типы данных.** Нет `DECIMAL`, `JSONB`, `ARRAY`, `ENUM`. Всё хранится как INTEGER/REAL/TEXT/BLOB.
3. **Нет сетевого доступа.** SQLite — встраиваемая БД. Несколько сервисов не могут подключиться к одной БД.
4. **Миграции.** `ALTER TABLE` ограничен: нельзя удалить колонку (до версии 3.35), переименовать можно не всё.

> **Зачем это Go-разработчику.** SQLite — идеальный компаньон для Go: один бинарник, один файл БД, никаких внешних зависимостей. Тесты на SQLite в памяти проходят мгновенно и не требуют Docker. Но не заменяйте PostgreSQL на SQLite в production бездумно: конкурентная запись и отсутствие PG-специфичных типов быстро станут проблемой. Правило: SQLite для локального/встраиваемого, PostgreSQL для серверного.

***

## Источники

### Официальная документация

* [pkg.go.dev/database/sql](https://pkg.go.dev/database/sql) — полная документация пакета `database/sql`
* [pkg.go.dev/database/sql/driver](https://pkg.go.dev/database/sql/driver) — интерфейс для реализации драйверов
* [Go Wiki: SQLDrivers](https://github.com/golang/go/wiki/SQLDrivers) — список всех драйверов баз данных для Go
* [Go Wiki: SQLInterface](https://github.com/golang/go/wiki/SQLInterface) — примеры использования `database/sql`
* [PostgreSQL Documentation](https://www.postgresql.org/docs/current/) — официальная документация PostgreSQL
* [SQLite Documentation](https://www.sqlite.org/docs.html) — официальная документация SQLite

### `database/sql`, `sqlx`, `pgx`

* [sqlx: иллюстрированный гайд](https://jmoiron.github.io/sqlx/) — Jason Moiron, автор `sqlx`
* [pgx: PostgreSQL Driver and Toolkit](https://github.com/jackc/pgx) — репозиторий `pgx` с примерами
* [Practical Persistence in Go: SQL Databases](https://www.alexedwards.net/blog/practical-persistence-sql) — Alex Edwards: `database/sql` от подключения до транзакций
* [Organising Database Access in Go](https://www.alexedwards.net/blog/organising-database-access) — Alex Edwards: паттерны доступа к БД, репозитории

### GORM

* [GORM Guides](https://gorm.io/docs/) — официальная документация: модели, запросы, ассоциации, миграции
* [GORM: The Good, The Bad, and The Ugly](https://dev.to/nadirbasalamah/gorm-the-good-the-bad-and-the-ugly-4p8o) — Nadir Basalamah: взвешенный разбор плюсов и минусов

### Миграции

* [golang-migrate](https://github.com/golang-migrate/migrate) — репозиторий с документацией: CLI, API, `embed`
* [Database Migrations in Go](https://medium.com/@cpk2468/database-migrations-in-golang-79bbf2e0c4a1) — пошаговый туториал по `golang-migrate`

### Безопасность

* [OWASP SQL Injection](https://owasp.org/www-community/attacks/SQL_Injection) — полный разбор SQL-инъекций от OWASP
* [OWASP: Query Parameterization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Query_Parameterization_Cheat_Sheet.html) — параметризованные запросы для всех языков и БД

### Производительность

* [Use The Index, Luke!](https://use-the-index-luke.com/) — Markus Winand: всё об индексах, планах запросов и оптимизации SQL
* [PostgreSQL: EXPLAIN](https://www.postgresql.org/docs/current/using-explain.html) — официальная документация по `EXPLAIN ANALYZE`
* [pg_stat_statements](https://www.postgresql.org/docs/current/pgstatstatements.html) — мониторинг запросов в PostgreSQL

### Тестирование

* [testcontainers-go](https://golang.testcontainers.org/) — официальная документация: PostgreSQL, MySQL, Redis в тестах
* [Testing Databases in Go](https://dev.to/kevin_van_der_wijst/testing-databases-in-go-4kbf) — Kevin van der Wijst: транзакционные тесты, testcontainers

### SQLite в Go

* [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) — драйвер SQLite на чистом Go, без CGO
* [Using SQLite in Go](https://earthly.dev/blog/golang-sqlite/) — туториал: подключение, запросы, `:memory:`, тесты

***

