package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"housekeeping/internal/claude"
	gmailpkg "housekeeping/internal/gmail"
	"housekeeping/internal/whatsapp"
)

type SyncHandler struct {
	db       *sql.DB
	claude   *claude.Client
	waClient *whatsapp.Client
}

func NewSyncHandler(db *sql.DB, waClient *whatsapp.Client) *SyncHandler {
	return &SyncHandler{
		db:       db,
		claude:   claude.NewClient(),
		waClient: waClient,
	}
}

func (h *SyncHandler) SyncGmail(w http.ResponseWriter, r *http.Request) {
	// Load integration
	var accessToken, refreshToken string
	var tokenExpiryStr string
	var lastMessageDate sql.NullString

	err := h.db.QueryRowContext(r.Context(), `
		SELECT access_token, refresh_token, token_expiry, last_message_date
		FROM integrations WHERE source = 'gmail'
	`).Scan(&accessToken, &refreshToken, &tokenExpiryStr, &lastMessageDate)
	if err == sql.ErrNoRows {
		http.Error(w, "Gmail not connected", http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Printf("sync: failed to load gmail integration: %v", err)
		http.Error(w, "Gmail not connected", http.StatusBadRequest)
		return
	}

	tokenExpiry, err := parseFlexibleTime(tokenExpiryStr)
	if err != nil {
		log.Printf("sync: failed to parse token_expiry %q: %v, treating as expired", tokenExpiryStr, err)
		tokenExpiry = time.Now().Add(-time.Hour) // treat as expired, oauth2 will refresh
	}

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "http://localhost:8080"
	}

	gmailClient := gmailpkg.NewClient(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		appURL+"/api/integrations/gmail/callback",
	)

	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       tokenExpiry,
		TokenType:    "Bearer",
	}

	// Determine lookback window
	since := time.Now().AddDate(0, 0, -30)
	if lastMessageDate.Valid && lastMessageDate.String != "" {
		if parsed, err := parseFlexibleTime(lastMessageDate.String); err == nil {
			since = parsed
		} else {
			log.Printf("sync: failed to parse last_message_date %q: %v", lastMessageDate.String, err)
		}
	}

	log.Printf("sync: fetching emails since %s", since.Format("2006-01-02"))
	emails, newToken, err := gmailClient.FetchEmails(r.Context(), token, since)
	if err != nil {
		h.logSync(r.Context(), "gmail", 0, 0, err.Error())
		respondErr(w, fmt.Errorf("fetch emails: %w", err), http.StatusInternalServerError)
		return
	}
	log.Printf("sync: fetched %d emails, %d new after dedup", len(emails), len(emails))

	// Update token if refreshed
	if newToken != nil && newToken.AccessToken != accessToken {
		h.db.ExecContext(r.Context(), `
			UPDATE integrations SET access_token = ?, token_expiry = ? WHERE source = 'gmail'
		`, newToken.AccessToken, newToken.Expiry.UTC().Format(time.RFC3339))
	}

	// Filter out already-processed emails
	var newEmails []gmailpkg.Email
	for _, e := range emails {
		var count int
		h.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM tasks WHERE source_ref = ?`, e.ID).Scan(&count)
		if count == 0 {
			newEmails = append(newEmails, e)
		}
	}

	if len(newEmails) == 0 {
		h.updateSyncTimestamp(r.Context(), emails)
		respondJSON(w, map[string]int{"tasks_created": 0, "emails_read": len(emails)})
		return
	}

	// Convert to Claude input format
	var claudeEmails []claude.EmailInput
	for _, e := range newEmails {
		claudeEmails = append(claudeEmails, claude.EmailInput{
			ID:      e.ID,
			Subject: e.Subject,
			From:    e.From,
			Date:    e.Date.Format("2006-01-02"),
			Body:    e.Body,
		})
	}

	// Extract tasks in batches of 20
	var tasksCreated int
	batchSize := 20
	for i := 0; i < len(claudeEmails); i += batchSize {
		end := i + batchSize
		if end > len(claudeEmails) {
			end = len(claudeEmails)
		}
		batch := claudeEmails[i:end]

		extracted, err := h.claude.ExtractTasks(r.Context(), batch)
		if err != nil {
			log.Printf("sync: claude batch %d error: %v", i/batchSize, err)
			continue
		}
		log.Printf("sync: claude batch %d extracted %d tasks", i/batchSize, len(extracted))

		for _, t := range extracted {
			// Find category ID
			var categoryID *int64
			var catID int64
			if err := h.db.QueryRowContext(r.Context(), `SELECT id FROM categories WHERE name = ?`, t.Category).Scan(&catID); err == nil {
				categoryID = &catID
			}

			_, err := h.db.ExecContext(r.Context(), `
				INSERT OR IGNORE INTO tasks (title, type, due_date, category_id, status, notes, source, source_ref)
				VALUES (?, 'one-off', ?, ?, 'pending', ?, 'gmail', ?)
			`, t.Title, t.DueDate, categoryID, t.Notes, t.EmailID)
			if err == nil {
				tasksCreated++
			}
		}
	}

	h.logSync(r.Context(), "gmail", len(newEmails), tasksCreated, "")
	h.updateSyncTimestamp(r.Context(), emails)

	respondJSON(w, map[string]int{"tasks_created": tasksCreated, "emails_read": len(emails)})
}

func (h *SyncHandler) SyncHistory(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT source, synced_at, emails_read, tasks_created, error
		FROM sync_log ORDER BY synced_at DESC LIMIT 10
	`)
	if err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type entry struct {
		Source       string  `json:"source"`
		SyncedAt     string  `json:"synced_at"`
		EmailsRead   int     `json:"emails_read"`
		TasksCreated int     `json:"tasks_created"`
		Error        *string `json:"error,omitempty"`
	}

	var entries []entry
	for rows.Next() {
		var e entry
		var errStr sql.NullString
		if err := rows.Scan(&e.Source, &e.SyncedAt, &e.EmailsRead, &e.TasksCreated, &errStr); err != nil {
			log.Printf("sync history: scan error: %v", err)
			continue
		}
		if errStr.Valid {
			e.Error = &errStr.String
		}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []entry{}
	}
	respondJSON(w, entries)
}

func (h *SyncHandler) logSync(ctx context.Context, source string, emailsRead, tasksCreated int, errMsg string) {
	var errVal interface{}
	if errMsg != "" {
		errVal = errMsg
	}
	h.db.ExecContext(ctx, `
		INSERT INTO sync_log (source, emails_read, tasks_created, error)
		VALUES (?, ?, ?, ?)
	`, source, emailsRead, tasksCreated, errVal)
}

func (h *SyncHandler) SyncWhatsApp(w http.ResponseWriter, r *http.Request) {
	msgs, err := h.waClient.GetUnprocessedMessages(r.Context())
	if err != nil {
		respondErr(w, fmt.Errorf("get messages: %w", err), http.StatusInternalServerError)
		return
	}
	if len(msgs) == 0 {
		respondJSON(w, map[string]int{"tasks_created": 0, "messages_read": 0})
		return
	}

	// Convert to Claude input format
	var claudeMsgs []claude.MessageInput
	for _, m := range msgs {
		claudeMsgs = append(claudeMsgs, claude.MessageInput{
			ID:         m.WAMessageID,
			SenderName: m.SenderName,
			Date:       m.ReceivedAt,
			Body:       m.Body,
		})
	}

	// Extract tasks in batches of 20
	var tasksCreated int
	batchSize := 20
	for i := 0; i < len(claudeMsgs); i += batchSize {
		end := i + batchSize
		if end > len(claudeMsgs) {
			end = len(claudeMsgs)
		}
		batch := claudeMsgs[i:end]

		extracted, err := h.claude.ExtractTasksFromMessages(r.Context(), batch)
		if err != nil {
			log.Printf("sync/whatsapp: claude batch %d error: %v", i/batchSize, err)
			continue
		}
		log.Printf("sync/whatsapp: claude batch %d extracted %d tasks", i/batchSize, len(extracted))

		for _, t := range extracted {
			var categoryID *int64
			var catID int64
			if err := h.db.QueryRowContext(r.Context(), `SELECT id FROM categories WHERE name = ?`, t.Category).Scan(&catID); err == nil {
				categoryID = &catID
			}

			_, err := h.db.ExecContext(r.Context(), `
				INSERT OR IGNORE INTO tasks (title, type, due_date, category_id, status, notes, source, source_ref)
				VALUES (?, 'one-off', ?, ?, 'pending', ?, 'whatsapp', ?)
			`, t.Title, t.DueDate, categoryID, t.Notes, t.EmailID)
			if err == nil {
				tasksCreated++
			}
		}
	}

	// Mark all messages as processed
	var ids []int64
	for _, m := range msgs {
		ids = append(ids, m.ID)
	}
	if err := h.waClient.MarkProcessed(r.Context(), ids); err != nil {
		log.Printf("sync/whatsapp: mark processed: %v", err)
	}

	h.logSync(r.Context(), "whatsapp", len(msgs), tasksCreated, "")
	h.waClient.UpdateLastSynced(r.Context())

	respondJSON(w, map[string]int{"tasks_created": tasksCreated, "messages_read": len(msgs)})
}

func (h *SyncHandler) updateSyncTimestamp(ctx context.Context, emails []gmailpkg.Email) {
	var latest time.Time
	for _, e := range emails {
		if e.Date.After(latest) {
			latest = e.Date
		}
	}
	if latest.IsZero() {
		h.db.ExecContext(ctx, `UPDATE integrations SET last_synced_at = CURRENT_TIMESTAMP WHERE source = 'gmail'`)
		return
	}
	h.db.ExecContext(ctx, `
		UPDATE integrations SET last_synced_at = CURRENT_TIMESTAMP, last_message_date = ? WHERE source = 'gmail'
	`, latest.UTC().Format(time.RFC3339))
}

// parseFlexibleTime parses timestamps stored in various formats, including
// Go's default time.Time.String() format which older data may use.
func parseFlexibleTime(s string) (time.Time, error) {
	// Strip Go's monotonic clock suffix (e.g. " m=+3608.716621793")
	if idx := strings.Index(s, " m="); idx != -1 {
		s = s[:idx]
	}

	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05 -0700 MST",  // Go time.Time.String() format
		"2006-01-02 15:04:05 +0000 UTC",   // Go time.Time.String() UTC variant
		"2006-01-02 15:04:05-07:00",       // SQLite with timezone
		"2006-01-02T15:04:05Z",            // ISO 8601
		"2006-01-02 15:04:05",             // SQLite default
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized time format: %q", s)
}
