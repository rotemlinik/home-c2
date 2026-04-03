# Backlog — Deferred Improvements

Things we consciously decided to start simple with, to revisit later.

---

## Integrations

- **Gmail / WhatsApp: real-time push** — currently sync is manual (button). Should become a background listener that creates tasks as messages arrive.
- **Gmail: sync UX** — sync takes ~60s with no progress feedback. Should show a progress indicator or stream results incrementally.
- **WhatsApp: contact whitelist UI** — contacts to monitor should be configurable in Settings. Currently will require manual DB entry or code change.
- **WhatsApp: group chat support** — design assumes whitelisted groups are included, but implementation scope TBD.

## Task Management

- **Task assignee: auto-detection** — assignee is always set manually. Could infer from message context (e.g. "tell Amit to..." → assign to husband).

## Testing

- **Backend tests — none exist yet.** Need comprehensive coverage:
  - *Unit*: task recurrence logic, snooze date arithmetic, Claude response parsing (markdown fence stripping, malformed JSON), Gmail body extraction, `last_message_date` advancement logic
  - *Integration*: full HTTP handler tests against a real SQLite DB — task CRUD, done/snooze flows, sync deduplication by `source_ref`, Gmail OAuth token refresh path

## Infrastructure

- **Secrets management** — credentials are in a local `.env` file. When deployed to AWS, move to AWS Secrets Manager or Parameter Store.
- **Deployment** — app runs locally. Needs Dockerization + AWS deployment (ECS or EC2).

## UI / UX

- **Mobile responsiveness** — UI is desktop-only. Sidebar layout breaks on small screens.
- **Notifications** — no push notifications or reminders when tasks are due.
- **Sync history UI** — `GET /api/sync/history` endpoint exists but isn't surfaced anywhere in the UI.
