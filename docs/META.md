# Project Meta — For Incoming Agent

## What This Project Is
A personal household "command and control" web app for Rotem (and her husband). It manages recurring and one-off household tasks, and auto-creates tasks from Gmail and WhatsApp messages using Claude AI.

---

## Design Process Summary
The project was designed iteratively in conversation with the user:
1. **v1**: Defined core task manager — recurring tasks, assignees, categories, snooze presets
2. **v2**: Added Gmail integration — OAuth, sync-on-demand, Claude for email-to-task extraction
3. **v3**: Added WhatsApp integration — whatsmeow unofficial client, contact whitelist, message buffering, Claude extraction

Key philosophy throughout: **start simple, defer complexity to backlog**.

---

## Implementation Summary

### What's Built and Working
- **Full task CRUD**: create, edit, done, snooze, delete, history
- **Recurring tasks**: auto-creates next instance on done, parent_id chain for history
- **Dark mode**: class-based Tailwind v4 strategy, persisted in localStorage
- **Gmail integration**: OAuth 2.0, incremental sync, Claude task extraction, deduplication
- **WhatsApp integration**: whatsmeow, QR pairing, contact whitelist, message buffering, Claude extraction
- **Batch delete**: multi-select with in-app confirmation modal
- **Task grouping by assignee**
- **Date window filters**: today / this week / this month
- **62 frontend tests** (Vitest + React Testing Library)
- **0 backend tests** (tracked in backlog)

### What's Not Yet Working
- WhatsApp QR pairing: implementation is complete but first live test got a WhatsApp server-side error ("Can't link new devices at this time") — likely a temporary throttle, not a code issue. Retry after ~30 minutes.

---

## Repository Layout
```
housekeeping/
├── backend/                    Go backend (chi router, SQLite)
│   ├── .env                    Secrets (gitignored) — see below
│   ├── internal/
│   │   ├── db/db.go            SQLite migrations + open
│   │   ├── handlers/           HTTP handlers (tasks, integrations, sync)
│   │   ├── gmail/client.go     Gmail OAuth + email fetching
│   │   ├── whatsapp/client.go  whatsmeow wrapper
│   │   └── claude/client.go    Claude API (Haiku) — email + WhatsApp extraction
│   └── main.go                 Router wiring, env loading, WhatsApp goroutine
├── frontend/                   React + Vite + Tailwind CSS v4
│   ├── src/
│   │   ├── api.ts              All API functions + TypeScript interfaces
│   │   ├── pages/              Tasks.tsx, Settings.tsx
│   │   └── components/         TaskCard, TaskForm, Modal, FilterSelect, SyncButton
│   └── vite.config.ts          Proxy to :8080, 2min timeout for sync
├── docs/                       ← YOU ARE HERE
│   ├── META.md                 This file
│   ├── design-task-manager.md  v1 core task manager design
│   ├── design-gmail-integration.md
│   └── design-whatsapp-integration.md
└── BACKLOG.md                  Deferred improvements — read this before starting work
```

---

## How to Run

### Backend
```bash
cd backend
go run .   # reads .env automatically
```
Listens on `:8080`. The `.env` file must exist with:
```
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
ANTHROPIC_API_KEY=...
```

### Frontend
```bash
cd frontend
npm run dev   # http://localhost:5173
```
Proxies `/api` to `http://localhost:8080` with a 2-minute timeout (needed for sync).

---

## Key Technical Decisions

| Decision | Choice | Reason |
|---|---|---|
| SQLite driver | `modernc.org/sqlite` (pure Go) | No CGO required |
| Frontend framework | React + Vite + Tailwind v4 | Fast setup, Tailwind v4 class-strategy dark mode |
| WhatsApp library | `go.mau.fi/whatsmeow` | Only well-maintained pure-Go WhatsApp Web client |
| Claude model | `claude-haiku-4-5-20251001` | Cost-effective for high-volume email/message parsing |
| Sync model | Manual (button) | Start simple; real-time push is in backlog |
| Confirmation dialogs | In-app Modal component | No `window.confirm` anywhere in the codebase |

---

## User Profile
- **Rotem** — the primary user. Her husband is also set up as an assignee.
- The app is personal/household — not a business product.
- Runs locally for now; AWS deployment is planned (secrets management deferred).
- Communication style: direct, prefers lean implementations, defers complexity explicitly to backlog.

---

## Important Gotchas

1. **whatsmeow + modernc sqlite**: `sqlstore.NewWithDB(db, "sqlite3", ...)` — pass `"sqlite3"` as dialect string even though DB opens with `"sqlite"`. This is intentional.
2. **Claude markdown fences**: Claude Haiku sometimes wraps JSON in ```json...``` — the client strips this before parsing (`internal/claude/client.go`).
3. **Gmail `last_message_date`**: This is what makes sync incremental. If NULL, defaults to 30 days ago. Always updated after sync.
4. **Frontend proxy timeout**: Sync takes ~60s. Vite proxy timeout is set to 120s in `vite.config.ts`.
5. **Task source field**: DB CHECK constraint — valid values are `manual`, `gmail`, `whatsapp`. Adding a new source requires a DB migration.
6. **No backend tests**: This is known and tracked in `BACKLOG.md`.

---

## Where to Look for Things

| Thing | Location |
|---|---|
| Deferred improvements | `BACKLOG.md` |
| Feature designs | `docs/design-*.md` |
| DB schema | `backend/internal/db/db.go` → `migrate()` |
| API contract | `frontend/src/api.ts` (interfaces + functions) |
| Claude prompts | `backend/internal/claude/client.go` |
| WhatsApp event handling | `backend/internal/whatsapp/client.go` → `handleEvent` |
| Sync logic | `backend/internal/handlers/sync.go` |
| Frontend tests | `frontend/src/test/` |
