# teriyaki-sauce-service

Бэкенд таск-трекера для Telegram Mini App: задачи, теги, заметки. Go, chi, Postgres.

Документация API для фронта — [Confluence](https://practiceilya.atlassian.net/wiki/spaces/~7120207c4bac83d9994bb493c9716a0b74c5f6/pages/425986/API),
синхронизирована с реализацией. Ниже — то же подробнее; что поменялось относительно первой версии
доки — в [отдельном разделе](#изменения-относительно-первой-версии-confluence).

## Локальный стенд в Docker (для фронта)

Нужен только Docker — Go и Postgres ставить не надо:

```sh
docker compose up --build    # Postgres + API: http://localhost:8080, документация: /docs
docker compose down          # остановить, данные сохранятся
docker compose down -v       # остановить и стереть базу
```

Авторизация отключена (`DEV_USER_ID=1`): заголовок `Authorization` и Telegram не нужны, все запросы
идут от пользователя 1. CORS открыт — фронт с dev-сервера (например `localhost:5173`) ходит в API
напрямую. Занят порт 8080 — `APP_PORT=8081 docker compose up --build`. После `git pull` —
снова `--build`, чтобы собрать свежий API.

## Запуск

Нужны Go 1.27 и Postgres.

```sh
cp .env.example .env   # заполнить DATABASE_URL и BOT_TOKEN или DEV_USER_ID
make run               # go run ./cmd, переменные берутся из .env
make build             # бинарник в bin/teriyaki-sauce-service
```

| Переменная | Обязательна | Описание |
|---|---|---|
| `HTTP_ADDR` | нет | Адрес сервера, по умолчанию `:8080` |
| `DATABASE_URL` | да | Строка подключения к Postgres |
| `BOT_TOKEN` | да, если нет `DEV_USER_ID` | Токен бота — им проверяется подпись initData |
| `DEV_USER_ID` | нет | **Только для разработки.** Отключает проверку initData: все запросы идут от этого Telegram ID |

**Миграции** накатываются автоматически при каждом старте (goose,
`internal/repository/migrations`) на базу из `DATABASE_URL`. Дайте сервису отдельную базу или хотя бы
отдельного пользователя, чтобы его таблицы (`tasks`, `tags`, `task_tags`, `notes`,
`goose_db_version`) не смешивались с чужими.

**Smoke-тест** — прогон всех эндпоинтов curl'ом против запущенного сервиса с `DEV_USER_ID`
(нужен `jq`). Убирает за собой всё, что создал:

```sh
make smoke                                  # против http://localhost:8080
BASE_URL=http://host:8080 ./scripts/smoke.sh
```

## Git-хуки

После клонирования: `make hooks` (ставит [lefthook](https://lefthook.dev) из `tools/go.mod`).
Нужен установленный `golangci-lint`, остальные инструменты запинены в `tools/go.mod`.

**pre-commit** — сначала `goimports` на застейдженных .go (исправления попадают в коммит), затем
параллельно:

| Проверка | Когда |
|---|---|
| gitleaks — секреты в застейдженном (`.gitleaks.toml`: + токен бота, пароль в postgres-URL) | всегда |
| `golangci-lint` на дифф (`.golangci.yml`) | .go |
| `go mod tidy -diff` | .go, go.mod/go.sum |
| `go test ./...` (в т.ч. сверка openapi.yaml с роутами) | .go, .sql, go.mod/go.sum, api/ |
| govulncheck | .go, go.mod/go.sum |

**commit-msg** — [Conventional Commits](https://www.conventionalcommits.org/ru/):
`<type>(<scope>)?: <описание>`, например `feat(tasks): фильтр по isCompleted`. Типы:
feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert. Merge/Revert от git пропускаются.

Пропустить хуки разово: `LEFTHOOK=0 git commit ...`.

## OpenAPI и Scalar

- `api/openapi.yaml` — спецификация OpenAPI 3.0, вшита в бинарник.
- `GET /openapi.yaml` — отдаёт её (без авторизации): для генерации клиентов.
- `GET /docs` — интерактивная документация [Scalar](https://scalar.com): описание роутов и
  отправка запросов из браузера. Для `/v1/*` в Scalar в Authentication указать
  `tma <initData>`; на dev-стенде с `DEV_USER_ID` заголовок не нужен.

Генерация типов на фронте, например:

```sh
npx openapi-typescript http://localhost:8080/openapi.yaml -o src/api/schema.d.ts
```

Спецификация поддерживается руками. `go test ./...` проверяет, что она валидна и описывает
ровно те роуты, что есть в роутере: добавили роут и не описали (или наоборот) — тест падает.

## Docker

Образ: сборка в `golang:1.27`, запуск в `gcr.io/distroless/static-debian12:nonroot`
(без shell, от непривилегированного пользователя). Миграции и SQL вшиты в бинарник.

```sh
make docker                           # docker build -t teriyaki-sauce-service .
```

Postgres на том же сервере — проще всего запустить контейнер в сети хоста, тогда `localhost`
в `DATABASE_URL` указывает на хостовый Postgres:

```sh
docker run -d --name teriyaki-sauce-service --restart unless-stopped \
  --network host \
  -e HTTP_ADDR=:8080 \
  -e DATABASE_URL='postgres://user:pass@localhost:5432/teriyaki?sslmode=disable' \
  -e BOT_TOKEN='123456:ABC...' \
  teriyaki-sauce-service
```

Или через файл: `--env-file /path/to/prod.env`. `DEV_USER_ID` на проде не задавать.

Без `--network host` (`-p 8080:8080`) хост из контейнера доступен как `host.docker.internal`
при `--add-host=host.docker.internal:host-gateway`, но тогда Postgres должен слушать docker-интерфейс
(`listen_addresses`) и пускать подсеть docker в `pg_hba.conf`.

Собрать на Mac (arm64) образ для сервера на amd64: `docker build --platform linux/amd64 -t teriyaki-sauce-service .`,
перенести без registry: `docker save teriyaki-sauce-service | ssh server docker load`.

## Авторизация

Все роуты `/v1/*` требуют заголовок с `initData` из Telegram Mini App:

```
Authorization: tma <window.Telegram.WebApp.initData>
```

Сервис проверяет подпись initData токеном бота и что `auth_date` не старше 24 часов, берёт
Telegram ID из поля `user`. Данные у каждого пользователя свои; чужая запись → 404.
Регистрации нет: любой пользователь Telegram, открывший Mini App, получает свой пустой список.
Ошибка авторизации → 401 с причиной в `error`.

`GET /v1/me` → `{"userId": 123}` — проверить, что авторизация с фронта работает.

## Формат ответа

Ответы API, включая ошибки и неизвестные роуты, — JSON-обёртка:

```json
{"data": ..., "error": null}
{"data": null, "error": "task 5 not found"}
```

| Код | Когда |
|---|---|
| 200 | Успех. `DELETE` тоже отвечает 200 с `{"data": null, "error": null}` |
| 201 | Создание (`POST`) |
| 400 | Невалидный запрос: JSON, параметры, поля. Причина — в `error` |
| 401 | Нет или невалидный `Authorization` |
| 404 | Нет записи или она чужая; неизвестный роут |
| 405 | Неподдерживаемый метод |
| 409 | Тег с таким именем уже есть |
| 500 | Внутренняя ошибка, детали только в логах сервиса |

Время — RFC3339 (`2026-10-01T10:00:00Z` или `2026-10-01T13:00:00+03:00`), в ответах всегда UTC.
Необязательные пустые поля в ответе опускаются. `tags` и `notes` — всегда массивы.
CORS открыт для любого origin.

## API

### Теги

Тег: `{"id": 1, "name": "work", "color": "#ff0000"}`. `color` необязателен.

| Метод и роут | Тело | Ответ |
|---|---|---|
| `POST /v1/tag` | `{name, color?}` | 201, тег |
| `GET /v1/tags` | — | теги пользователя по имени |
| `PUT /v1/tag/{id}` | `{name, color?}` | тег. Полная замена: без `color` цвет сбрасывается |
| `DELETE /v1/tag/{id}` | — | `null`. Тег снимается со всех задач |

- `name` обязателен, пробелы по краям обрезаются; уникален у пользователя, иначе 409.
- `color` — строго `#RRGGBB`.

### Задачи

Задача в ответе:

```json
{
  "id": 1,
  "name": "Купить молоко",
  "description": "2 литра",
  "date": "2026-10-01T10:00:00Z",
  "notifyAt": "2026-10-01T09:00:00Z",
  "priority": 2,
  "isCompleted": false,
  "tags": [{"id": 1, "name": "home", "color": "#00ff00"}],
  "notes": [{"id": 1, "taskId": 1, "text": "обезжиренное", "date": "2026-09-27T08:00:00Z"}]
}
```

Тело запроса (`POST`, `PUT`):

```json
{
  "name": "Купить молоко",
  "description": "2 литра",
  "date": "2026-10-01T10:00:00Z",
  "notifyAt": "2026-10-01T09:00:00Z",
  "priority": 2,
  "tagIds": [1],
  "isCompleted": false
}
```

| Метод и роут | Ответ |
|---|---|
| `POST /v1/task` | 201, задача. `isCompleted` игнорируется — новая задача не завершена |
| `GET /v1/task/{id}` | задача |
| `GET /v1/tasks` | список задач, фильтры ниже |
| `PUT /v1/task/{id}` | задача. **Полная замена**, см. ниже |
| `PATCH /v1/task/{id}/complete` | задача с `isCompleted: true`. Повторный вызов безопасен |
| `DELETE /v1/task/{id}` | `null`. Заметки задачи удаляются вместе с ней |

- `name` обязателен, пробелы по краям обрезаются. `priority`: 0 — нет, 1 — low, 2 — medium,
  3 — high; по умолчанию 0, в ответе есть всегда.
- `tagIds` — ID своих тегов, дубли схлопываются. Чужой/несуществующий тег → 400
  `tagIds contain unknown tags`, задача не создаётся и не меняется.
- **PUT заменяет задачу целиком**: поле, которого нет в запросе, очищается (`description`, `date`,
  `notifyAt`, `tagIds`), `priority` → 0, `isCompleted` → `false`. Чтобы поменять одно поле,
  отправьте всю задачу. `notes` в теле игнорируются — заметки меняются через `/v1/note`.
- Снять отметку о завершении — `PUT` с `"isCompleted": false`.

**Фильтры `GET /v1/tasks`** (все необязательны, комбинируются):

| Параметр | Значение |
|---|---|
| `startDate` | RFC3339. Без `endDate` — ровно сутки: `[startDate, startDate + 24h)` |
| `endDate` | RFC3339, не включается: `date < endDate` |
| `isCompleted` | `true` / `false` |

- Задачи без `date` при фильтре по дате не попадают.
- `endDate` не позже `startDate` → 400.
- `+` в смещении нужно url-кодировать: `startDate=2026-10-01T00:00:00%2B03:00`
  (`URLSearchParams` делает это сам). Иначе `+` превращается в пробел → 400.
- «Задачи на день пользователя» — `startDate` = полночь в его часовом поясе, например
  `2026-10-01T00:00:00+03:00`.
- Сортировка: по `date` (без даты — в конце), затем по `id`.

### Заметки

Заметка: `{"id": 1, "taskId": 1, "text": "...", "date": "2026-09-27T08:00:00Z"}`,
`date` — время создания, при редактировании не меняется. В задаче заметки идут по `date`.

| Метод и роут | Тело | Ответ |
|---|---|---|
| `POST /v1/note` | `{taskId, text}` | 201, заметка |
| `PUT /v1/note/{id}` | `{taskId, text}` | заметка. Другой `taskId` переносит заметку в эту задачу |
| `DELETE /v1/note/{id}` | — | `null` |

- `taskId` и непустой `text` обязательны в обоих запросах.
- Задача не найдена или чужая → 404. В `PUT` при 404 не уточняется, чего нет — заметки или
  задачи: `note 5 or task 7 not found`.

## Изменения относительно первой версии Confluence

- Ответы обёрнуты в `{"data", "error"}`, коды ответов описаны выше.
- `id` задачи есть во всех ответах (в Confluence местами пропущен).
- Теги — объекты `{id, name, color}` с отдельным CRUD (`/v1/tag`, `/v1/tags`) — в Confluence его нет.
  В запросе задачи вместо `tags` передаются `tagIds`.
- Заметки задачи — поле `notes`.
- Фильтр `isCompleted` (в Confluence опечатка `isComleted`), даты фильтров — RFC3339,
  `endDate` не включается.
- Добавлены `GET /health` (без авторизации) и `GET /v1/me`.
- Авторизация через `Authorization: tma <initData>` (в Confluence не описана).
- `notifyAt` только хранится — уведомления в Telegram пока не отправляются.

## Структура

```
cmd/main.go                    точка входа
internal/app                   конфиг из env, сборка зависимостей, старт
internal/auth                  проверка initData, middleware, UserID(ctx)
internal/model                 модели и доменные ошибки
internal/api/rest              HTTP-сервер, router (chi, CORS, логи), handler, response
internal/service               валидация и бизнес-логика
internal/repository            Postgres (pgx): запросы в queries/*.sql, миграции в migrations/
scripts/smoke.sh               smoke-тест API
```

Слои разделены по предметным областям: `TagHandler` → `TagService` → `TagRepository`, так же
`Task*` и `Note*`. План и решения по реализации — `.claude/tasks/api-v1.md`.
