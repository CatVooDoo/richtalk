# RichTalk — WebSocket Protocol

## Connection

**URL:** `ws://localhost/ws`

### Authentication

JWT is sent as the **first message** after the TCP/WebSocket handshake, not in the URL query string.

**Why first message, not query param?**  
Query params appear in server access logs, browser history, and Nginx `$request_uri` logs — leaking the token to anyone with log access. The WebSocket handshake HTTP request is logged by Nginx; a token in the URL would be logged in plaintext. A first-message auth keeps the token inside the encrypted WebSocket payload and never touches the URL.

**Timeout:** The server closes the connection with code `4001` if an auth message is not received within **5 seconds** of the handshake.

### Auth handshake

After connecting, the client must send:

```json
{
  "type": "auth",
  "payload": {
    "access_token": "<jwt>"
  }
}
```

If auth succeeds, the server responds:

```json
{
  "type": "auth.ok",
  "payload": {
    "user_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479"
  }
}
```

If auth fails, the server sends and closes the connection:

```json
{
  "type": "auth.error",
  "payload": {
    "message": "Token expired or invalid"
  }
}
```

### Close codes

| Code | Reason |
|------|--------|
| 4001 | Auth timeout — no auth message received |
| 4002 | Invalid or expired token |
| 4003 | Malformed message |
| 1000 | Normal closure (client-initiated) |

---

## Message Format

All messages (both directions) use the same envelope:

```json
{
  "type": "<event_type>",
  "payload": { ... }
}
```

---

## Client → Server Events

### `typing.start`

User started typing in a chat.

```json
{
  "type": "typing.start",
  "payload": {
    "chat_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  }
}
```

Server broadcasts `typing.start` to other members of the chat (not back to sender).  
No response to sender.

---

### `typing.stop`

User stopped typing.

```json
{
  "type": "typing.stop",
  "payload": {
    "chat_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  }
}
```

---

## Server → Client Events

### `message.new`

A new message was sent to a chat the user is a member of.

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
    "content": "Hello, Bob!",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

> Note: also sent to the author's own connection so they get confirmation on other devices.

---

### `message.edited`

A message in a chat the user is a member of was edited.

```json
{
  "type": "message.edited",
  "payload": {
    "id": "d5e6f7a8-b9c0-4d1e-2f3a-4b5c6d7e8f9a",
    "chat_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "content": "Hello, Bob! (edited)",
    "updated_at": "2024-01-15T10:31:00Z"
  }
}
```

---

### `message.deleted`

A message was soft-deleted. Client should replace the content with a "сообщение удалено" placeholder.

```json
{
  "type": "message.deleted",
  "payload": {
    "id": "d5e6f7a8-b9c0-4d1e-2f3a-4b5c6d7e8f9a",
    "chat_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "deleted_at": "2024-01-15T10:32:00Z"
  }
}
```

---

### `typing.start` (broadcast)

Another member of a shared chat started typing.

```json
{
  "type": "typing.start",
  "payload": {
    "chat_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "user": {
      "id": "b3d1c2e4-6f8a-4b2c-9d0e-1f2a3b4c5d6e",
      "username": "bob"
    }
  }
}
```

---

### `typing.stop` (broadcast)

```json
{
  "type": "typing.stop",
  "payload": {
    "chat_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "user": {
      "id": "b3d1c2e4-6f8a-4b2c-9d0e-1f2a3b4c5d6e",
      "username": "bob"
    }
  }
}
```

---

## Redis Pub/Sub — Multi-instance Fan-out

### Problem

When multiple API instances run behind Nginx (or during scaling), Client A may connect to Instance 1 while Client B is connected to Instance 2. A WS event published by Instance 1 must also reach Client B on Instance 2.

### Solution: per-chat Redis channels

Each API instance maintains a set of goroutines that subscribe to Redis channels for every chat whose members are currently connected to that instance.

**Channel naming:** `chat:{chat_id}`  
Example: `chat:a1b2c3d4-e5f6-7890-abcd-ef1234567890`

### Flow for `message.new`

```
Client A                API Instance 1           Redis Pub/Sub        API Instance 2          Client B
   │                          │                        │                     │                    │
   │── POST /api/chats/{id}/messages ──────────────────│                     │                    │
   │                          │                        │                     │                    │
   │                   [INSERT INTO messages]          │                     │                    │
   │                          │                        │                     │                    │
   │                          │── PUBLISH chat:{id} ──▶│                     │                    │
   │                          │                        │                     │                    │
   │◀─── 201 Created ─────────│                        │─── message ────────▶│                    │
   │                          │                        │                     │                    │
   │                          │◀──────── message ──────│   (Instance 1 also  │                    │
   │◀─── WS: message.new ─────│           (own sub)    │   receives its own  │── WS: message.new ▶│
   │                          │                        │   publish)          │                    │
```

### Subscription lifecycle

- When a user's WS connection is established (after auth), the instance checks which chats that user belongs to and subscribes to `chat:{id}` for each.
- When a user disconnects, the instance unsubscribes from channels where no other local client remains.
- On receiving a publish on `chat:{id}`, the instance looks up all locally connected WS clients that are members of that chat and writes the event to their connections.

### Typing events

Typing events (`typing.start` / `typing.stop`) are **not persisted to the DB** — they are published directly to Redis and fan out to other members. They are transient: no delivery guarantee, no replay.

### Future: at-least-once delivery

When offline delivery or push notifications are needed, a message queue (e.g., Redis Streams or NATS JetStream) can replace Pub/Sub for `message.new` while keeping Pub/Sub for ephemeral typing events.
