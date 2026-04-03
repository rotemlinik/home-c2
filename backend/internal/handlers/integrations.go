package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"housekeeping/internal/gmail"
	"housekeeping/internal/whatsapp"
)

type IntegrationsHandler struct {
	db       *sql.DB
	gmail    *gmail.Client
	appURL   string
	waClient *whatsapp.Client
}

func NewIntegrationsHandler(db *sql.DB, waClient *whatsapp.Client) *IntegrationsHandler {
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "http://localhost:8080"
	}
	return &IntegrationsHandler{
		db:       db,
		gmail:    gmail.NewClient(os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"), appURL+"/api/integrations/gmail/callback"),
		appURL:   appURL,
		waClient: waClient,
	}
}

func (h *IntegrationsHandler) GmailAuth(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, h.gmail.AuthURL(), http.StatusFound)
}

func (h *IntegrationsHandler) GmailCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	token, err := h.gmail.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "exchange failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	email, err := h.gmail.GetUserEmail(r.Context(), token)
	if err != nil {
		email = ""
	}

	_, err = h.db.ExecContext(r.Context(), `
		INSERT INTO integrations (source, access_token, refresh_token, token_expiry, email)
		VALUES ('gmail', ?, ?, ?, ?)
		ON CONFLICT(source) DO UPDATE SET
			access_token  = excluded.access_token,
			refresh_token = excluded.refresh_token,
			token_expiry  = excluded.token_expiry,
			email         = excluded.email
	`, token.AccessToken, token.RefreshToken, token.Expiry.UTC().Format(time.RFC3339), email)
	if err != nil {
		http.Error(w, "save token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect back to the frontend
	http.Redirect(w, r, "http://localhost:5173/", http.StatusFound)
}

func (h *IntegrationsHandler) GmailStatus(w http.ResponseWriter, r *http.Request) {
	var result struct {
		Connected    bool    `json:"connected"`
		Email        *string `json:"email,omitempty"`
		LastSyncedAt *string `json:"last_synced_at,omitempty"`
		TasksCreated *int    `json:"tasks_created_last_sync,omitempty"`
	}

	var email sql.NullString
	var lastSyncedAt sql.NullString
	err := h.db.QueryRowContext(r.Context(), `
		SELECT email, last_synced_at FROM integrations WHERE source = 'gmail'
	`).Scan(&email, &lastSyncedAt)

	if err == nil {
		result.Connected = true
		if email.Valid {
			result.Email = &email.String
		}
		if lastSyncedAt.Valid {
			result.LastSyncedAt = &lastSyncedAt.String
		}
		// Get tasks created in last sync
		var count int
		h.db.QueryRowContext(r.Context(), `
			SELECT tasks_created FROM sync_log WHERE source = 'gmail' ORDER BY synced_at DESC LIMIT 1
		`).Scan(&count)
		result.TasksCreated = &count
	}

	respondJSON(w, result)
}

func (h *IntegrationsHandler) GmailDisconnect(w http.ResponseWriter, r *http.Request) {
	if _, err := h.db.ExecContext(r.Context(), `DELETE FROM integrations WHERE source = 'gmail'`); err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *IntegrationsHandler) WhatsAppConnect(w http.ResponseWriter, r *http.Request) {
	if err := h.waClient.Connect(r.Context()); err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	respondJSON(w, map[string]string{"status": "pending_qr"})
}

func (h *IntegrationsHandler) WhatsAppDisconnect(w http.ResponseWriter, r *http.Request) {
	h.waClient.Disconnect()
	w.WriteHeader(http.StatusNoContent)
}

func (h *IntegrationsHandler) WhatsAppStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.waClient.Status(r.Context())
	if err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}
	respondJSON(w, status)
}

func (h *IntegrationsHandler) WhatsAppQR(w http.ResponseWriter, r *http.Request) {
	qr, err := h.waClient.GetQR(r.Context())
	if err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}
	respondJSON(w, qr)
}

func (h *IntegrationsHandler) WhatsAppListContacts(w http.ResponseWriter, r *http.Request) {
	contacts, err := h.waClient.ListContacts(r.Context())
	if err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}
	respondJSON(w, contacts)
}

func (h *IntegrationsHandler) WhatsAppAddContact(w http.ResponseWriter, r *http.Request) {
	var body struct {
		JID   string `json:"jid"`
		Label string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.JID == "" {
		http.Error(w, "jid required", http.StatusBadRequest)
		return
	}
	contact, err := h.waClient.AddContact(r.Context(), body.JID, body.Label)
	if err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	respondJSON(w, contact)
}

func (h *IntegrationsHandler) WhatsAppRemoveContact(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.waClient.RemoveContact(r.Context(), id); err != nil {
		respondErr(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
