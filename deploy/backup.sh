#!/bin/sh
# Ежедневный дамп БД teriyaki, храним последние 7. Установка (имя без точки — иначе run-parts
# его пропустит):
#   sudo install -m 755 backup.sh /etc/cron.daily/teriyaki-backup
# Восстановление:
#   sudo -u postgres pg_restore --clean --if-exists -d teriyaki /var/backups/teriyaki/<файл>.dump
set -eu

dir=/var/backups/teriyaki
keep=7

install -d -o postgres -g postgres -m 700 "$dir"
runuser -u postgres -- pg_dump -Fc -f "$dir/teriyaki-$(date +%F).dump" teriyaki

# Удаляем всё, кроме $keep самых свежих. Имена генерируем сами (без пробелов) — ls безопасен.
# shellcheck disable=SC2012
ls -1t "$dir"/teriyaki-*.dump | tail -n +"$((keep + 1))" | xargs -r rm --
