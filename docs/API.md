# RichTalk — HTTP API Reference

Base URL: `http://localhost/api`  
All timestamps: RFC 3339 (`2024-01-15T10:30:00Z`)  
All IDs: UUID v4  
Content-Type: `application/json`

## Authentication

Protected endpoints require `Authorization: Bearer <access_token>`.  
Access token TTL: **15 minutes**.  
Refresh token TTL: **30 days** (rotated on each use).

---

## Auth

### POST /api/auth/register

Создаёт пользователя. Автоматически создаёт Notes-чат для нового пользователя.

**Auth required:** No

**Request**
```json
{
  "username": "alice",
  "password": "s3cr3t"
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
  "refresh_token": "<opaque_hex_token>"
}
```

**Errors**
| Code | error.code | Reason |
|------|------------|--------|
| 400  | validation_error | username/password не прошли валидацию |
| 409  | conflict | username уже занят |

---

### POST /api/auth/login

**Auth required:** No

**Request**
```json
{
  "username": "alice",
  "password": "s3cr3t"
}
```

**Response 200** — та же схема, что у register.

**Errors**
| Code | error.code | Reason |
|------|------------|--------|
| 400  | bad_request | Отсутствуют поля |
| 401  | invalid_credentials | Неверный пароль или пользователь не найден |

---

### POST /api/auth/refresh

Обменивает refresh_token на новый access_token + ротированный refresh_token.  
Старый refresh_token становится недействительным сразу после вызова.

**Auth required:** No

**Request**
```json
{
  "refresh_token": "<opaque_hex_token>"
}
```

**Response 200**
```json
{
  "access_token": "<new_jwt>",
  "refresh_token": "<new_opaque_hex_token>"
}
```

**Errors**
| Code | error.code | Reason |
|------|------------|--------|
| 401  | invalid_refresh_token | Токен истёк, не существует или уже ротирован |

---

### POST /api/auth/logout

Удаляет refresh_token из БД.

**Auth required:** Yes

**Request**
```json
{
  "refresh_token": "<opaque_hex_token>"
}
```

**Response 204** (no body)

---

## Users

### GET /api/users/me

**Auth required:** Yes

**Response 200**
```json
{
  "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "username": "alice",
  "created_at": "2024-01-15T10:00:00Z"
}
```

---

### GET /api/users/search?q=alice

Поиск пользователей по префиксу username.

**Auth required:** Yes

**Query params**
| Param | Type | Required | Notes |
|-------|------|----------|-------|
| q | string | Yes | Минимум 2 символа |

**Response 200**
```json
{
  "users": [
    { "id": "f47ac10b-...", "username": "alice" },
    { "id": "b3d1c2e4-...", "username": "alice_smith" }
  ]
}
```

**Errors**
| Code | Reason |
|------|--------|
| 400  | q короче 2 символов |

---

## Chats

### GET /api/chats

Список всех чатов пользователя. Notes-чат всегда идёт первым.

**Auth required:** Yes

**Response 200**
```json
{
  "chats": [
    {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "type": "notes",
      "name": null,
      "other_user": null,
      "last_message": null,
      "created_at": "2024-01-15T09:00:00Z"
    },
    {
      "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
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
        "created_at": "2024-01-15T10:29:00Z",
        "deleted": false
      },
      "created_at": "2024-01-15T09:00:00Z"
    }
  ]
}
```

> `last_message.content` равен `""` если `deleted: true`.

---

### GET /api/chats/notes

Возвращает Notes-чат текущего пользователя.  
**Маршрут должен быть объявлен до `/chats/{chatID}`** (иначе chi перехватит `notes` как UUID).

**Auth required:** Yes

**Response 200** — объект чата (та же схема, что элемент в списке выше).

**Errors**
| Code | Reason |
|------|--------|
| 404  | Notes-чат не был создан при регистрации (нештатная ситуация) |

---

### POST /api/chats/direct

Создаёт или возвращает существующий direct-чат с другим пользователем. **Идемпотентен.**

**Auth required:** Yes

**Request**
```json
{
  "user_id": "b3d1c2e4-6f8a-4b2c-9d0e-1f2a3b4c5d6e"
}
```

**Response 200**
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "type": "direct",
  "name": null,
  "other_user": {
    "id": "b3d1c2e4-6f8a-4b2c-9d0e-1f2a3b4c5d6e",
    "username": "bob"
  },
  "last_message": null,
  "created_at": "2024-01-15T09:00:00Z"
}
```

**Errors**
| Code | error.code | Reason |
|------|------------|--------|
| 400  | bad_request | user_id отсутствует или некорректный UUID |
| 400  | self_chat | user_id совпадает с вызывающим |
| 404  | user_not_found | Целевой пользователь не найден |

---

### GET /api/chats/{chatID}

**Auth required:** Yes (вызывающий должен быть членом чата)

**Response 200** — объект чата (та же схема).

**Errors**
| Code | Reason |
|------|--------|
| 403  | Не является членом чата |
| 404  | Чат не найден |

---

### GET /api/chats/{chatID}/messages

Пагинированная история сообщений. Курсорная пагинация по `created_at`, ответ в порядке **от старых к новым**.

**Auth required:** Yes

**Query params**
| Param | Type | Required | Default | Notes |
|-------|------|----------|---------|-------|
| before | RFC 3339 datetime | No | now() | Вернуть сообщения с `created_at < before` |
| limit | integer | No | 50 | Диапазон 1–100 |

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
      "deleted": false,
      "attachment_type": null,
      "created_at": "2024-01-15T10:29:00Z",
      "updated_at": "2024-01-15T10:29:00Z"
    }
  ],
  "has_more": true,
  "next_cursor": "2024-01-15T10:29:00.000000000Z"
}
```

**Пагинация:** передать `next_cursor` как `before` в следующем запросе. Остановиться при `has_more: false`.

**Удалённые сообщения** включены в список: `"deleted": true`, `"content": ""`. Клиент показывает плейсхолдер.

**Поля сообщения**

| Поле | Тип | Описание |
|------|-----|----------|
| `deleted` | bool | Мягкое удаление; при true `content` = `""` |
| `attachment_type` | string\|null | Тип вложения (`"image"`, `"file"` и т.д.) — зарезервировано для будущего |
| `attachment_url` | string\|null | URL файла — зарезервировано |
| `attachment_name` | string\|null | Имя файла — зарезервировано |
| `attachment_size` | int64\|null | Размер в байтах — зарезервировано |

**Errors**
| Code | Reason |
|------|--------|
| 400  | Некорректные before/limit |
| 403  | Не является членом чата |
| 404  | Чат не найден |

---

## Messages

### POST /api/chats/{chatID}/messages

Отправляет сообщение. После сохранения в БД публикует WS-событие `message.new` всем членам чата через Redis.

**Auth required:** Yes

**Request**
```json
{
  "content": "Hello, Bob!"
}
```

**Response 201** — объект сообщения (та же схема, что в списке выше).

**Errors**
| Code | Reason |
|------|--------|
| 400  | Пустой content |
| 403  | Не является членом чата |
| 404  | Чат не найден |

---

### PATCH /api/messages/{messageID}

Редактирует сообщение. Только автор. Публикует `message.edited`.

**Auth required:** Yes

**Request**
```json
{
  "content": "Hello, Bob! (edited)"
}
```

**Response 200** — обновлённый объект сообщения.

**Errors**
| Code | Reason |
|------|--------|
| 400  | Пустой content |
| 403  | Не является автором |
| 404  | Сообщение не найдено |

---

### DELETE /api/messages/{messageID}

Мягкое удаление. Только автор. Публикует `message.deleted`.

**Auth required:** Yes

**Response 204** (no body)

**Errors**
| Code | Reason |
|------|--------|
| 403  | Не является автором |
| 404  | Сообщение не найдено |

---

## Общий формат ошибок

```json
{
  "error": {
    "code": "invalid_credentials",
    "message": "Неверные имя пользователя или пароль"
  }
}
```

Ошибки валидации:
```json
{
  "errors": [
    { "field": "username", "message": "Минимальная длина — 3 символа" }
  ]
}
```

---

## Health check

### GET /api/health

Проверяет соединения с Postgres и Redis.

**Auth required:** No

**Response 200**
```json
{ "status": "ok", "postgres": "ok", "redis": "ok" }
```
