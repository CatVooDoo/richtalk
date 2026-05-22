# RichTalk — Backend Architecture

## 1. Слои backend

```
services/api/
├── cmd/
│   └── api/
│       └── main.go          # точка входа: wire dependencies, start server
├── internal/
│   ├── handler/             # HTTP handlers + WS upgrader
│   │   ├── auth.go
│   │   ├── chat.go
│   │   ├── message.go
│   │   ├── user.go
│   │   └── ws.go
│   ├── service/             # бизнес-логика
│   │   ├── auth.go
│   │   ├── chat.go
│   │   ├── message.go
│   │   └── user.go
│   ├── repository/          # работа с БД (только SQL, никакой логики)
│   │   ├── user.go
│   │   ├── chat.go
│   │   └── message.go
│   ├── domain/              # чистые Go-структуры (User, Chat, Message)
│   │   └── models.go
│   ├── ws/                  # WebSocket hub + Redis fan-out
│   │   ├── hub.go
│   │   └── client.go
│   ├── middleware/          # JWT validation, logging, CORS
│   │   └── auth.go
│   └── config/              # читает env, возвращает typed Config struct
│       └── config.go
├── migrations/              # SQL миграции (golang-migrate)
└── go.mod
```

### Почему такая структура

**Handler** — знает только HTTP: парсит request, вызывает service, пишет response. Не знает про SQL.

**Service** — вся бизнес-логика: проверяет права ("только автор может редактировать"), вызывает несколько repository-методов в одной транзакции, публикует события в Redis. Не знает про HTTP.

**Repository** — чистый data-access: принимает и возвращает domain-структуры, инкапсулирует SQL. Не знает про бизнес-правила.

Это не полный Clean Architecture с интерфейсами на каждый слой — для MVP это избыточно. Но разделение handler/service/repository даёт: тестируемость service без HTTP, замену БД без правки хендлеров, и чёткое место для каждой новой фичи.

---

## 2. Auth-флоу

### Токены

| Токен         | Тип     | TTL      | Где хранится |
|---------------|---------|----------|--------------|
| access_token  | JWT     | 15 мин   | Только в памяти клиента (не localStorage) |
| refresh_token | Opaque  | 30 дней  | HttpOnly cookie + хеш в БД |

**Access token** — подписанный JWT (HS256 или RS256), содержит `user_id` и `exp`. Не хранится в БД — проверяется только подписью. Если нужно немедленно отозвать (бан, смена пароля) — требует короткого TTL или blocklist в Redis.

**Refresh token** — криптографически случайная строка (32 байта, base64url). В БД хранится только `bcrypt(token)` — так утечка таблицы `refresh_tokens` не даёт злоумышленнику использовать токены. При каждом `/auth/refresh` старый токен удаляется, выдаётся новый (rotation). Обнаружение повторного использования (если кто-то украл и ротировал до владельца) — будущее расширение.

### Флоу

```
[Login] ──▶ issue access_token (15m) + refresh_token (30d)
                │
                ▼
         access_token в памяти клиента
         refresh_token в HttpOnly cookie
                │
         через 15m access_token истёк
                │
                ▼
[Refresh] ─POST /auth/refresh──▶ новый access_token + ротированный refresh_token
                │
                ▼
         при logout ──▶ DELETE refresh_token из БД
```

### JWT payload

```json
{
  "sub": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "iat": 1705312200,
  "exp": 1705313100
}
```

Минимальный payload — только `sub` (user_id) + стандартные claims. Роли и права — будущее расширение.

---

## 3. Доставка сообщений в реальном времени

### Схема

```
  ┌─────────────────────────────────────────────────────────────────┐
  │                         Nginx                                    │
  │          /api/* ──▶ api:8080     /ws ──▶ api:8080 (WS upgrade)  │
  └──────────────────────────┬──────────────────┬────────────────────┘
                             │                  │
                    HTTP     │                  │ WebSocket
                             ▼                  ▼
  ┌──────────────────────────────────────────────────────────────────┐
  │                      API Instance 1                              │
  │                                                                  │
  │  POST /api/chats/{id}/messages                                   │
  │         │                                                        │
  │  [1] INSERT message INTO postgres                                │
  │         │                                                        │
  │  [2] PUBLISH "chat:{id}" → Redis ──────────────────────────────▶│──┐
  │         │                                                        │  │
  │  [3] 201 Created ──▶ Client A                                   │  │
  │                                                                  │  │
  │  WS Hub (goroutine)                                              │  │
  │  ◀── SUBSCRIBE "chat:{id}" (own publish received back) ──────────│◀─┘
  │         │                                                        │
  │  [4] find local WS clients in chat:{id}                         │
  │         │                                                        │
  │  [5] write message.new to Client A (other device / same chat)   │
  └──────────────────────────────────────────────────────────────────┘

  ┌──────────────────────────────────────────────────────────────────┐
  │                      API Instance 2                              │
  │                                                                  │
  │  WS Hub (goroutine)                                              │
  │  ◀── SUBSCRIBE "chat:{id}" ──────────────────────────────────────│◀── Redis
  │         │                                                        │
  │  [4] find local WS clients in chat:{id}                         │
  │         │                                                        │
  │  [5] write message.new to Client B                               │
  └──────────────────────────────────────────────────────────────────┘
```

### Детали реализации Hub

- Каждый API-инстанс держит in-memory `Hub`: `map[chatID][]WSClient`
- При подключении клиента: загрузить список его чатов → добавить его в Hub → подписаться на `chat:{id}` в Redis для всех его чатов, где ещё не подписаны
- При отключении клиента: удалить из Hub → отписаться от каналов, где больше нет локальных клиентов
- Одна горутина на Redis SUBSCRIBE (pub/sub listener) — читает сообщения и dispatches в Hub
- Отправка клиенту: горутина-writer на каждое WS-соединение с буферизованным каналом; если буфер заполнен — соединение считается медленным и закрывается

### Typing-события

Не сохраняются в БД. Клиент → Instance → Redis PUBLISH → все инстансы → другие участники чата. Если Redis недоступен, typing просто не приходит — это приемлемо для ephemeral-события.

---

## 4. Как это будет расти

### Групповые чаты

Добавить `'group'` в enum `chat_type`. Таблицы `chats` и `chat_members` уже поддерживают N участников — никаких изменений схемы. `direct_chat_lookup` используется только для `type='direct'`. Нужно добавить: поле `role` в `chat_members` (owner/admin/member), поле `name` в `chats` (уже есть).

### Файлы и медиа

Добавить `message_type ENUM ('text', 'image', 'file')` в `messages`. Добавить таблицу `message_attachments (id, message_id, storage_key, mime_type, size)`. Сами файлы — S3-совместимое хранилище (Minio для self-host). Схема сообщений не ломается.

### Реакции

Новая таблица `message_reactions (message_id, user_id, emoji, created_at)` с PRIMARY KEY (message_id, user_id, emoji). Не требует изменений в `messages`.

### Reply / Forward

`messages` получит `reply_to_id UUID NULL REFERENCES messages(id)` и `forwarded_from_id UUID NULL`. Additive migration — нет breaking changes.

### Online-статусы

Redis Hash: `user:online:{user_id}` с TTL. Обновляется при каждом WS ping. Не требует изменений в БД.

### Масштабирование

Сейчас: 1 инстанс API. Redis Pub/Sub уже заложен — горизонтальное масштабирование API работает без дополнительной работы. Следующий шаг: read-реплика Postgres для тяжёлых SELECT (история сообщений).

---

## 5. Что сознательно НЕ делаем в MVP

| Фича | Причина |
|------|---------|
| End-to-end шифрование | Требует key exchange protocol (Signal), кардинально усложняет архитектуру |
| Групповые и публичные чаты | Вторая итерация — схема БД уже готова |
| Файлы и медиа | Требует S3 + CDN + upload flow |
| Реакции | UI-фича, не блокер |
| Reply / Forward | UI-фича, не блокер |
| Online-статусы (last seen) | Privacy concerns + дополнительная нагрузка на Redis |
| Push-уведомления | Требует APNs/FCM интеграцию и мобильные клиенты |
| Звонки (audio/video) | WebRTC — отдельный проект |
| Аватарки и профиль | Требует файловое хранилище |
| Чтение сообщений (read receipts) | Дополнительная таблица + события — вторая итерация |
| Поиск по тексту сообщений | Full-text search (pg tsvector или Meilisearch) — после MVP |
| Rate limiting | Нужен, но настраивается на уровне Nginx/middleware — не в схеме |
| Email-верификация | Требует SMTP — пропускаем для MVP |
