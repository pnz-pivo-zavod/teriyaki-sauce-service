# API v1 — таск-трекер (Telegram miniapp)

Контекст, решения, план и прогресс реализации. Читать целиком перед работой.

## Материалы

- Документация API (Confluence): https://practiceilya.atlassian.net/wiki/spaces/~7120207c4bac83d9994bb493c9716a0b74c5f6/pages/425986/API
  (page id `425986`). В Jira не ходим.
- Шаблон структуры: https://github.com/golang-standards/project-layout/blob/master/README_ru.md
- Библиотеки: chi (роутер), telego (только валидация initData: `telegoutil.ValidateWebAppData`),
  lo (slice/map там, где нет в stdlib slices/maps), zerolog (логи), pgx/v5 (Postgres),
  goose (миграции).
- Кодстайл: скиллы `go-code-style`, `go-tests`.

## Принципы

- Максимально просто для быстрой интеграции с фронтом: без graceful shutdown и прочих обвязок.
- Слои handler → service → repository на конкретных типах, без интерфейсов.
- Каждый слой разделён структурами по предметным областям: `TagHandler` → `TagService` →
  `TagRepository` (дальше `Task*`, `Note*`), конструкторы `NewTagHandler` и т.п. Методы без
  повторения домена: `Create`, `List`, `Get`, `Update`, `Delete`. Кросс-доменные зависимости
  передаются явно в конструктор. `Health`/`Me` — функции пакета handler.
- Работаем по одному подпункту: сделал → остановился → пользователь проверяет и коммитит сам.
  Ничего не коммитить. После коммита отметить подпункт `[x]` ниже.

## Решения

### Стек и инфраструктура

- Go 1.27, модуль `teriyaki-sauce-service`, entry point `cmd/main.go`.
- Postgres уже развёрнут на сервере деплоя — docker-compose нет. Подключение только через
  `DATABASE_URL`. Локально — brew Postgres (перед запуском/созданием БД спросить пользователя).
- Миграции goose, вшиты через `//go:embed`, `goose.Up` на старте сервиса.
- Весь SQL — в `.sql` файлах `internal/repository/queries/<сущность>/<действие>.sql`, грузится
  через `//go:embed` в пакетные `string`-переменные (`_tagCreateSQL`). Инлайн-SQL в Go нет.
- Конфиг из env: `HTTP_ADDR` (default `:8080`), `DATABASE_URL`, `BOT_TOKEN`, `DEV_USER_ID`.
  Makefile подхватывает `.env` (в .gitignore), есть `.env.example`.
- timestamptz читаются из БД в UTC (кодек в `repository.NewPool`), ответы не зависят от
  таймзоны сервера.
- Деплой — Docker-образ на distroless `static-debian12:nonroot` (UID 65532, без shell).
  Postgres на хосте → контейнер в `--network host`.
- OpenAPI-спецификация `api/openapi.yaml` — источник для генерации клиентов фронтом. Меняешь роут или
  контракт → правишь спеку в том же подпункте (тест `TestOpenAPIMatchesRouter` ловит расхождение
  роутов, но не полей). Scalar запинен на версию (`@scalar/api-reference@1.72.1`).
- Git-хуки — lefthook, состав выбран пользователем: pre-commit = goimports (фикс) → gitleaks,
  golangci-lint на дифф, go mod tidy -diff, go test, govulncheck; commit-msg = Conventional
  Commits; pre-push нет. Dev-инструменты — отдельный модуль `tools/go.mod`, чтобы не засорять
  go.mod сервиса; golangci-lint — системный (не через go tool, так рекомендуют авторы).
- CORS allow-all (минимальный middleware, разрешён заголовок `Authorization`).

### Авторизация

- Заголовок `Authorization: tma <initData>`, проверка `telegoutil.ValidateWebAppData(BOT_TOKEN, …)`
  + `auth_date` не старше 24ч. User ID из поля `user` (JSON) кладётся в context.
- `GET /v1/me` → `{userId}`: проверка авторизации с фронта (добавлен в 1.4; в chi middleware
  группы не срабатывает, пока в ней нет роутов).
- Без `BOT_TOKEN` и `DEV_USER_ID` сервис не стартует. Ошибка валидации → 401 с текстом причины.
- `DEV_USER_ID` задан → проверка пропускается, используется этот ID (разработка вне Telegram).
- Таблицы users нет: `user_id BIGINT` = Telegram ID. Чужой ресурс → 404.

### Формат ответов

- Успех: `{"data": ..., "error": null}`. Ошибка: `{"data": null, "error": "текст"}`.
- Коды: 200, 201 (создание), 400, 401, 404, 409, 500. DELETE → 200 с обёрткой (не 204).
- Время — RFC3339. Опциональные поля отсутствуют в ответе, если пусты.
  `tags` и `notes` — всегда массивы (`[]`, не `null`).

### Задачи

- `id` во всех ответах (в доке местами пропущен).
- Запрос: `name` (обязателен), `description?`, `date?`, `notifyAt?`, `priority?` (0–3, default 0),
  `tagIds?: []int`, `isCompleted` (в PUT). Ответ: `tags: [{id, name, color}]`, `notes: [...]`.
- POST: `isCompleted` из запроса игнорируется (новая задача всегда не завершена); дубли в
  `tagIds` схлопываются; чужой/несуществующий тег → 400 `tagIds contain unknown tags`.
- `priority` в ответе всегда есть (0 = No), не опускается.
- PUT — полная замена (непереданное поле очищается), включая `tagIds` и `isCompleted`;
  `notes` в запросе игнорируются.
- PATCH `/complete` → `isCompleted=true`, идемпотентно, возвращает задачу целиком. Снять — через PUT.
- PUT при ошибке в `tagIds` откатывается целиком (задача не меняется).
- DELETE — каскадно удаляет заметки.
- `GET /v1/tasks`: фильтры `startDate`, `endDate` (RFC3339), `isCompleted` (опечатка `isComleted`
  в доке исправлена).
  - только `startDate` → date ∈ `[startDate, startDate+24h)`;
  - оба → `[startDate, endDate)`;
  - только `endDate` → date `< endDate`;
  - задачи без `date` при дата-фильтре не попадают;
  - сортировка: date (null в конце), id.
  - `endDate <= startDate` → 400. Невалидные параметры → 400. `+` в смещении фронт должен
    url-кодировать (`%2B`), иначе он превращается в пробел — сообщение об ошибке это подсказывает.
- Уведомления по `notifyAt` не отправляем — только храним.

### Теги (нет в доке — добавить в README)

- `{id, name, color?}`, color `#RRGGBB`, `name` уникален в пределах пользователя → 409.
- Роуты: `POST /v1/tag`, `GET /v1/tags`, `PUT /v1/tag/{id}`, `DELETE /v1/tag/{id}`.
- Удаление тега снимает его со всех задач (FK cascade в `task_tags`).

### Заметки

- `date` — время создания, при редактировании не меняется.
- В ответах задач `notes` отсортированы по `date`, затем `id`.
- PUT `/v1/note/{id}` с другим `taskId` переносит заметку (задача должна быть того же юзера).
- Владелец заметки — владелец её задачи (своего `user_id` у notes нет, проверка через JOIN).
- `taskId` обязателен и в POST, и в PUT; `text` тримится, пустой → 400.
- Задача из `taskId` не найдена/чужая → 404 (`task 7 not found`). В PUT не различаем, чего нет —
  заметки или целевой задачи: `note 5 or task 7 not found`.

### Ошибки

- Доменные ошибки в `internal/model`: `ErrNotFound` → 404, `ErrConflict` → 409,
  `*ValidationError` → 400. Текст ошибки уходит клиенту (`tag 5 not found`). Прочее → 500
  `internal error` + лог.
- Строковые поля (`name`) тримятся перед валидацией и сохранением.

### Тесты

- Только юнит-тест на валидацию initData. Остальное — `scripts/smoke.sh` (curl + jq)
  против запущенного сервиса с `DEV_USER_ID`; дополняется на каждом шаге.

## Структура

```
cmd/main.go
internal/
  app/app.go                     env, логгер, pgxpool, goose.Up, сборка слоёв, старт
  auth/auth.go, auth_test.go     валидация initData, middleware, UserID(ctx)
  model/model.go                 Task/Note/Tag, входные структуры, доменные ошибки
  api/rest/rest.go               http.Server
  api/rest/router/router.go      chi, Recoverer, лог запросов, CORS, auth, роуты
  api/rest/response/response.go обёртка {data, error}
  api/rest/handler/              handler.go (хелперы, Health, Me), tag.go, task.go, note.go
  service/                       tag.go, task.go, note.go
  repository/                    repository.go (Migrate, хелперы), tag.go, task.go, note.go
  repository/migrations/00001_init.sql
  repository/queries/{tag,task,note}/*.sql  запросы, грузятся через //go:embed
scripts/smoke.sh
Makefile, .env.example, .gitignore, README.md
```

## План и прогресс

Каждый подпункт — отдельный коммит.

### Шаг 1. Каркас
- [x] 1.1 Контекст для агентов: этот файл + `CLAUDE.md`.
- [x] 1.2 HTTP-каркас: `cmd/main.go` (удалить корневой `main.go`), app, rest, router (Recoverer,
  лог zerolog, CORS), response, `GET /health`; Makefile (`run`, `build`, `lint`), `.gitignore`,
  `.env.example`, `scripts/smoke.sh`. Проверка: `curl /health`.
- [x] 1.3 Postgres: pgxpool, goose + `00001_init.sql` (tasks, tags, task_tags, notes), миграции на
  старте. Проверка: сервис стартует с `DATABASE_URL`, таблицы созданы.
- [x] 1.4 Auth: `internal/auth`, подключение на `/v1`, юнит-тест. Проверка: `go test ./...`,
  без заголовка → 401.

### Шаг 2. Теги
- [x] 2.1 CRUD тегов: модель, repository, service (валидация, 409), handlers, роуты, smoke.

### Шаг 3. Задачи
- [x] 3.1 `POST /v1/task`, `GET /v1/task/{id}`: task + task_tags в транзакции, tagIds
  принадлежат юзеру, ответ с tags и `notes: []`.
- [x] 3.2 `GET /v1/tasks`: фильтры, пакетная подгрузка тегов (`ANY($1)`, без N+1).
- [x] 3.3 `PUT /v1/task/{id}`, `PATCH /v1/task/{id}/complete`, `DELETE /v1/task/{id}`.

### Шаг 4. Заметки
- [x] 4.1 `POST /v1/note`, `PUT /v1/note/{id}` (с переносом), `DELETE /v1/note/{id}`.
- [x] 4.2 Заметки во всех ответах задач (пакетно).

### Шаг 5. Финал
- [x] 5.1 README: запуск, env, `DATABASE_URL` (миграции на старте → отдельная БД/юзер),
  auth для фронта, отличия от Confluence-доки (обёртка, id везде, tagIds, API тегов, фильтры).
- [x] 5.2 Docker: multi-stage `Dockerfile` (golang:1.27 → distroless static nonroot),
  `.dockerignore`, `make docker`, раздел в README (запуск с `--network host` к Postgres хоста).
- [x] 5.3 OpenAPI + Scalar: `api/openapi.yaml` (OpenAPI 3.0, руками) и `api/docs.html` (Scalar с CDN),
  вшиты пакетом `api`, роуты `/openapi.yaml` и `/docs` без авторизации; тест в router сверяет
  спеку с роутами chi и валидирует её (kin-openapi).
- [ ] 5.4 Git-хуки (lefthook): `lefthook.yml`, `.golangci.yml` (standard + bodyclose, errname,
  errorlint, gosec, lll 180), `.gitleaks.toml`, `tools/go.mod` (lefthook, gitleaks, govulncheck,
  goimports через `go tool -modfile`), `make hooks`, раздел в README.

## Журнал

- 2026-09-26: план согласован. 1.1 закоммичен.
- 2026-09-26: 1.2 закоммичен. Неизвестный роут/метод → 404/405 тоже в обёртке.
- 2026-09-26: 1.3 закоммичен. Локальная БД: brew `postgresql@18`, база `teriyaki`
  (`DATABASE_URL=postgres://localhost:5432/teriyaki?sslmode=disable`, юзер ОС без пароля).
- 2026-09-27: 1.4 закоммичен. Добавлен `GET /v1/me`.
- 2026-09-27: 2.1 закоммичен (+ SQL в .sql, слои разделены по доменам). smoke.sh переписан: проверяет код ответа + jq,
  идемпотентен (уникальные имена).
- 2026-09-27: 3.1 закоммичен. Задача+теги в одной транзакции, теги задач грузятся
  пакетно (`TagRepository.ListByTaskIDs`), `notes: []` до 4.2.
- 2026-09-27: 3.2 закоммичен.
- 2026-09-27: 3.3 закоммичен. smoke теперь убирает за собой все созданные задачи и
  теги; dev-БД очищена от накопленных данных user 1.
- 2026-09-27: 4.1 закоммичен.
- 2026-09-27: 4.2 закоммичен. `TaskService.fill` грузит теги и заметки двумя
  пакетными запросами на любой список задач.
- 2026-09-27: 5.1 закоммичен.
- 2026-09-27: добавлен 5.2 (Docker) по запросу. Образ 14.8 MB, smoke против контейнера (OrbStack,
  `--network host`) прошёл. Кросс-сборка `--platform linux/amd64` не проверялась.
- 2026-09-27: страница API в Confluence (425986) обновлена до v6 под реализацию: разделы «Общее»
  (авторизация, обёртка, коды, время, модели) и «Теги», уточнения по фильтрам/PUT/заметкам.
  Inline-комментарий на «Создать таску» сохранён. README ссылается на неё как на актуальную.
- 2026-09-27: 5.3 по запросу — OpenAPI + Scalar. Спека проходит kin-openapi, Redocly lint
  (2 warning: нет license, у /health нет 4xx) и генерацию openapi-typescript. Рендер Scalar в
  браузере не проверялся.
- 2026-09-27: 5.3 закоммичен. Сделан 5.4 (хуки), ждёт коммита. Хуки уже установлены в .git/hooks —
  коммит 5.4 сам пройдёт через них, сообщение нужно в формате Conventional Commits.
