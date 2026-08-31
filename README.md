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
- **gorilla/websocket** — WebSocket-протокол
- **golang-jwt/jwt v5** — проверка JWT-токенов
- **sqlc** — codegen запросов PostgreSQL в Go из SQL

### Инфраструктура
- **PostgreSQL 16** — чаты, участники, друзья, пользователи
- **ScyllaDB** — хранение и timestamp-сортировка сообщений
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

### WebSocket Hub
`Hub` (internal/ws/hub.go) управляет комнатами (`rooms map[string]map[*Client]struct{}`) с помощью каналов `register`, `unregister` и `broadcast`, что делает его потокобезопасным без блокировок на этапе передачи.

### Клиенты
`Client` (internal/ws/client.go):
- **WritePump** — пишет исходящие сообщения и ping-keepalive
- **ReadPump** — читает входящие, обрабатывает join-room / send-message

### События WebSocket
| Событие (входящее) | Описание |
|--------------------|----------|
| `join-room` | Регистрация клиента в комнате (чате) |
| `send-message` | Отправка сообщения в комнату |
| `receive-message` | Доставка сообщения всем участникам комнаты |

### Аутентификация
При подключении к `/ws` проверяется JWT-токен из cookie `token` (валидация HMAC-подписи через `SECRET_KEY`, извлечение `userID` из claims).

---

## Запуск

### Через Docker Compose

```bash
docker compose up --build
```

Сервис становится доступен на `http://localhost:8080`. Запускаются и ожидают готовности (healthcheck) ScyllaDB и PostgreSQL.

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

SCYLLA_URL=scylla
SCYLLA_KEYSPACE=messages

DB_NAME=chat
DB_USER=root
DB_PASSWORD=your_password
```

> При запуске через docker-compose `SCYLLA_URL` указывает на имя сервиса `scylla` (см. docker-compose.yaml). При локальном запуске ScyllaDB вместо этого следует указать `SCYLLA_URL=localhost`.