# Design: Gmail Integration (v2)

## Overview
Auto-create tasks from incoming emails using Gmail OAuth + Claude API for parsing. User manually triggers sync via a button in the UI.

## Auth Flow
- Gmail OAuth 2.0 via `golang.org/x/oauth2`
- Google Cloud project with OAuth client credentials (stored in `.env`)
- Redirect URI: `http://localhost:8080/api/integrations/gmail/callback`
- Tokens (access + refresh) stored in `integrations` table
- Token refresh is handled automatically by the oauth2 library

## Sync Flow
1. User clicks **Sync** button in the UI
2. `POST /api/sync/gmail` fetches emails since `last_message_date` (or 30 days if first sync)
3. Emails filtered by deduplication: `source_ref` (email message ID) already in tasks → skip
4. Remaining emails batched (20 per batch) and sent to Claude Haiku for task extraction
5. Extracted tasks inserted with `source='gmail'`, `source_ref=email_id`
6. `last_message_date` updated to the most recent email's date → next sync is incremental
7. Sync logged to `sync_log` table

## Claude Prompt Strategy
Model: `claude-haiku-4-5-20251001`

**Extract tasks for:**
- Package pickup/ready notices → due_date = email date
- Payment requests, invoices requiring payment, overdue notices → Finance
- Appointment reminders requiring action → Health or Car
- Subscriptions requiring manual action → Finance
- Financial/legal documents to read: annual reports, pension statements, tax docs, insurance updates → Finance
- Soft signals: "reminder", "don't forget", "action required"

**Skip:**
- Restaurant/hotel/event reservation confirmations
- Auto-renewal notifications ("your subscription will renew") — informational only
- Receipts, payment confirmations, delivery confirmations
- Newsletters, promotions, marketing
- General informational updates

**Markdown fence stripping**: Claude sometimes wraps JSON in ```json ... ``` — the client strips this before parsing.

## Cost Estimate
- ~$0.03 per full 200-email sync (initial backfill)
- ~$0.005/day for incremental syncs
- ~$0.15/month ongoing

## API Endpoints
```
GET  /api/integrations/gmail/auth       → redirect to Google OAuth
GET  /api/integrations/gmail/callback   → exchange code, store tokens, redirect to frontend
GET  /api/integrations/gmail/status     → { connected, email, last_synced_at, tasks_created_last_sync }
DELETE /api/integrations/gmail          → disconnect (delete row)
POST /api/sync/gmail                    → run sync, return { emails_read, tasks_created }
GET  /api/sync/history                  → last 10 sync_log entries
```

## Key Files
- `internal/gmail/client.go` — OAuth client, FetchEmails, MIME body extraction
- `internal/claude/client.go` — Claude API HTTP client, ExtractTasks
- `internal/handlers/integrations.go` — OAuth flow handlers
- `internal/handlers/sync.go` — SyncGmail + SyncHistory

## Settings
Credentials in `backend/.env`:
```
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
ANTHROPIC_API_KEY=...
```
Loaded at startup via `loadEnv(".env")` in `main.go`. `.env` is gitignored.
