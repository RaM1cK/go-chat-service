# Spoty Chat

Сервис сообщений и чатов для [Spotify Clone](https://github.com/RaM1cK/spotify-clone), реализованный на **Go**. Отвечает за реалтайм-переписку между пользователями.

Проект использует гибридную архитектуру хранения данных: реляционную (PostgreSQL) для метаданных чатов и пользователей, и NoSQL (ScyllaDB) для высоконагруженного хранения сообщений.

---

## Стек технологий

### Язык и библиотеки
- **Go 1.26**
- **pgx / pgxpool v5** — драйвер PostgreSQL
- **goose v3** — миграции PostgreSQL
- **gocql / gocqlx v3** — драйвер и ORM для ScyllaDB (CQL)
- **socket.io v3** ([zishang520/socket.io](https://github.com/zishang520/socket.io)) — WebSocket-фреймворк (rooms, broadcast, acks)
- **nats.go** — клиент NATS (межсервисный fan-out сообщений)
- **golang-jwt/jwt v5** — проверка JWT-токенов
- **google.golang.org/grpc** — gRPC-API для API-сервиса
- **sqlc** — codegen запросов PostgreSQL в Go из SQL

### Инфраструктура
- **PostgreSQL 16** — чаты, участники, друзья, пользователи
- **ScyllaDB** — хранение и timestamp-сортировка сообщений
- **NATS** — PUB/SUB брокер для рассылки сообщений между инстансами
- **Docker Compose** — оркестрация всех сервисов
- **goose** (SQL) + **gocqlx migrate** (CQL) — управление схемами

---

## Структура проекта

```
spoty-chat/
├── cmd/
│   └── main.go              # Точка входа: миграции, БД, WS-сервер, gRPC
├── internal/
│   ├── auth/                # Валидация JWT
│   ├── db/
│   │   ├── db.go            # Подключение к PostgreSQL / ScyllaDB, миграции
│   │   ├── pgsql/           # Сгенерированные sqlc запросы (pgx)
│   │   └── scylla/          # Сгенерированные модели gocqlx
│   ├── dto/                 # DTO сообщений и чатов
│   ├── grpc/                # gRPC-сервер (чаты, сообщения)
│   ├── repository/          # Слой доступа к данным
│   ├── service/             # Бизнес-логика
│   └── ws/                  # Socket.io сервер (server.go)
├── migrations/
│   ├── pgsql/               # Миграции goose (SQL)
│   └── scylla/              # Миграции gocqlx (CQL)
├── sql/
│   └── queries/             # SQL-запросы для sqlc
├── docker-compose.yaml
├── dockerfile
├── sqlc.yaml
├── .env / .env.test
├── go.mod
└── go.sum
```

---

## Схема данных

### PostgreSQL (метаданные)
| Таблица | Описание |
|---------|----------|
| **users** | Пользователи (UUID) |
| **chats** | Чаты (type: личный/группа, name, logo) |
| **chat_members** | Принадлежность пользователя к чату (M2M) |
| **dm_chats** | Личные чаты (пара user_a/user_b → chat_id, с CHECK user_a < user_b) |
| **friendships** | Дружба между пользователями (статус, timestamps) |

### ScyllaDB (сообщения)
| Таблица | Ключи |
|---------|-------|
| **messages_by_chat** | Partition key `chat_id`, clustering `created_at DESC, id DESC` |

Таблица сообщений оптимизирована под выборку истории сообщений конкретного чата в порядке убывания времени (новые сообщения первыми).

---

## Архитектура

Сервис поднимает два эндпоинта:
- **HTTP/WebSocket** на `SERVER_PORT` — socket.io на пути `/ws` (`/ws` и `/ws/` зарегистрированы, чтобы ловить хендшейки вида `/ws/?EIO=4...`);
- **gRPC** на `GRPC_PORT` — RPC для остальных сервисов (чаты, сообщения).

### WebSocket

Реалтайм построен на **socket.io v3**. Каждый клиент после хендшейка автоматически попадает в приватную комнату, названную его socket-id (`s.Join(Room(s.id))`). Комнаты чатов называются `chat:<chatID>`.

Клиент может подписаться на несколько чатов разом, прислав событие `join-room` с несколькими аргументами — сервер собирает их в `[]sio.Room` и вызывает единый `sock.Join(rooms...)`.

При горизонтальном масштабировании (несколько инстансов) сообщение сохраняется и публикуется через **NATS**: каждый инстанс публикует событие в subject `room.<chatID>` и подписан на `room.>` — доставка идёт клиентам **всех** инстансов.

### События WebSocket
| Событие | Направление | Описание |
|---------|-------------|----------|
| `join-room` | client → server | Подписка на один или несколько чатов: `emit("join-room", ...chatIds)` |
| `send-message` | client → server | Отправка сообщения `{room, msg}`; последний аргумент — ack-callback |
| `receive-message` | server → client | Доставка сообщение участникам комнаты (кроме сокета-отправителя) |
| `message-status` | — | Зарезервировано (обновление статусов через ack вместо отдельного события) |

### Поток отправки сообщения
1. Клиент шлёт `send-message` с `{room, msg}`, где `msg.id` — **клиентский UUID** (нужен для оптимистичного UI и дедупликации).
2. Сервер сохраняет сообщение в Scylla (`senderId` берётся из JWT, а не из payload).
3. При успехе в **ack** возвращается сохранённый `dto.Message` — клиент обновляет свой оптимистичный экземпляр (статус, `createdAt`).
4. При ошибке в ack возвращается `{error: "..."}`.
5. Вместе с ack сервер публикует сообщение в NATS subject `room.<chatID>`, прокинув туда **socket-id отправителя** (`SenderSid`).
6. Каждый инстанс в `deliverFromNats` делает broadcast `To(chat:<room>)` c `Except(SenderSid)` — сообщение получают все, **кроме сокета, который его отправил**.

### Несколько устройств одного пользователя
Исключается только конкретный сокет-отправитель, а не все сокеты пользователя. Поэтому если пользователь сидит на двух устройствах в одном чате:
- устройство A получает только **ack** (сразу, без исследования через broadcast);
- устройство B получает сообщение через **receive-message**, как и остальные.

В крайнем случае (сокет A переподключился до доставки broadcast) его старый sid мёртв, `Except` не находит никого — A получит дубликат `receive-message`. От этого защищает **дедупликация по id сообщения на клиенте** (клиентский UUID сохраняется сервером как есть), так что дубли не появляются в UI.

### Оптимистичный UI
- Клиент сразу рисует сообщение со статусом **Pending**.
- Ack с сохранённым сообщением обновляет статус и `createdAt`.
- Ack с `{error}` помечает сообщение как не отправленное.

### Аутентификация
При хендшейке `allowRequest` проверяет JWT из cookie `token` (валидация HMAC-подписи через `SECRET_KEY`, извлечение `userID` из claims). Это же `userID` сохраняется в `sock.Data()` и используется как `senderId` при отправке сообщений.

---

## Запуск

### Через Docker Compose

```bash
docker compose up --build
```

Запускаются ScyllaDB и PostgreSQL (ожидание готовности через healthcheck), стартует NATS и контейнер `chat`. Сервис доступен:
- HTTP/WebSocket: `http://localhost:8081` (в контейнере `8080`),
- gRPC: `localhost:50051`.

### Локальная разработка

Запуск миграций выполняется автоматически в `main.go` при старте:

```bash
go run ./cmd/main.go
```

Для генерации кода sqlc (при изменении SQL-запросов):

```bash
sqlc generate
```

---

## Переменные окружения (.env)

```env
SERVER_PORT=8080
GRPC_PORT=50051
SECRET_KEY=your_jwt_secret
WS_ORIGIN=*

NATS_URL=nats://localhost:4222

SCYLLA_URL=scylla
SCYLLA_KEYSPACE=messages

DB_NAME=chat
DB_USER=root
DB_PASSWORD=your_password
DATABASE_URL=postgres://root:your_password@localhost:5432/chat?sslmode=disable
```

> При запуске через docker-compose `SCYLLA_URL` указывает на имя сервиса `scylla`, `NATS_URL` — на `nats://nats:4222`, а `DATABASE_URL` собирается из `DB_*` для хоста `pgsql` (см. docker-compose.yaml). При локальном запуске ScyllaDB вместо этого следует указать `SCYLLA_URL=localhost`.
>
> `WS_ORIGIN` — допустимый `Origin` для CORS WebSocket-хендшейков (`*` для всех). Его обязан передавать браузер, поэтому без корректного значения браузерная сессия отклоняется на этапе handshake.