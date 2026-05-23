# RichTalk — WebSocket Protocol

## Подключение и аутентификация

**URL:** `ws://localhost/ws?token=<access_token>`

JWT передаётся в **query-параметре `token`** ещё до HTTP Upgrade.

### Почему query-param, а не первое сообщение?

Токен проверяется **до** апгрейда соединения. Это позволяет вернуть нормальный HTTP 401 при неверном/истёкшем токене — после того как Upgrade выполнен, отправить HTTP-статус уже невозможно. Альтернатива "первым сообщением" усложняет обработку ошибок и требует таймаут-логики на сервере.

> В dev-среде токен попадает в access-логи Nginx. В production рекомендуется переключиться на cookie или добавить Nginx-правило для маскировки query.

### Флоу соединения (frontend)

```
1. Пользователь залогинен → access_token в authStore (Zustand)
2. useWebSocket hook строит URL: ws://host/ws?token=<access_token>
3. new WebSocket(url)
4. ws.onopen → connected = true, retries = 0
5. ws.onclose(code !== 4001) → retry через 3s, макс 5 попыток
6. ws.onclose(code === 4001) → не переподключаться (невалидный токен)
```

### HTTP-ответы при ошибках до Upgrade

| HTTP status | Причина |
|-------------|---------|
| 401 | Параметр `token` отсутствует |
| 401 | Токен невалиден или истёк |

После успешного Upgrade ошибки передаются через WebSocket Close frames.

### WebSocket Close codes

| Code | Причина |
|------|---------|
| 4001 | Невалидный/истёкший токен (фактически: HTTP 401 до Upgrade → клиент трактует как 4001 и не переподключается) |
| 1000 | Нормальное закрытие (клиент) |
| 1001 | Going Away |

---

## Формат сообщений

Все сообщения в обоих направлениях используют одну обёртку:

```json
{
  "type": "<event_type>",
  "payload": { ... }
}
```

---

## Client → Server

### `typing.start`

Пользователь начал печатать в чате.

```json
{
  "type": "typing.start",
  "payload": {
    "chat_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  }
}
```

Сервер публикует событие в Redis → все инстансы → другие участники чата (отправителю не возвращается). Ответа не ждём.

---

### `typing.stop`

Пользователь прекратил печатать (автоматически через 3s тишины на клиенте).

```json
{
  "type": "typing.stop",
  "payload": {
    "chat_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  }
}
```

---

## Server → Client

### `message.new`

Новое сообщение в любом чате пользователя.  
**Отправляется в том числе отправителю** — для синхронизации между устройствами.  
Payload идентичен HTTP-ответу `POST /api/chats/{id}/messages`.

```json
{
  "type": "message.new",
  "payload": {
    "id": "d5e6f7a8-b9c0-4d1e-2f3a-4b5c6d7e8f9a",
    "chat_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "author": {
      "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "username": "alice"
    },
    "content": "Hello!",
    "deleted": false,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

> **Дедупликация на клиенте:** HTTP-ответ `sendMessage` и `message.new` от WS приходят оба. `chatStore.addMessage` пропускает сообщение если ID уже есть — в store попадает только одно.

---

### `message.edited`

Сообщение отредактировано. Payload идентичен HTTP `PATCH /api/messages/{id}`.

```json
{
  "type": "message.edited",
  "payload": {
    "id": "d5e6f7a8-b9c0-4d1e-2f3a-4b5c6d7e8f9a",
    "chat_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "author": {
      "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "username": "alice"
    },
    "content": "Hello! (edited)",
    "deleted": false,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:31:00Z"
  }
}
```

---

### `message.deleted`

Сообщение мягко удалено. Клиент заменяет контент плейсхолдером.

```json
{
  "type": "message.deleted",
  "payload": {
    "id": "d5e6f7a8-b9c0-4d1e-2f3a-4b5c6d7e8f9a",
    "chat_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "deleted": true,
    "content": ""
  }
}
```

---

### `typing.start` (broadcast)

Другой участник чата начал печатать.  
Payload содержит только `user_id` (не объект с username).

```json
{
  "type": "typing.start",
  "payload": {
    "chat_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "user_id": "b3d1c2e4-6f8a-4b2c-9d0e-1f2a3b4c5d6e"
  }
}
```

Клиент (chatStore) добавляет `user_id` в `typingUsers[chat_id]`. Sidebar показывает "печатает..." в подзаголовке активного чата, ChatArea показывает анимированный индикатор (3 точки).

---

### `typing.stop` (broadcast)

```json
{
  "type": "typing.stop",
  "payload": {
    "chat_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "user_id": "b3d1c2e4-6f8a-4b2c-9d0e-1f2a3b4c5d6e"
  }
}
```

---

## Redis Pub/Sub — архитектура fan-out

### Один глобальный канал

Канал: **`richtalk:events`** (не per-chat, один на весь кластер).

```go
// event/event.go
const RedisChannel = "richtalk:events"

type Envelope struct {
    Type          Type            `json:"type"`
    Payload       json.RawMessage `json:"payload"`
    TargetUserIDs []string        `json:"target_user_ids"`
}
```

`TargetUserIDs` вычисляется при публикации по таблице `chat_members`. Hub каждого API-инстанса подписан на единый канал и доставляет только тем пользователям, у которых есть активное соединение **на этом инстансе**.

### Hub: map[userID][]*Client

```go
clients map[string][]*Client  // userID → slice (несколько вкладок)
```

Один userID → несколько соединений. Событие доставляется всем соединениям этого пользователя.

### Флоу `message.new`

```
Client A                   API Instance 1                Redis               API Instance 2          Client B
   │                             │                          │                        │                   │
   │─ POST /chats/{id}/messages ─▶│                          │                        │                   │
   │                             │                          │                        │                   │
   │                      [INSERT INTO messages]             │                        │                   │
   │                             │                          │                        │                   │
   │                             │─ PUBLISH richtalk:events ▶│                        │                   │
   │                             │   {type, payload,         │                        │                   │
   │                             │    target_user_ids: [A,B]}│                        │                   │
   │◀─ 201 Created ──────────────│                          │                        │                   │
   │                             │                          │─── envelope ──────────▶│                   │
   │                             │◀────── own envelope ─────│   (Instance 2 delivers │                   │
   │◀── WS: message.new ─────────│          (delivers to A) │    to B if connected)  │── WS: message.new ▶│
```

### Флоу `typing.start`

```
Client A ─ typing.start ──▶ Hub (Instance 1) ─ broadcastTyping()
                                  │
                           GetMemberIDs(chat_id)   [DB call в горутине]
                                  │
                           PUBLISH richtalk:events {type: "typing.start",
                                                    payload: {chat_id, user_id: A},
                                                    target_user_ids: [B, C, ...]}
                                  │
                  Redis ──────────▶ Instance 1 Hub + Instance 2 Hub + ...
                                         │                  │
                                  deliver to B        deliver to C
```

Typing-события не сохраняются в БД. Нет гарантии доставки — это приемлемо для ephemeral-событий.

### Масштабирование

N инстансов API — один Redis-канал. Никакой координации между инстансами не нужно. Следующий шаг для надёжной доставки `message.new` — Redis Streams вместо Pub/Sub (at-least-once с replay); typing остаётся на Pub/Sub.

---

## Ping / Pong

Сервер отправляет WebSocket Ping каждые ~54 секунды (`pingPeriod = pongWait * 9/10`, `pongWait = 60s`). Если Pong не пришёл в течение 60s — соединение закрывается. Браузер отвечает Pong автоматически.
