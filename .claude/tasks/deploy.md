# Деплой на VPS

Контекст, решения, план и прогресс. Читать целиком перед работой. Правила работы — как в
`api-v1.md`: один подпункт → стоп → пользователь проверяет и коммитит сам (Conventional Commits,
хуки lefthook активны). Ничего не коммитить.

## Схема

```
push в master ─► CI (ci.yml): tidy, golangci-lint, go test, gitleaks, govulncheck
               └► образ ghcr.io/pnz-pivo-zavod/teriyaki-sauce-service:{sha-<short>,latest}
Actions ▸ deploy (кнопка, тег) ─ssh deploy@VPS─► /opt/teriyaki: docker compose pull && up -d
    caddy (host net, :80/:443, Let's Encrypt) ─► app (127.0.0.1:8080) ─► Postgres хоста (localhost:5432)
```

## Решения

- VPS: Ubuntu/Debian amd64. Postgres уже стоит на хосте.
- Своего домена нет → бесплатный поддомен DuckDNS (`<имя>.duckdns.org` → IP VPS), Caddy получает
  на него сертификат Let's Encrypt. Переезд на свой домен — сменить `DOMAIN` в `caddy.env`.
- Работа по деплою — в ветке `cicd`, в master уйдёт одним PR (образ соберётся после merge).
- Основная ветка — `master` (не main). Плавающий тег образа — `latest`, неизменяемый — `sha-<short>`.
- Доставка: GitHub Actions → GHCR. Репо публичный → пакет GHCR сделать public (один раз руками после
  первой публикации), логин на сервере не нужен.
- CI (`ci.yml`) на PR и push в master: `go mod tidy -diff`, golangci-lint (версия как локально,
  v2.13.2), `go test`, gitleaks по всей истории, govulncheck. Образ собирается и публикуется только
  на push в master после успешных проверок. Инструменты — из `tools/go.mod`, как в хуках.
- Выкатка — только кнопкой: workflow `deploy` (workflow_dispatch, вход — тег, по умолчанию `latest`),
  GitHub Environment `production`. Копирует `deploy/compose.yml` и `deploy/Caddyfile` на сервер,
  пишет тег в `/opt/teriyaki/.env` (APP_TAG), `docker compose pull && up -d`, проверяет
  `https://<домен>/health`. Откат — deploy с предыдущим `sha-…`.
- SSH: пользователь `deploy` в группе docker, без sudo, отдельный ed25519-ключ только для CI; ключ,
  хост и known_hosts — секреты Environment `production`. Без сторонних ssh-экшенов.
- Запуск: docker compose, app + caddy, оба `network_mode: host`. App слушает `127.0.0.1:8080`.
  Логи — json-file 10MB×3.
- Секреты только на сервере: `/opt/teriyaki/app.env` (`DATABASE_URL`, `BOT_TOKEN`) и
  `caddy.env` (домен, логин/хэш basic auth), `chmod 600`. В репо — только `*.env.example`.
  Значения — в одинарных кавычках: compose подставляет `$VAR` в `env_file` (проверено — bcrypt-хэш
  без кавычек обрезается), кавычки это отключают.
- Caddy запинен на `caddy:2.11`. Пароль роли Postgres — `openssl rand -hex 24`, чтобы не
  url-кодировать.
- `/docs` и `/openapi.yaml` на проде — под basic auth в Caddy.
- Postgres: своя роль `teriyaki` (владелец БД, без суперправ) + БД `teriyaki`; ежедневный
  `pg_dump -Fc` в `/var/backups/teriyaki`, храним 7.
- Локальный стенд (`compose.yml` в корне) и прод (`deploy/compose.yml`) — разные файлы: на проде
  Postgres на хосте и Caddy, локально Postgres в контейнере и без HTTPS. Пароль Postgres в
  локальном стенде — `postgres` (в allowlist gitleaks).
- Не делаем сейчас: фронт (статика) и сужение CORS, healthcheck в compose (distroless без shell —
  проверяет deploy), внешний мониторинг, pre-push.

## План и прогресс

### Шаг 6. Деплой
- [x] 6.1 CI: `.github/workflows/ci.yml` — проверки на PR/push, сборка и публикация образа в GHCR
  на push в master (`sha-<short>`, `latest`).
- [ ] 6.2 Прод-конфиг `deploy/`: `compose.yml`, `Caddyfile`, `app.env.example`, `caddy.env.example`,
  `backup.sh`, `README.md` (bootstrap сервера: Docker, ufw 22/80/443, юзер deploy, роль/БД Postgres,
  pg_hba, бэкап, DNS).
- [ ] 6.2.1 Локальный стенд для фронта (по просьбе фронта): `compose.yml` в корне — Postgres 18 в
  контейнере + API из исходников (`build: .`), `DEV_USER_ID=1`, порт `${APP_PORT:-8080}`, раздел в README.
- [ ] 6.3 Deploy workflow: `.github/workflows/deploy.yml` (dispatch с тегом, environment production,
  scp конфига, compose up, проверка /health).
- [ ] 6.4 Первый деплой (руками по runbook): сервер, секреты Environment, пакет public, запуск deploy.
  Проверка: `/health` 200, `/v1/me` без заголовка 401, `/docs` просит пароль, дамп появился.
- [ ] 6.5 Доки: раздел «Деплой» в корневом README со ссылкой на `deploy/README.md`.

## Что нужно от пользователя (6.4)

- Поддомен DuckDNS (или свой домен) с IP VPS.
- SSH-доступ к серверу с sudo для bootstrap.
- Доступ к настройкам репо на GitHub (Environments, Packages).

## Журнал

- 2026-09-27: план согласован. Основная ветка оказалась `master` — CI на неё, тег образа `latest`.
- 2026-09-27: сделан 6.1, ждёт коммита. actionlint чистый; реальный прогон — после push. Экшены на
  последних major: checkout@v7, setup-go@v7, golangci-lint-action@v9, docker/*@v4-v7.
- 2026-09-27: 6.1 закоммичен в ветку `cicd` (запушена). Дальше работаем в `cicd`.
- 2026-09-27: сделан 6.2, ждёт коммита. Стек проверен локально (OrbStack, `DOMAIN=localhost`,
  внутренний CA Caddy): proxy, 308 http→https, basic auth на /docs и /openapi.yaml (401/401/200),
  app на 127.0.0.1. `backup.sh` — только `sh -n`, на сервере не запускался.
- 2026-09-27: 6.2.1 по просьбе фронта — локальный `compose.yml`. Проверено: smoke 82/82 против
  стенда, миграции не накатываются повторно при рестарте, данные переживают `down`/`up`.
- 2026-09-27: gitleaks блокировал коммит 6.2: своё правило postgres-url-password ловило плейсхолдер
  `<пароль>` в deploy/README.md. В allowlist добавлены `<...>`. Урок: `gitleaks dir <подпапка>` не
  видит корневой `.gitleaks.toml` — проверять с `--config .gitleaks.toml` или `git --staged`.
