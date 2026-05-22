# RichTalk — HTTP API Reference

Base URL: `http://localhost/api`  
All timestamps: ISO 8601 / RFC 3339 (`2024-01-15T10:30:00Z`)  
All IDs: UUID v4

## Authentication

Protected endpoints require `Authorization: Bearer <access_token>` header.  
Access token lifetime: **15 minutes**.  
Refresh token lifetime: **30 days** (rotated on each refresh).

---

## Auth

### POST /api/auth/register

Register a new user.

**Auth required:** No

**Request**
```json
{
  "username": "alice",
  "password": "s3cr3t_P@ssword"
}
```

**Response 201**
```json
{
  "user": {
    "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "username": "alice",
    "created_at": "2024-01-15T10:00:00Z"
  },
  "access_token": "<jwt>",
  "refresh_token": "<opaque_token>"
}
```

**Errors**
| Code | Reason |
|------|--------|
| 400  | Validation failed (username too short, invalid characters) |
| 409  | Username already taken |

---

### POST /api/auth/login

**Auth required:** No

**Request**
```json
{
  "username": "alice",
  "password": "s3cr3t_P@ssword"
}
```

**Response 200**
```json
{
  "user": {
    "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "username": "alice",
    "created_at": "2024-01-15T10:00:00Z"
  },
  "access_token": "<jwt>",
  "refresh_token": "<opaque_token>"
}
```

**Errors**
| Code | Reason |
|------|--------|
| 400  | Missing fields |
| 401  | Invalid credentials |

---

### POST /api/auth/refresh

Exchange a refresh token for a new access token + rotated refresh token.  
Old refresh token is invalidated after this call.

**Auth required:** No

**Request**
```json
{
  "refresh_token": "<opaque_token>"
}
```

**Response 200**
```json
{
  "access_token": "<new_jwt>",
  "refresh_token": "<new_opaque_token>"
}
```

**Errors**
| Code | Reason |
|------|--------|
| 401  | Token invalid, expired, or already rotated |

---

### POST /api/auth/logout

Invalidates the provided refresh token (deletes it from DB).

**Auth required:** Yes

**Request**
```json
{
  "refresh_token": "<opaque_token>"
}
```

**Response 204** (no body)

**Errors**
| Code | Reason |
|------|--------|
| 401  | Not authenticated |

---

## Users

### GET /api/users/me

Returns the authenticated user's profile.

**Auth required:** Yes

**Response 200**
```json
{
  "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "username": "alice",
  "created_at": "2024-01-15T10:00:00Z"
}
```

**Errors**
| Code | Reason |
|------|--------|
| 401  | Not authenticated |

---

### GET /api/users/search?q=alice

Search users by username prefix. Used to find someone to start a chat with.

**Auth required:** Yes

**Query params**
| Param | Type   | Required | Description |
|-------|--------|----------|-------------|
| q     | string | Yes      | Username prefix, min 2 chars |

**Response 200**
```json
{
  "users": [
    {
      "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "username": "alice"
    },
    {
      "id": "b3d1c2e4-6f8a-4b2c-9d0e-1f2a3b4c5d6e",
      "username": "alice_smith"
    }
  ]
}
```

**Errors**
| Code | Reason |
|------|--------|
| 400  | Query too short (< 2 chars) |
| 401  | Not authenticated |

---

## Chats

### GET /api/chats

List all chats the current user is a member of, ordered by most recent message.

**Auth required:** Yes

**Response 200**
```json
{
  "chats": [
    {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "type": "direct",
      "name": null,
      "other_user": {
        "id": "b3d1c2e4-6f8a-4b2c-9d0e-1f2a3b4c5d6e",
        "username": "bob"
      },
      "last_message": {
        "id": "c4d5e6f7-...",
        "content": "Hey!",
        "author_id": "b3d1c2e4-...",
        "created_at": "2024-01-15T10:29:00Z"
      },
      "created_at": "2024-01-15T09:00:00Z"
    }
  ]
}
```

**Errors**
| Code | Reason |
|------|--------|
| 401  | Not authenticated |

---

### POST /api/chats/direct

Create or retrieve the existing direct chat with another user. **Idempotent.**  
If a direct chat between the two users already exists, returns it with `201` replaced by `200`.

**Auth required:** Yes

**Request**
```json
{
  "user_id": "b3d1c2e4-6f8a-4b2c-9d0e-1f2a3b4c5d6e"
}
```

**Response 200** (existing) or **201** (created)
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "type": "direct",
  "other_user": {
    "id": "b3d1c2e4-6f8a-4b2c-9d0e-1f2a3b4c5d6e",
    "username": "bob"
  },
  "created_at": "2024-01-15T09:00:00Z"
}
```

**Errors**
| Code | Reason |
|------|--------|
| 400  | user_id missing or same as caller |
| 401  | Not authenticated |
| 404  | Target user not found |

---

### GET /api/chats/{id}

Get metadata for a single chat. Caller must be a member.

**Auth required:** Yes

**Response 200**
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "type": "direct",
  "other_user": {
    "id": "b3d1c2e4-6f8a-4b2c-9d0e-1f2a3b4c5d6e",
    "username": "bob"
  },
  "created_at": "2024-01-15T09:00:00Z"
}
```

**Errors**
| Code | Reason |
|------|--------|
| 401  | Not authenticated |
| 403  | Not a member of this chat |
| 404  | Chat not found |

---

### GET /api/chats/{id}/messages?before=&limit=

Paginated message history. Cursor-based, oldest-to-newest in response, but fetches backwards.

**Auth required:** Yes

**Query params**
| Param  | Type              | Required | Default | Description |
|--------|-------------------|----------|---------|-------------|
| before | ISO 8601 datetime | No       | now()   | Return messages with created_at < before |
| limit  | integer           | No       | 50      | 1–100 |

**Response 200**
```json
{
  "messages": [
    {
      "id": "c4d5e6f7-a8b9-4c0d-1e2f-3a4b5c6d7e8f",
      "chat_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "author": {
        "id": "b3d1c2e4-6f8a-4b2c-9d0e-1f2a3b4c5d6e",
        "username": "bob"
      },
      "content": "Hey!",
      "created_at": "2024-01-15T10:29:00Z",
      "updated_at": "2024-01-15T10:29:00Z",
      "deleted_at": null
    }
  ],
  "has_more": true,
  "next_cursor": "2024-01-15T10:29:00Z"
}
```

> **Pagination flow:** Take `next_cursor` from the response and pass it as `before` in the next request. Stop when `has_more` is `false`.

> **Deleted messages:** Included in the list with `deleted_at` set and `content` replaced by `""`. Client renders "Сообщение удалено".

**Errors**
| Code | Reason |
|------|--------|
| 400  | Invalid before/limit params |
| 401  | Not authenticated |
| 403  | Not a member of this chat |
| 404  | Chat not found |

---

## Messages

### POST /api/chats/{id}/messages

Send a new message to a chat. On success the message is also broadcast via WebSocket (`message.new`).

**Auth required:** Yes

**Request**
```json
{
  "content": "Hello, Bob!"
}
```

**Response 201**
```json
{
  "id": "d5e6f7a8-b9c0-4d1e-2f3a-4b5c6d7e8f9a",
  "chat_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "author": {
    "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "username": "alice"
  },
  "content": "Hello, Bob!",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z",
  "deleted_at": null
}
```

**Errors**
| Code | Reason |
|------|--------|
| 400  | Empty content |
| 401  | Not authenticated |
| 403  | Not a member of this chat |
| 404  | Chat not found |

---

### PATCH /api/messages/{id}

Edit a message. Only the author can edit. Broadcast via WebSocket (`message.edited`).

**Auth required:** Yes

**Request**
```json
{
  "content": "Hello, Bob! (edited)"
}
```

**Response 200**
```json
{
  "id": "d5e6f7a8-b9c0-4d1e-2f3a-4b5c6d7e8f9a",
  "chat_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "author": {
    "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "username": "alice"
  },
  "content": "Hello, Bob! (edited)",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:31:00Z",
  "deleted_at": null
}
```

**Errors**
| Code | Reason |
|------|--------|
| 400  | Empty content |
| 401  | Not authenticated |
| 403  | Not the author |
| 404  | Message not found |

---

### DELETE /api/messages/{id}

Soft-delete a message. Only the author can delete. Broadcast via WebSocket (`message.deleted`).

**Auth required:** Yes

**Response 204** (no body)

**Errors**
| Code | Reason |
|------|--------|
| 401  | Not authenticated |
| 403  | Not the author |
| 404  | Message not found |

---

## Common Error Format

All errors return JSON:
```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Access token is expired or invalid"
  }
}
```
