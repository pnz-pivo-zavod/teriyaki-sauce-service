# Деплой на VPS

```
push в master ─► CI: проверки ─► образ ghcr.io/pnz-pivo-zavod/teriyaki-sauce-service:{sha-<short>,latest}
Actions ▸ deploy ─ssh deploy@VPS─► /opt/teriyaki: docker compose pull && up -d
    caddy (:80/:443, HTTPS) ─► app (127.0.0.1:8080) ─► Postgres хоста (localhost:5432)
```

| Файл | На сервере | Кто кладёт |
|---|---|---|
| `compose.yml`, `Caddyfile` | `/opt/teriyaki/` | workflow deploy при каждом деплое |
| `.env` (`APP_TAG=...`) | `/opt/teriyaki/.env` | workflow deploy |
| `app.env`, `caddy.env` | `/opt/teriyaki/` | руками, один раз (секреты) |
| `backup.sh` | `/etc/cron.daily/teriyaki-backup` | руками, один раз |

Ниже — разовая подготовка сервера (Ubuntu/Debian amd64, Postgres уже стоит). Команды — под
пользователем с sudo.

## 1. Домен

Бесплатно — [DuckDNS](https://www.duckdns.org): войти (GitHub/Google), создать поддомен
(например `teriyaki`), в поле `current ip` вписать IP VPS, `update ip`. Свой домен — A-запись на IP.

```sh
dig +short teriyaki.duckdns.org   # должен вернуть IP VPS
```

## 2. Docker

```sh
curl -fsSL https://get.docker.com | sudo sh   # Docker Engine + compose plugin
docker compose version
```

## 3. Firewall

Наружу — только SSH и HTTP(S); Postgres и сервис слушают localhost.

```sh
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 443/udp   # HTTP/3
sudo ufw enable
sudo ufw status
```

## 4. Пользователь deploy

Под ним CI заходит по ssh. В группе docker — значит фактически root на хосте, поэтому ключ
отдельный и только для CI.

```sh
sudo adduser --disabled-password --gecos '' deploy
sudo usermod -aG docker deploy
sudo install -d -o deploy -g deploy -m 700 /home/deploy/.ssh
```

Ключ генерируем **локально** (на Mac), приватная часть уйдёт в секреты GitHub (шаг 6.3):

```sh
ssh-keygen -t ed25519 -N '' -C 'github-actions teriyaki' -f ~/.ssh/teriyaki-deploy
cat ~/.ssh/teriyaki-deploy.pub   # → на сервер
```

На сервере:

```sh
echo '<содержимое teriyaki-deploy.pub>' | sudo tee /home/deploy/.ssh/authorized_keys
sudo chown deploy:deploy /home/deploy/.ssh/authorized_keys
sudo chmod 600 /home/deploy/.ssh/authorized_keys
```

Проверка с Mac: `ssh -i ~/.ssh/teriyaki-deploy deploy@<IP> docker ps`.

## 5. Postgres: роль и база

Пароль — hex, чтобы не url-кодировать в `DATABASE_URL`:

```sh
openssl rand -hex 24
sudo -u postgres psql -c "CREATE ROLE teriyaki LOGIN PASSWORD '<пароль>'"
sudo -u postgres psql -c "CREATE DATABASE teriyaki OWNER teriyaki"
```

Роль без суперправ, владелец только своей базы; таблицы создаст сам сервис миграциями при старте.
Сервис ходит по TCP на `localhost` с паролем — в `pg_hba.conf` должна быть строка (в Debian/Ubuntu
есть по умолчанию):

```
host    all    all    127.0.0.1/32    scram-sha-256
```

Проверка:

```sh
psql 'postgres://teriyaki:<пароль>@localhost:5432/teriyaki' -c 'select 1'
```

## 6. /opt/teriyaki и секреты

```sh
sudo install -d -o deploy -g deploy -m 750 /opt/teriyaki
sudo -u deploy -i
cd /opt/teriyaki
```

Под `deploy` создать два файла по образцам [`app.env.example`](app.env.example) и
[`caddy.env.example`](caddy.env.example) — значения в одинарных кавычках:

```sh
nano app.env     # DATABASE_URL (пароль из шага 5), BOT_TOKEN
nano caddy.env   # DOMAIN, DOCS_USER, DOCS_PASSWORD_HASH
chmod 600 app.env caddy.env
```

Хэш пароля для `/docs`:

```sh
docker run --rm caddy:2.11 caddy hash-password --plaintext '<пароль для фронтендера>'
```

## 7. Бэкап

```sh
scp deploy/backup.sh <user>@<IP>:   # с Mac, из корня репо
```

На сервере:

```sh
sudo install -m 755 backup.sh /etc/cron.daily/teriyaki-backup
sudo /etc/cron.daily/teriyaki-backup                             # проверить руками
sudo ls -l /var/backups/teriyaki
```

Дамп раз в сутки (`pg_dump -Fc`), храним 7 последних. Дампы лежат на том же диске — от ошибок и
кривых миграций спасают, от потери VPS нет. Скачать к себе:
`ssh <user>@<IP> sudo cat /var/backups/teriyaki/<файл>.dump > teriyaki.dump`.

Восстановление:

```sh
sudo -u postgres pg_restore --clean --if-exists -d teriyaki /var/backups/teriyaki/<файл>.dump
```

## Эксплуатация

Под `deploy` в `/opt/teriyaki`:

```sh
docker compose ps
docker compose logs -f app          # логи сервиса (ротация 10MB×3)
docker compose logs -f caddy        # выдача сертификата, ошибки прокси
docker compose restart app
cat .env                            # какой тег сейчас выкачен
```

Откат — запустить workflow deploy с предыдущим тегом `sha-<short>` (теги — в GitHub Packages).
Руками то же самое:

```sh
echo 'APP_TAG=sha-<short>' > .env && docker compose pull app && docker compose up -d
```

Проверка после деплоя:

```sh
curl https://<домен>/health              # {"data":"ok","error":null}
curl https://<домен>/v1/me               # 401 — авторизация включена
curl -u frontend:<пароль> https://<домен>/docs -o /dev/null -w '%{http_code}\n'   # 200
```
