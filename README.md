# RichTalk

Web messenger — pet project. Go backend, React + Vite frontend, PostgreSQL, Redis, Nginx.

## Быстрый старт (dev)

```bash
cp .env.example .env
./scripts/dev.sh
```

Остановить всё:

```bash
./scripts/down.sh
```

## Порты в dev-режиме

| Сервис    | Хост            | Описание                        |
|-----------|-----------------|----------------------------------|
| Nginx     | http://localhost | Единая точка входа               |
| Vite      | localhost:5173  | Прямой доступ к dev-серверу      |
| PostgreSQL| localhost:5432  | Прямое подключение к БД          |
| Redis     | localhost:6379  | Прямое подключение к Redis       |

## Роутинг Nginx

- `/` → frontend (статика или Vite dev-server)
- `/api/` → api:8080
- `/ws` → api:8080 (WebSocket upgrade)

## Проверка работоспособности

```bash
# Health-check API
curl http://localhost/api/health

# Ожидаемый ответ
{"status":"ok"}
```

## Структура

```
services/   — исходники сервисов (api, frontend)
infra/      — конфиги инфраструктуры (postgres, redis, nginx)
scripts/    — вспомогательные скрипты запуска
```

Prod-like сборка: `docker compose up --build`  
Dev-сборка (hot reload): `./scripts/dev.sh`
