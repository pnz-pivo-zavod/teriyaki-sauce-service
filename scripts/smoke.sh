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
BARE_ID="$(last .data.id)"
check "create task: unknown tag -> 400" 400 '.error == "tagIds contain unknown tags"' \
	POST /v1/task '{"name":"x","tagIds":[999999999]}'
check "create task: blank name -> 400" 400 '.error == "name is required"' POST /v1/task '{"name":" "}'
check "create task: priority 4 -> 400" 400 '.error == "priority must be 0..3"' \
	POST /v1/task '{"name":"x","priority":4}'
check "create task: bad date -> 400" 400 '.error != null' POST /v1/task '{"name":"x","date":"27.09.2026"}'
check "get missing task -> 404" 404 '.error != null' GET /v1/task/999999999
check "task tag removed on tag delete" 200 '.data == null' DELETE "/v1/tag/$TASK_TAG_ID"
check "get task after tag delete" 200 '.data.tags == []' GET "/v1/task/$TASK_ID"

# Список задач: t1 — 2026-10-01 10:00Z с тегом, t2 — 2026-10-02 10:00Z, t3 — без даты.
check "list: tag" 201 '.data.id > 0' POST /v1/tag "{\"name\":\"$TAG-list\"}"
LIST_TAG_ID="$(last .data.id)"
check "list: create t1" 201 '.data.id > 0' POST /v1/task \
	"{\"name\":\"t1\",\"date\":\"2026-10-01T10:00:00Z\",\"tagIds\":[$LIST_TAG_ID]}"
T1="$(last .data.id)"
check "list: create t2" 201 '.data.id > 0' POST /v1/task '{"name":"t2","date":"2026-10-02T10:00:00Z"}'
T2="$(last .data.id)"
check "list: create t3" 201 '.data.id > 0' POST /v1/task '{"name":"t3"}'
T3="$(last .data.id)"

# has <id...> / hasnt <id...> — jq-условия на наличие задач в .data.
has() { local c="true"; for id in "$@"; do c="$c and any(.data[]; .id == $id)"; done; echo "$c"; }
hasnt() { local c="true"; for id in "$@"; do c="$c and all(.data[]; .id != $id)"; done; echo "$c"; }

check "list: no filters -> all, sorted date asc, no date last" 200 \
	"$(has "$T1" "$T2" "$T3") and ([.data[].id] | index($T1) < index($T2) and index($T2) < index($T3))" \
	GET /v1/tasks
check "list: tags filled, notes []" 200 \
	"(.data[] | select(.id == $T1) | .tags[0].id == $LIST_TAG_ID and .notes == [])" GET /v1/tasks
check "list: startDate only -> one day" 200 "$(has "$T1") and $(hasnt "$T2" "$T3")" \
	GET "/v1/tasks?startDate=2026-10-01T00:00:00Z"
check "list: startDate with offset (%2B)" 200 "$(has "$T1") and $(hasnt "$T2" "$T3")" \
	GET "/v1/tasks?startDate=2026-10-01T03:00:00%2B03:00"
check "list: startDate+endDate -> [start, end)" 200 "$(has "$T1" "$T2") and $(hasnt "$T3")" \
	GET "/v1/tasks?startDate=2026-10-01T00:00:00Z&endDate=2026-10-02T10:00:01Z"
check "list: endDate exclusive" 200 "$(has "$T1") and $(hasnt "$T2" "$T3")" \
	GET "/v1/tasks?startDate=2026-10-01T00:00:00Z&endDate=2026-10-02T10:00:00Z"
check "list: endDate only" 200 "$(has "$T1") and $(hasnt "$T2" "$T3")" \
	GET "/v1/tasks?endDate=2026-10-02T00:00:00Z"
check "list: isCompleted=false" 200 "$(has "$T1" "$T2" "$T3")" GET "/v1/tasks?isCompleted=false"
check "list: isCompleted=true" 200 "$(hasnt "$T1" "$T2" "$T3")" GET "/v1/tasks?isCompleted=true"
check "list: empty result -> []" 200 '.data == []' \
	GET "/v1/tasks?startDate=1990-01-01T00:00:00Z"
check "list: bad startDate -> 400" 400 '.error != null' GET "/v1/tasks?startDate=2026-10-01"
check "list: end <= start -> 400" 400 '.error == "endDate must be after startDate"' \
	GET "/v1/tasks?startDate=2026-10-02T00:00:00Z&endDate=2026-10-01T00:00:00Z"
check "list: bad isCompleted -> 400" 400 '.error != null' GET "/v1/tasks?isCompleted=yes"
check "list: cleanup tag" 200 '.error == null' DELETE "/v1/tag/$LIST_TAG_ID"

# Изменение задач
check "update: tag" 201 '.data.id > 0' POST /v1/tag "{\"name\":\"$TAG-upd-task\"}"
UPD_TAG_ID="$(last .data.id)"
check "update: create task" 201 '.data.id > 0' POST /v1/task \
	"{\"name\":\"u\",\"description\":\"d\",\"date\":\"2026-10-01T10:00:00Z\",\"priority\":3,
	\"notifyAt\":\"2026-10-01T09:00:00Z\",\"tagIds\":[$UPD_TAG_ID]}"
UPD_ID="$(last .data.id)"
check "update: full replace clears omitted fields, sets isCompleted, ignores notes" 200 \
	".data.id == $UPD_ID and .data.name == \"u2\" and .data.description == null and .data.date == null
	and .data.notifyAt == null and .data.priority == 0 and .data.isCompleted == true
	and .data.tags == [] and .data.notes == []" \
	PUT "/v1/task/$UPD_ID" '{"name":" u2 ","isCompleted":true,"notes":[{"text":"ignored"}]}'
check "update: set tags" 200 ".data.tags[0].id == $UPD_TAG_ID and .data.isCompleted == false" \
	PUT "/v1/task/$UPD_ID" "{\"name\":\"u3\",\"tagIds\":[$UPD_TAG_ID]}"
check "update: unknown tag -> 400" 400 '.error == "tagIds contain unknown tags"' \
	PUT "/v1/task/$UPD_ID" '{"name":"broken","tagIds":[999999999]}'
check "update: rolled back after 400" 200 ".data.name == \"u3\" and .data.tags[0].id == $UPD_TAG_ID" \
	GET "/v1/task/$UPD_ID"
check "update: blank name -> 400" 400 '.error == "name is required"' PUT "/v1/task/$UPD_ID" '{"name":""}'
check "update: missing task -> 404" 404 '.error != null' PUT /v1/task/999999999 '{"name":"x"}'
check "complete" 200 ".data.id == $UPD_ID and .data.isCompleted == true and .data.name == \"u3\"" \
	PATCH "/v1/task/$UPD_ID/complete"
check "complete: idempotent" 200 '.data.isCompleted == true' PATCH "/v1/task/$UPD_ID/complete"
check "complete: shows in isCompleted=true" 200 "$(has "$UPD_ID")" GET "/v1/tasks?isCompleted=true"
check "complete: missing task -> 404" 404 '.error != null' PATCH /v1/task/999999999/complete
check "delete task" 200 '.data == null and .error == null' DELETE "/v1/task/$UPD_ID"
check "delete task: get -> 404" 404 '.error != null' GET "/v1/task/$UPD_ID"
check "delete task: again -> 404" 404 '.error != null' DELETE "/v1/task/$UPD_ID"
check "delete task: tag survives" 200 "any(.data[]; .id == $UPD_TAG_ID)" GET /v1/tags

# Заметки: к задачам T1/T2 из блока списка.
check "create note" 201 ".data.id > 0 and .data.taskId == $T1 and .data.text == \"n1\" and .data.date != null" \
	POST /v1/note "{\"taskId\":$T1,\"text\":\" n1 \"}"
NOTE_ID="$(last .data.id)"
NOTE_DATE="$(last .data.date)"
check "create note: blank text -> 400" 400 '.error == "text is required"' \
	POST /v1/note "{\"taskId\":$T1,\"text\":\" \"}"
check "create note: no taskId -> 400" 400 '.error == "taskId is required"' POST /v1/note '{"text":"x"}'
check "create note: missing task -> 404" 404 '.error != null' \
	POST /v1/note '{"taskId":999999999,"text":"x"}'
check "second note" 201 '.data.id > 0' POST /v1/note "{\"taskId\":$T1,\"text\":\"n2\"}"
NOTE2_ID="$(last .data.id)"
check "get task: notes in creation order" 200 \
	"[.data.notes[] | {id, taskId, text}] == [{id: $NOTE_ID, taskId: $T1, text: \"n1\"}, {id: $NOTE2_ID, taskId: $T1, text: \"n2\"}]" \
	GET "/v1/task/$T1"
check "list tasks: notes filled, other tasks []" 200 \
	"(.data[] | select(.id == $T1) | .notes | length == 2) and (.data[] | select(.id == $T2) | .notes == [])" \
	GET /v1/tasks
check "put task: keeps notes, ignores notes in body" 200 ".data.notes | length == 2" \
	PUT "/v1/task/$T1" '{"name":"t1","date":"2026-10-01T10:00:00Z","notes":[]}'
check "complete task: returns notes" 200 '.data.isCompleted == true and (.data.notes | length == 2)' \
	PATCH "/v1/task/$T1/complete"
check "update note: text, date unchanged" 200 \
	".data.id == $NOTE_ID and .data.taskId == $T1 and .data.text == \"n1-upd\" and .data.date == \"$NOTE_DATE\"" \
	PUT "/v1/note/$NOTE_ID" "{\"taskId\":$T1,\"text\":\"n1-upd\"}"
check "update note: move to other task" 200 ".data.taskId == $T2 and .data.text == \"moved\"" \
	PUT "/v1/note/$NOTE_ID" "{\"taskId\":$T2,\"text\":\"moved\"}"
check "update note: move to missing task -> 404" 404 '.error != null' \
	PUT "/v1/note/$NOTE_ID" '{"taskId":999999999,"text":"x"}'
check "update note: missing note -> 404" 404 '.error != null' \
	PUT /v1/note/999999999 "{\"taskId\":$T1,\"text\":\"x\"}"
check "delete note" 200 '.data == null and .error == null' DELETE "/v1/note/$NOTE_ID"
check "delete note: again -> 404" 404 '.error != null' DELETE "/v1/note/$NOTE_ID"
check "note for cascade" 201 '.data.id > 0' POST /v1/note "{\"taskId\":$T3,\"text\":\"cascade\"}"
CASCADE_NOTE_ID="$(last .data.id)"

# Уборка: задачи и теги, созданные прогоном.
for id in "$TASK_ID" "$BARE_ID" "$T1" "$T2" "$T3"; do
	check "cleanup task $id" 200 '.error == null' DELETE "/v1/task/$id"
done
check "cleanup tag" 200 '.error == null' DELETE "/v1/tag/$UPD_TAG_ID"
check "cascade: note removed with its task" 404 '.error != null' DELETE "/v1/note/$CASCADE_NOTE_ID"
