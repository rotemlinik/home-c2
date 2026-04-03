# Design: WhatsApp Integration (v3)

## Overview
Auto-create tasks from WhatsApp messages using `go.mau.fi/whatsmeow` (unofficial WhatsApp Web multi-device protocol). User manually triggers sync via a button in the UI.

## Key Decisions
- **Unofficial API**: `whatsmeow` — against WhatsApp ToS technically, but acceptable risk for personal low-volume use
- **Sync model**: manual (button), NOT real-time push (real-time is in backlog)
- **Contact whitelist**: only messages from explicitly whitelisted contacts are buffered
- **No historical backfill**: WhatsApp doesn't expose message history via the unofficial API — only messages received after connecting are captured

## Architecture

### Background Connection
whatsmeow maintains a persistent WebSocket to WhatsApp servers. This runs as a background goroutine (`AutoReconnect`) even though sync is manual — necessary to receive and buffer incoming messages in real-time.

On app startup:
- If `whatsapp_session.status = 'connected'`, auto-reconnect using stored device keys (no QR needed)
- If disconnected, wait for user to initiate connect from Settings

### QR Pairing Flow
1. User clicks **Connect** in Settings
2. `POST /api/integrations/whatsapp/connect` starts the QR flow
3. Frontend polls `GET /api/integrations/whatsapp/qr` every 2 seconds
4. QR data returned as a string — frontend renders it as an image via external QR renderer (`api.qrserver.com`)
5. User scans with phone (WhatsApp → Linked Devices → Link a device)
6. On success, `whatsapp_session.status` → `'connected'`, phone stored, QR cleared
7. Frontend detects `status='connected'` via polling and stops

QR codes expire every ~60 seconds. whatsmeow issues new ones automatically via `GetQRChannel`.

### Message Buffering
- Incoming messages from whitelisted contacts → `whatsapp_messages` table (`processed=0`)
- Deduplication: `INSERT OR IGNORE` on `wa_message_id`
- Group chats: supported (sender JID must be in whitelist)

### Sync Flow
1. `POST /api/sync/whatsapp`
2. Fetch all `whatsapp_messages WHERE processed=0`
3. Batch to Claude (20/batch) for task extraction
4. Insert tasks with `source='whatsapp'`, `source_ref=wa_message_id`
5. Mark messages as `processed=1`
6. Log to `sync_log`, update `whatsapp_session.last_synced_at`

## Claude Prompt Strategy
Model: `claude-haiku-4-5-20251001`

**Extract tasks for:**
- Commitments you made: "I'll call you", "I'll send that", "I'll check on it"
- Requests from others: "can you buy X?", "don't forget to Y"
- Scheduled events/appointments: "party Saturday", "dentist Thursday" → Social or Health
- Time-sensitive info: "offer expires tomorrow", "last day Sunday"

**Skip:**
- Casual conversation, greetings, reactions/emojis
- News forwards, jokes
- General chit-chat with no actionable content

## DB Schema (additions)
```sql
whatsapp_session    -- singleton (id=1): status, phone, qr_data, qr_expires_at, last_synced_at
whatsapp_messages   -- buffer: wa_message_id, sender_jid, body, received_at, processed
whatsapp_contacts   -- whitelist: jid, label
```
whatsmeow also creates its own `whatsmeow_*` tables for device key storage — managed by the library.

## sqlstore Driver Workaround
`sqlstore.NewWithDB(db, "sqlite3", ...)` is called with `"sqlite3"` as the dialect string even though the app's DB was opened with `"sqlite"` (modernc driver). This is intentional — sqlstore only uses the string to select DDL dialect, not to open a connection.

## Cost Estimate
- ~$0.002 per batch of 20 messages
- Typical day (50 monitored messages): ~$0.005/day → ~$0.15/month

## API Endpoints
```
POST /api/integrations/whatsapp/connect          → start QR flow, 202
GET  /api/integrations/whatsapp/status           → { status, phone, last_synced_at }
GET  /api/integrations/whatsapp/qr               → { qr_data, expires_at }
POST /api/integrations/whatsapp/disconnect       → 204
GET  /api/integrations/whatsapp/contacts         → list whitelist
POST /api/integrations/whatsapp/contacts         → add { jid, label }
DELETE /api/integrations/whatsapp/contacts/{id}  → remove
POST /api/sync/whatsapp                          → { messages_read, tasks_created }
```

## Key Files
- `internal/whatsapp/client.go` — Manager wrapping whatsmeow: lifecycle, QR, event handler, contact whitelist, message buffer
- `internal/claude/client.go` — Added `MessageInput` + `ExtractTasksFromMessages`
- `internal/handlers/integrations.go` — WhatsApp HTTP handlers (added to existing file)
- `internal/handlers/sync.go` — `SyncWhatsApp` (added to existing file)

## Current Status
Implementation complete and building clean. QR pairing was attempted but WhatsApp showed "Can't link new devices at this time" — likely a temporary server-side throttle. Not a code issue. Retry after ~30 minutes.
