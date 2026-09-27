#!/usr/bin/env bash
# Прогоняет эндпоинты против запущенного сервиса с DEV_USER_ID. Требует curl и jq.
# Можно запускать повторно: создаёт свои данные с уникальными именами.
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
BODY_FILE="$(mktemp)"
trap 'rm -f "$BODY_FILE"' EXIT

# req <method> <path> [json] — тело ответа пишет в $BODY_FILE, код печатает в stdout.
req() {
	local args=(-sS -o "$BODY_FILE" -w '%{http_code}' -X "$1" "$BASE_URL$2")
	if [[ $# -ge 3 ]]; then
		args+=(-H 'Content-Type: application/json' -d "$3")
	fi

	curl "${args[@]}"
}

# check <описание> <код> <jq-условие> <method> <path> [json]
check() {
	local name="$1" want="$2" cond="$3"
	shift 3

	local status
	status="$(req "$@")"
	if [[ "$status" != "$want" ]] || ! jq -e "$cond" "$BODY_FILE" >/dev/null; then
		echo "FAIL: $name: $status $(cat "$BODY_FILE")" >&2
		exit 1
	fi

	echo "ok: $name"
}

# last <jq-выражение> — значение из последнего ответа.
last() {
	jq -r "$1" "$BODY_FILE"
}

check "health" 200 '.data == "ok" and .error == null' GET /health
check "unknown route -> 404" 404 '.data == null and .error == "route not found"' GET /nope
check "me (DEV_USER_ID)" 200 '.data.userId > 0' GET /v1/me

# Теги
TAG="smoke-$$-$RANDOM"
check "create tag" 201 ".data.id > 0 and .data.name == \"$TAG\" and .data.color == \"#ff0000\"" \
	POST /v1/tag "{\"name\":\" $TAG \",\"color\":\"#ff0000\"}"
TAG_ID="$(last .data.id)"
check "create tag: same name -> 409" 409 '.error != null' POST /v1/tag "{\"name\":\"$TAG\"}"
check "create tag: bad color -> 400" 400 '.error == "color must be #RRGGBB"' \
	POST /v1/tag '{"name":"x","color":"red"}'
check "create tag: blank name -> 400" 400 '.error == "name is required"' POST /v1/tag '{"name":"  "}'
check "create tag: invalid json -> 400" 400 '.error != null' POST /v1/tag '{'
check "list tags" 200 "any(.data[]; .id == $TAG_ID)" GET /v1/tags
check "update tag: full replace, color cleared" 200 \
	".data.id == $TAG_ID and .data.name == \"$TAG-upd\" and .data.color == null" \
	PUT "/v1/tag/$TAG_ID" "{\"name\":\"$TAG-upd\"}"
check "delete tag" 200 '.data == null and .error == null' DELETE "/v1/tag/$TAG_ID"
check "delete tag again -> 404" 404 '.error != null' DELETE "/v1/tag/$TAG_ID"
check "update missing tag -> 404" 404 '.error != null' PUT "/v1/tag/$TAG_ID" '{"name":"x"}'
check "bad id -> 400" 400 '.error == "invalid id"' DELETE /v1/tag/abc

# Задачи
check "task: create tag for tasks" 201 '.data.id > 0' POST /v1/tag "{\"name\":\"$TAG-task\"}"
TASK_TAG_ID="$(last .data.id)"
check "create task" 201 \
	".data.id > 0 and .data.name == \"buy milk\" and .data.priority == 2 and .data.isCompleted == false
	and .data.date == \"2026-09-27T09:00:00Z\" and .data.description == null
	and .data.tags == [{id: $TASK_TAG_ID, name: \"$TAG-task\"}] and .data.notes == []" \
	POST /v1/task "{\"name\":\" buy milk \",\"date\":\"2026-09-27T12:00:00+03:00\",\"priority\":2,
	\"tagIds\":[$TASK_TAG_ID,$TASK_TAG_ID],\"isCompleted\":true}"
TASK_ID="$(last .data.id)"
check "get task" 200 ".data.id == $TASK_ID and .data.tags[0].id == $TASK_TAG_ID and .data.notes == []" \
	GET "/v1/task/$TASK_ID"
check "create task without optional fields" 201 '.data.priority == 0 and .data.tags == [] and .data.date == null' \
	POST /v1/task '{"name":"bare"}'
check "create task: unknown tag -> 400" 400 '.error == "tagIds contain unknown tags"' \
	POST /v1/task '{"name":"x","tagIds":[999999999]}'
check "create task: blank name -> 400" 400 '.error == "name is required"' POST /v1/task '{"name":" "}'
check "create task: priority 4 -> 400" 400 '.error == "priority must be 0..3"' \
	POST /v1/task '{"name":"x","priority":4}'
check "create task: bad date -> 400" 400 '.error != null' POST /v1/task '{"name":"x","date":"27.09.2026"}'
check "get missing task -> 404" 404 '.error != null' GET /v1/task/999999999
check "task tag removed on tag delete" 200 '.data == null' DELETE "/v1/tag/$TASK_TAG_ID"
check "get task after tag delete" 200 '.data.tags == []' GET "/v1/task/$TASK_ID"
