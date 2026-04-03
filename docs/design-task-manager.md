# Design: Core Task Manager (v1)

## Overview
A personal household task manager for one user (Rotem) and optionally her husband. The goal is to reduce the cognitive load of remembering recurring household tasks and one-off action items.

## Core Concepts

### Tasks
- **Type**: `one-off` or `recurring`
- **Recurring logic**: on mark-as-done, a new task instance is auto-created with the next due date. History is preserved as a chain of completed instances linked via `parent_id`.
- **Status**: `pending`, `done`, `snoozed`
- **Snooze presets**: tomorrow, 3 days, next week, next month (no custom date picker — deferred to backlog)
- **Source**: `manual`, `gmail`, `whatsapp` — tracks where the task came from
- **source_ref**: stores the originating email/message ID for deduplication

### Categories (fixed, seeded on startup)
- Housekeeping, Car, Health, Packages, Finance, Social

### Assignees
- People are managed in Settings (add/remove)
- Currently: Rotem + husband
- Assignment is always manual (no auto-detection from message context — deferred)

### Task History
- No separate `TaskCompletion` table — history is the list of completed instances linked via `parent_id`
- `GET /api/tasks/:id/history` walks the parent_id chain

## Views & Filtering
- All tasks shown in a single Tasks page (Dashboard was merged into Tasks)
- Default filter: Pending
- Filters: status, category, date window (today / this week / this month / all time)
- Tasks grouped by assignee, sorted by due date ascending (NULLs last)
- Batch delete: checkbox mode with confirmation modal

## UI Stack
- React + Vite + Tailwind CSS v4
- Dark mode: class-based strategy via `.dark` on `<html>`, toggled from Settings
- Custom dropdown components (`FilterSelect`) — replaced native `<select>` due to bad UX
- All confirmation dialogs are in-app modals (no `window.confirm`)

## Backend Stack
- Go + chi router
- SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- DB migrations in `internal/db/db.go`
