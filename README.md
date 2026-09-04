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
- **Melody** (поверх gorilla/websocket) — WebSocket-фреймворк (pumps, heartbeats, broadcast)
- **nats.go** — клиент NATS (межсервисный fan-out сообщений)
- **golang-jwt/jwt v5** — проверка JWT-токенов
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
│   └── main.go              # Точка входа, подключение к БД, WebSocket
├── internal/
│   ├── auth/                # Валидация JWT
│   ├── db/
│   │   ├── pgsql/           # Сгенерированные sqlc запросы (pgx)
│   │   └── scylla/          # Сгенерированные модели gocqlx
│   ├── dto/                 # DTO сообщений
│   ├── repository/          # Слой доступа к данным
│   ├── service/             # Бизнес-логика
│   └── ws/                  # WebSocket hub и клиенты
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

### WebSocket
`Server` (internal/ws/server.go) построен на библиотеке **Melody** (поверх gorilla/websocket), которая берёт на себя WritePump/ReadPump, ping/pong heartbeat и конкурентную запись в сессии. На каждую сессию через `sess.Set("rooms", []string{...})` запоминается список комнат, на которые подписан клиент.

Клиент может подключаться к нескольким чатам одновременно, отправляя `join-room` для каждого. При доставке сообщения `BroadcastFilter` проверяет, входит ли комната в список подписанта — это позволяет получать сообщения сразу от всех выбранных чатов.

При горизонтальном масштабировании (несколько инстансов `chat-service`) сообщение сохраняется и публикуется через **NATS**: каждый `Server` публикует broadcast в subject `room.<chatID>` и подписан на `room.>` — доставка идёт клиентам **всех** инстансов, включая отправителя.

### События WebSocket
| Событие (входящее) | Описание |
|--------------------|----------|
| `join-room` | Регистрация клиента в комнате (чате) |
| `send-message` | Отправка сообщения в комнату |
| `receive-message` | Доставка сообщения всем участникам комнаты |
| `message-status` | Статус отправки для конкретного клиента (Pending → Sent/Failed) |

### Поток доставки (Pending → Sent → receive-message)
1. Клиент шлёт `send-message` (с уникальным `id` сообщения) → `sendMessage` сохраняет в Scylla
2. При успехе → клиенту отправляется `message-status: {id, status: "sent"}`
3. При ошибке → клиенту отправляется `message-status: {id, status: "failed"}`
4. Сообщение публикуется в NATS subject `room.<chatID>` и доставляется всем участникам
5. Каждый инстанс получает `receive-message` из NATS и рассылает его локальным сессиям

### Оптимистичный UI (Pending → Sent)
- Клиент сразу рисует сообщение в интерфейсе со статусом **Pending**
- Как только приходит `message-status: sent`, UI обновляет статус
- Если приходит `failed` — показывается ошибка и сообщение помечается как не отправленное
- Идентификатор `id` генерируется клиентом и используется для сопоставления Pending → Sent

### Аутентификация
При подключении к `/ws` проверяется JWT-токен из cookie `token` (валидация HMAC-подписи через `SECRET_KEY`, извлечение `userID` из claims) и кладётся в ключи сессии. Отправитель сообщения определяется по JWT, а не по payload.

---

## Запуск

### Через Docker Compose

```bash
docker compose up --build
```

Сервис становится доступен на `http://localhost:8080`. Запускаются и ожидают готовности (healthcheck) ScyllaDB и PostgreSQL, стартует NATS.

### Локальная разработка

Запуск миграций и сервера выполняются автоматически в `main.go` при старте:

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
SECRET_KEY=your_jwt_secret

NATS_URL=nats://localhost:4222

SCYLLA_URL=scylla
SCYLLA_KEYSPACE=messages

DB_NAME=chat
DB_USER=root
DB_PASSWORD=your_password
```

> При запуске через docker-compose `SCYLLA_URL` указывает на имя сервиса `scylla`, а `NATS_URL` — на `nats://nats:4222` (см. docker-compose.yaml). При локальном запуске ScyllaDB вместо этого следует указать `SCYLLA_URL=localhost`.