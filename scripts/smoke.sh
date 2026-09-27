#!/usr/bin/env bash
# Прогоняет эндпоинты против запущенного сервиса. Требует curl и jq.
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

# check <описание> <jq-условие> <curl-аргументы...>
check() {
	local name="$1" cond="$2"
	shift 2

	local body
	body="$(curl -sS "$@")"
	if ! jq -e "$cond" <<<"$body" >/dev/null; then
		echo "FAIL: $name: $body" >&2
		exit 1
	fi

	echo "ok: $name"
}

check "health" '.data == "ok" and .error == null' "$BASE_URL/health"
check "unknown route" '.data == null and .error == "route not found"' "$BASE_URL/nope"
check "me (DEV_USER_ID)" '.data.userId > 0' "$BASE_URL/v1/me"
