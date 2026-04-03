package whatsapp

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// Client wraps a whatsmeow client and persists session state to the app DB.
type Client struct {
	db  *sql.DB
	mu  sync.RWMutex
	wac *whatsmeow.Client // nil until connected
}

// NewClient creates a Client. It does not connect.
func NewClient(db *sql.DB) *Client {
	return &Client{db: db}
}

// AutoReconnect checks if the DB has a connected session and reconnects.
// It should be run in a goroutine. It does NOT poll — whatsmeow handles reconnect internally.
func (c *Client) AutoReconnect(ctx context.Context) {
	var status string
	err := c.db.QueryRowContext(ctx, `SELECT status FROM whatsapp_session WHERE id = 1`).Scan(&status)
	if err == nil && status == "connected" {
		if err := c.connect(ctx); err != nil {
			log.Printf("whatsapp: auto-reconnect failed: %v", err)
		}
	}
	<-ctx.Done()
}

// connect is the internal connect implementation shared by Connect and AutoReconnect.
func (c *Client) connect(ctx context.Context) error {
	container := sqlstore.NewWithDB(c.db, "sqlite3", waLog.Noop)
	if err := container.Upgrade(ctx); err != nil {
		return err
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return err
	}

	wac := whatsmeow.NewClient(deviceStore, waLog.Noop)
	wac.MessengerConfig = &whatsmeow.MessengerConfig{
		UserAgent:    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36",
		BaseURL:      "https://web.whatsapp.com",
		WebsocketURL: "wss://web.whatsapp.com/ws/chat",
	}
	wac.AddEventHandler(c.handleEvent)

	c.mu.Lock()
	c.wac = wac
	c.mu.Unlock()

	if wac.Store.ID == nil {
		// No existing session: start QR pairing flow
		qrChan, err := wac.GetQRChannel(ctx)
		if err != nil {
			return err
		}
		if err := wac.Connect(); err != nil {
			return err
		}
		go func() {
			for item := range qrChan {
				switch item.Event {
				case whatsmeow.QRChannelEventCode:
					c.upsertSession(map[string]interface{}{
						"status":        "pending_qr",
						"qr_data":       item.Code,
						"qr_expires_at": time.Now().Add(60 * time.Second).Format(time.RFC3339),
						"error":         nil,
					})
				case "success":
					phone := ""
					if wac.Store.ID != nil {
						phone = wac.Store.ID.User
					}
					c.upsertSession(map[string]interface{}{
						"status":       "connected",
						"phone":        phone,
						"qr_data":      nil,
						"connected_at": time.Now().Format(time.RFC3339),
						"error":        nil,
					})
				default:
					c.upsertSession(map[string]interface{}{
						"status": "disconnected",
						"error":  item.Event,
					})
				}
			}
		}()
	} else {
		// Existing session: reconnect silently
		if err := wac.Connect(); err != nil {
			return err
		}
	}
	return nil
}

// Connect is called from an HTTP handler to initiate the connection / QR flow.
func (c *Client) Connect(ctx context.Context) error {
	return c.connect(ctx)
}

// Disconnect disconnects the WhatsApp client and marks the session as disconnected.
func (c *Client) Disconnect() {
	c.mu.RLock()
	wac := c.wac
	c.mu.RUnlock()

	if wac != nil {
		wac.Disconnect()
	}

	c.upsertSession(map[string]interface{}{
		"status": "disconnected",
	})
}

// handleEvent is called by whatsmeow's internal goroutine for each event.
func (c *Client) handleEvent(evt interface{}) {
	switch e := evt.(type) {
	case *events.Message:
		if e.Info.IsFromMe {
			return
		}
		body := e.Message.GetConversation()
		if body == "" {
			if ext := e.Message.GetExtendedTextMessage(); ext != nil {
				body = ext.GetText()
			}
		}
		if body == "" {
			return
		}

		senderJID := e.Info.Sender.String()

		// Only store if sender is in whatsapp_contacts
		var count int
		err := c.db.QueryRow(`SELECT COUNT(*) FROM whatsapp_contacts WHERE jid = ?`, senderJID).Scan(&count)
		if err != nil || count == 0 {
			return
		}

		_, err = c.db.Exec(`
			INSERT OR IGNORE INTO whatsapp_messages (wa_message_id, sender_jid, sender_name, body, received_at)
			VALUES (?, ?, ?, ?, ?)
		`, e.Info.ID, senderJID, e.Info.PushName, body, e.Info.Timestamp.UTC().Format(time.RFC3339))
		if err != nil {
			log.Printf("whatsapp: insert message: %v", err)
		}

	case *events.Connected:
		c.mu.RLock()
		wac := c.wac
		c.mu.RUnlock()
		phone := ""
		if wac != nil && wac.Store.ID != nil {
			phone = wac.Store.ID.User
		}
		c.upsertSession(map[string]interface{}{
			"status": "connected",
			"phone":  phone,
			"error":  nil,
		})

	case *events.Disconnected:
		c.upsertSession(map[string]interface{}{
			"status": "disconnected",
		})

	case *events.LoggedOut:
		c.upsertSession(map[string]interface{}{
			"status": "disconnected",
			"error":  "logged_out",
		})
	}
}

// validSessionCols is the allowlist of columns that can be updated in whatsapp_session.
var validSessionCols = map[string]bool{
	"status": true, "phone": true, "qr_data": true, "qr_expires_at": true,
	"connected_at": true, "last_synced_at": true, "error": true,
}

// upsertSession updates whatsapp_session row (id=1) with the given fields.
func (c *Client) upsertSession(fields map[string]interface{}) {
	c.db.Exec(`INSERT OR IGNORE INTO whatsapp_session (id, status) VALUES (1, 'disconnected')`) //nolint:errcheck

	for col, val := range fields {
		if !validSessionCols[col] {
			log.Printf("whatsapp: ignoring invalid session column %q", col)
			continue
		}
		c.db.Exec(`UPDATE whatsapp_session SET `+col+` = ? WHERE id = 1`, val) //nolint:errcheck
	}
}

// SessionStatus holds the current session state.
type SessionStatus struct {
	Status      string  `json:"status"`
	Phone       *string `json:"phone,omitempty"`
	QRData      *string `json:"qr_data,omitempty"`
	QRExpiresAt *string `json:"qr_expires_at,omitempty"`
	ConnectedAt *string `json:"connected_at,omitempty"`
	LastSynced  *string `json:"last_synced_at,omitempty"`
	Error       *string `json:"error,omitempty"`
}

// Status returns the current session status from the DB.
func (c *Client) Status(ctx context.Context) (SessionStatus, error) {
	var s SessionStatus
	var phone, qrData, qrExpiresAt, connectedAt, lastSynced, errStr sql.NullString
	err := c.db.QueryRowContext(ctx, `
		SELECT status, phone, qr_data, qr_expires_at, connected_at, last_synced_at, error
		FROM whatsapp_session WHERE id = 1
	`).Scan(&s.Status, &phone, &qrData, &qrExpiresAt, &connectedAt, &lastSynced, &errStr)
	if err == sql.ErrNoRows {
		s.Status = "disconnected"
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if phone.Valid {
		s.Phone = &phone.String
	}
	if qrData.Valid {
		s.QRData = &qrData.String
	}
	if qrExpiresAt.Valid {
		s.QRExpiresAt = &qrExpiresAt.String
	}
	if connectedAt.Valid {
		s.ConnectedAt = &connectedAt.String
	}
	if lastSynced.Valid {
		s.LastSynced = &lastSynced.String
	}
	if errStr.Valid {
		s.Error = &errStr.String
	}
	return s, nil
}

// QRData holds the QR code and expiry for the pending_qr state.
type QRData struct {
	QRData    *string `json:"qr_data"`
	ExpiresAt *string `json:"expires_at"`
}

// GetQR returns the current QR code data if in pending_qr state.
func (c *Client) GetQR(ctx context.Context) (QRData, error) {
	var d QRData
	var status string
	var qrData, qrExpiresAt sql.NullString
	err := c.db.QueryRowContext(ctx, `
		SELECT status, qr_data, qr_expires_at FROM whatsapp_session WHERE id = 1
	`).Scan(&status, &qrData, &qrExpiresAt)
	if err == sql.ErrNoRows || status != "pending_qr" {
		return d, nil
	}
	if err != nil {
		return d, err
	}
	if qrData.Valid {
		d.QRData = &qrData.String
	}
	if qrExpiresAt.Valid {
		d.ExpiresAt = &qrExpiresAt.String
	}
	return d, nil
}

// Contact represents a whatsapp_contacts row.
type Contact struct {
	ID        int64   `json:"id"`
	JID       string  `json:"jid"`
	Label     *string `json:"label,omitempty"`
	CreatedAt string  `json:"created_at"`
}

// ListContacts returns all whatsapp contacts.
func (c *Client) ListContacts(ctx context.Context) ([]Contact, error) {
	rows, err := c.db.QueryContext(ctx, `SELECT id, jid, label, created_at FROM whatsapp_contacts ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []Contact
	for rows.Next() {
		var ct Contact
		var label sql.NullString
		if err := rows.Scan(&ct.ID, &ct.JID, &label, &ct.CreatedAt); err != nil {
			return nil, err
		}
		if label.Valid {
			ct.Label = &label.String
		}
		contacts = append(contacts, ct)
	}
	if contacts == nil {
		contacts = []Contact{}
	}
	return contacts, nil
}

// AddContact inserts a new contact.
func (c *Client) AddContact(ctx context.Context, jid, label string) (Contact, error) {
	var labelVal interface{}
	if label != "" {
		labelVal = label
	}
	res, err := c.db.ExecContext(ctx, `INSERT INTO whatsapp_contacts (jid, label) VALUES (?, ?)`, jid, labelVal)
	if err != nil {
		return Contact{}, err
	}
	id, _ := res.LastInsertId()
	var ct Contact
	var lbl sql.NullString
	c.db.QueryRowContext(ctx, `SELECT id, jid, label, created_at FROM whatsapp_contacts WHERE id = ?`, id).
		Scan(&ct.ID, &ct.JID, &lbl, &ct.CreatedAt)
	if lbl.Valid {
		ct.Label = &lbl.String
	}
	return ct, nil
}

// RemoveContact deletes a contact by ID.
func (c *Client) RemoveContact(ctx context.Context, id int64) error {
	_, err := c.db.ExecContext(ctx, `DELETE FROM whatsapp_contacts WHERE id = ?`, id)
	return err
}

// WAMessage represents an unprocessed whatsapp message.
type WAMessage struct {
	ID          int64  `json:"id"`
	WAMessageID string `json:"wa_message_id"`
	SenderJID   string `json:"sender_jid"`
	SenderName  string `json:"sender_name"`
	Body        string `json:"body"`
	ReceivedAt  string `json:"received_at"`
}

// GetUnprocessedMessages returns all unprocessed whatsapp messages.
func (c *Client) GetUnprocessedMessages(ctx context.Context) ([]WAMessage, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT id, wa_message_id, sender_jid, COALESCE(sender_name, ''), body, received_at
		FROM whatsapp_messages WHERE processed = 0 ORDER BY received_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []WAMessage
	for rows.Next() {
		var m WAMessage
		if err := rows.Scan(&m.ID, &m.WAMessageID, &m.SenderJID, &m.SenderName, &m.Body, &m.ReceivedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

// MarkProcessed marks the given message IDs as processed.
func (c *Client) MarkProcessed(ctx context.Context, ids []int64) error {
	for _, id := range ids {
		if _, err := c.db.ExecContext(ctx, `UPDATE whatsapp_messages SET processed = 1 WHERE id = ?`, id); err != nil {
			return err
		}
	}
	return nil
}

// UpdateLastSynced updates the last_synced_at timestamp for the session.
func (c *Client) UpdateLastSynced(ctx context.Context) {
	c.db.ExecContext(ctx, `UPDATE whatsapp_session SET last_synced_at = CURRENT_TIMESTAMP WHERE id = 1`) //nolint:errcheck
}
