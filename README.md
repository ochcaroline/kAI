# kAI

A local kanban board for you and your AI agents. One Go binary serves a web UI and a JSON API on `http://127.0.0.1:7777`, and keeps all tasks in one JSON file (`~/.kai/board.json`).

## Install

    brew install ochcaroline/tap/kai
    brew services start kai          # starts at login, restarts if it crashes

Or build it yourself: `go build -o kai . && ./kai`

## Use

- Open http://127.0.0.1:7777. Drag cards between Todo, In Progress and Done. Agent changes show up within 3 seconds.
- Use another board file (e.g. synced via OneDrive): `kai -file ~/OneDrive/kai.json`, or set `KAINBAN_FILE`.
- Use another address: `kai -addr kai.localhost:8080`.

## Claude Code skill

    mkdir -p ~/.claude/skills/todo && kai skill > ~/.claude/skills/todo/SKILL.md

If the server runs with `-addr`, pass the same flag: `kai -addr host:port skill`.

- `/todo` lists tasks.
- `/todo plan 12` asks you questions, then saves a plan to task #12.
- `/todo do 12` runs the plan and marks the task done.

## API

Tasks are addressed by `{ref}`: id (`12`, `#12`) or unique title text. Statuses: `todo`, `in_progress`, `done`. Priorities: `critical`, `high`, `medium`, `low`. Writes need `Content-Type: application/json`.

    GET    /api/tasks?q=text&status=todo
    POST   /api/tasks            {"title","description","plan","priority","status"}
    GET    /api/tasks/{ref}
    PATCH  /api/tasks/{ref}      any subset of the fields, e.g. {"status":"done"}
    DELETE /api/tasks/{ref}

Only requests to `localhost`, `127.0.0.1`, `*.localhost` or the `-addr` host are accepted.
