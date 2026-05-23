# RichTalk — Architecture

> Актуально для ветки `feature/frontend`, коммит ~0e14333 (май 2026).

---

## 1. Репозиторий: структура сервисов

```
RichTalk/
├── services/
│   ├── api/          # Go backend
│   └── frontend/     # React frontend
├── nginx/
│   └── default.conf  # reverse proxy: /api + /ws → api:8080
├── docker-compose.yml
├── .env              # секреты (не в git)
└── docs/
```

---

## 2. Backend (`services/api`)

### Структура файлов

```
services/api/
├── cmd/api/main.go              # точка входа: читает env, создаёт App
├── internal/
│   ├── app/app.go               # DI wiring + HTTP server goroutine
│   ├── config/config.go         # typed Config struct, читает os.Getenv
│   ├── model/                   # чистые Go-структуры, ошибки-значения
│   │   ├── chat.go              # Chat, ChatWithLastMessage, OtherUser, LastMessage, ChatType
│   │   ├── message.go           # Message, Author, MessageEventPayload, ToEventPayload()
│   │   └── user.go              # User + sentinel errors (ErrUserNotFound и т.д.)
│   ├── repository/              # только SQL, никакой логики
│   │   ├── chat_repo.go         # ListByUserID, CreateDirect, GetByID, CreateNotesChat, GetNotesChat, GetMemberIDs
│   │   ├── message_repo.go      # Insert, GetByID, List, SoftDelete, Update
│   │   ├── refresh_repo.go      # Create, FindByToken, Delete (хранит bcrypt-хеш RT)
│   │   └── user_repo.go         # Create, FindByUsername, FindByID, Search
│   ├── service/                 # бизнес-логика
│   │   ├── auth_service.go      # Register (+ создаёт notes-чат), Login, Refresh, Logout
│   │   ├── chat_service.go      # GetOrCreateDirect, ListChats, GetChat, GetNotesChat
│   │   ├── jwt_service.go       # Issue, Validate (HS256, 15m TTL)
│   │   ├── message_service.go   # SendMessage (+ Redis fan-out), EditMessage, DeleteMessage, ListMessages
│   │   └── user_service.go      # Me, Search
│   ├── handler/                 # HTTP: парсинг запроса → service → JSON
│   │   ├── router.go            # chi-роутер, все маршруты
│   │   ├── auth_handler.go
│   │   ├── chat_handler.go
│   │   ├── health_handler.go
│   │   ├── message_handler.go
│   │   └── user_handler.go
│   ├── ws/                      # WebSocket hub
│   │   ├── handler.go           # ServeWS: проверяет ?token= → upgrade → запускает Client
│   │   ├── hub.go               # event loop: register/unregister/dispatch/broadcastTyping
│   │   ├── client.go            # readPump / writePump горутины, ping/pong
│   │   └── events.go            # ClientEvent, TypingPayload; re-export типов из event/
│   ├── event/event.go           # Envelope {Type, Payload, TargetUserIDs} + RedisChannel; импорт-нейтральный пакет
│   ├── middleware/              # Auth (JWT), Logger, Recover
│   ├── httpx/                   # JSON/error helpers, DecodeValidate
│   └── migrations/
│       ├── 000001_init.{up,down}.sql
│       ├── 000002_users.{up,down}.sql
│       ├── 000003_chats.{up,down}.sql
│       ├── 000004_messages.{up,down}.sql
│       └── 000005_notes_chat.{up,down}.sql  # enum 'notes', attachment columns
└── go.mod
```

### Слои и их границы

| Слой | Знает про | Не знает про |
|------|-----------|--------------|
| **handler** | HTTP, JSON, path params | SQL, бизнес-правила |
| **service** | Репозитории, Redis Pub/Sub | HTTP, SQL |
| **repository** | SQL, pgx | Бизнес-правила, HTTP |
| **model** | Только Go-типы | Всё остальное |

`event/` — отдельный пакет без зависимостей от `service` или `ws`, чтобы разорвать import cycle: service публикует в Redis, ws читает из Redis — оба импортируют только `event`.

---

## 3. База данных (PostgreSQL)

### Ключевые таблицы

```
users           id, username (unique), password_hash, created_at
refresh_tokens  id, user_id, token_hash (bcrypt), expires_at
chats           id, type (enum: direct|notes), name, created_at, updated_at
chat_members    chat_id, user_id, joined_at  — многие-ко-многим
direct_chat_lookup  chat_id, user1_id, user2_id  — уникальный индекс на (user1_id, user2_id)
messages        id, chat_id, author_id, content, attachment_*, created_at, updated_at, deleted_at
```

### chat_type enum

- `direct` — чат между двумя пользователями, идентифицируется через `direct_chat_lookup`
- `notes` — самочат пользователя, создаётся при регистрации (1 участник)

### Мягкое удаление сообщений

`messages.deleted_at TIMESTAMPTZ NULL` — при удалении выставляется timestamp, `content` не трогается. Клиент получает `"deleted": true` и `"content": ""`.

---

## 4. Auth-флоу

### Токены

| Токен | Тип | TTL | Где хранится |
|-------|-----|-----|--------------|
| access_token | JWT (HS256) | 15 мин | In-memory у клиента (Zustand authStore) |
| refresh_token | Opaque (32 байта, hex) | 30 дней | `localStorage` с ключом `rt`; в БД хранится `bcrypt(token)` |

**Почему не HttpOnly cookie для RT:** В текущей реализации RT передаётся в теле JSON-запроса и хранится в `localStorage`. HttpOnly cookie — будущее улучшение безопасности.

### Флоу

```
[Register/Login] ──▶ { access_token, refresh_token, user }
                          │
                 access_token → Zustand (память)
                 refresh_token → localStorage['rt']
                          │
               через 15m: любой API-запрос возвращает 401
                          │
                 Axios interceptor: POST /api/auth/refresh { refresh_token }
                          │
                 новый access_token + ротированный refresh_token
                          │
               повторяем оригинальный запрос с новым токеном
                          │
          [Logout]: POST /api/auth/logout { refresh_token } → удаляет RT из БД
```

### Параллельные 401

Axios interceptor ставит новые запросы в очередь на время обновления токена — `isRefreshing` флаг + Promise-очередь. Все накопившиеся запросы переотправляются с одним новым access_token.

---

## 5. WebSocket и доставка событий

### Аутентификация WS

JWT передаётся **в query-параметре** `?token=<access_token>` ещё до HTTP Upgrade.  
Сервер проверяет токен до апгрейда и возвращает HTTP 401 при ошибке — это позволяет вернуть нормальный HTTP-статус, после Upgrade это уже невозможно.

```
GET /ws?token=<jwt> HTTP/1.1
Upgrade: websocket
```

### Hub

Hub — единый goroutine, владеющий `map[userID][]*Client`. Все мутации (register/unregister/incoming) идут через каналы — никаких mutex.

```
    register chan *Client
    unregister chan *Client
    incoming chan incomingMessage   (typing events от клиентов)
```

### Redis fan-out

**Один глобальный канал** `richtalk:events` (не per-chat).  
Конверт:

```json
{
  "type": "message.new",
  "payload": { ... },
  "target_user_ids": ["uuid1", "uuid2"]
}
```

`TargetUserIDs` заполняется при публикации (по `chat_members`). Hub каждого API-инстанса подписан на `richtalk:events` и доставляет только тем `userID`, которые есть у него в памяти. Горизонтальное масштабирование: N инстансов, один Redis-канал, никакой координации между инстансами.

### Медленные клиенты

Если канал `Client.send` (буфер 256 сообщений) переполнен — соединение закрывается немедленно, сообщение дропается.

---

## 6. Frontend (`services/frontend`)

```
services/frontend/src/
├── api/
│   ├── client.ts     # axios instance, interceptors (401 → refresh → retry)
│   ├── auth.ts       # register, login, logout (plain axios, без auth header)
│   ├── chats.ts      # getChats, createDirect, getNotesChat, getMessages, sendMessage, editMessage, deleteMessage
│   └── users.ts      # getMe, searchUsers
├── store/
│   ├── authStore.ts  # user, accessToken, isAuthenticated; setAuth, setUser, logout
│   ├── chatStore.ts  # chats[], activeChat, messages[chatId], hasMore[chatId], typingUsers[chatId]
│   └── wsStore.ts    # connected, send(event)
├── hooks/
│   ├── useWebSocket.ts  # connect/reconnect (max 5 retries, 3s delay), dispatch WS events
│   └── useAuth.ts       # signOut()
├── components/
│   ├── Avatar/       # инициалы + hash-based градиент по username
│   ├── Button/       # variant: primary | ghost | icon
│   ├── Input/
│   ├── GlassPanel/
│   └── PrivateRoute/ # проверяет RT в localStorage → getMe() → redirect /login
├── pages/
│   ├── Auth/         # LoginPage, RegisterPage — общий Auth.module.css
│   └── Messenger/
│       ├── Sidebar/  # список чатов, debounced поиск (300ms), unread badge
│       └── ChatArea/ # day dividers, infinite scroll вверх, пузыри, context menu, edit bar, typing indicator
├── index.css         # глобальные классы: .glass, .glass-strong, scrollbar
└── vite-env.d.ts     # /// <reference types="vite/client" /> — типы для *.module.css
```

### Дизайн-система: Liquid Glass

- Фон: `linear-gradient(135deg, #1a0533, #0d1b4b, #0a2a6e)`
- `.glass` / `.glass-strong` — `backdrop-filter: blur + saturate` + полупрозрачный белый border
- Акцентный цвет: `#a855f7` (purple-500)
- Радиусы: каждый компонент задаёт свой `border-radius`, глобальные `.glass`-классы его **не** задают (иначе конфликт)

### chatStore: per-chat storage

```ts
messages:    Record<chatId, Message[]>   // oldest first
hasMore:     Record<chatId, boolean>
typingUsers: Record<chatId, string[]>    // userIDs, кто сейчас печатает
```

`addMessage` дедуплицирует по ID — HTTP-ответ `sendMessage` и WS `message.new` доставляют одно и то же сообщение; в store попадает только одно.

---

## 7. Nginx

```
/api/* → http://api:8080      (HTTP reverse proxy)
/ws    → ws://api:8080/ws     (WebSocket proxy, с заголовками Upgrade)
/      → frontend:5173        (Vite dev server в dev-режиме)
```

---

## 8. Docker Compose

| Сервис | Образ | Порт |
|--------|-------|------|
| postgres | postgres:16-alpine | 5432 (внутренний) |
| redis | redis:7-alpine | 6379 (внутренний) |
| api | golang:1.23 + air | 8080 (внутренний) |
| frontend | node:20-alpine | 5173 (внутренний) |
| nginx | nginx:1.27-alpine | **80 (внешний)** |

API-контейнер использует `air` для hot-reload в dev. Миграции применяются при старте API (`golang-migrate`).

---

## 9. Что намеренно не в MVP

| Фича | Почему отложена |
|------|----------------|
| Групповые и публичные чаты | Схема БД готова (N членов, enum); нужен UI + роли |
| Файлы и медиа | Attachment-колонки в БД есть; нужен S3 + upload flow |
| HttpOnly cookie для RT | Текущий вариант (localStorage) проще; cookie — следующий шаг |
| Read receipts | Дополнительная таблица + WS-событие |
| Online-статусы | Redis Hash с TTL; privacy-решение требует обсуждения |
| Реакции, Reply, Forward | UI-фичи, не блокер |
| Push-уведомления | Требует APNs/FCM + мобильный клиент |
| Full-text search | pg tsvector или Meilisearch — после MVP |
| Rate limiting | Nginx/middleware уровень, не в схеме сейчас |
| Email-верификация | Требует SMTP |
| E2E-шифрование | Signal Protocol — кардинально другая архитектура |
