---
name: todo
description: Work with the user's local kanban board (kAI API on 127.0.0.1:7777). Use when the user says "todo", refers to a task by id (#12) or title on the board, or asks to list, add, plan, start, finish or update tasks.
---

Board API: `http://127.0.0.1:7777/api`. Use `curl -s`; writes need `-H 'Content-Type: application/json'`. If the connection fails, tell the user the server is not running (`brew services start kai`, or run `kai`) and stop.

Task: `id`, `title`, `description`, `plan`, `priority` (critical|high|medium|low), `status` (todo|in_progress|done). `{ref}` is an id (`12`, `#12`) or unique title text; 409 means ambiguous, so list and ask.

The task's `plan` field on the API is the only place plans live. Never keep a plan only in chat, a plan file or plan mode; always save it with `PATCH /tasks/{ref}` `{"plan": "..."}`.

Interpret the argument:

- **none / "list"**: `GET /tasks`, show grouped by status, as `#id [priority] title`.
- **text or id only**: `GET /tasks/{ref}` and summarize it including the plan.
- **"add ..."**: `POST /tasks` with title, and priority/description if given (default medium, status todo). Confirm the new id.
- **"plan <ref>"** (e.g. `/todo plan 12`): read the task and the code it touches, then draft a concrete plan. Before finishing, ask the user the open questions (ambiguous scope, multiple interpretations, tradeoffs, missing details) and wait for answers. Only then write the final plan, `PATCH {"plan": "..."}`, and show it. Include the answered decisions in the plan. Do not execute it.
- **"do / start / work on <ref>"**: read the task. If `plan` is empty, make one as in "plan" (questions first) and save it before starting. `PATCH {"status":"in_progress"}`, execute the plan, then `PATCH {"status":"done"}` only once the work is verified. If blocked, leave it in_progress and say why.
- **"done / move / prio <ref> ..."**: `PATCH` the named field.
- **"delete <ref>"**: only after the user explicitly confirms; then `DELETE`.

Never change tasks the user did not mention. Report each change made (id, field, new value).
