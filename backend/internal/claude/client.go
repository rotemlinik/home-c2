package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type ExtractedTask struct {
	EmailID  string  `json:"email_id"`
	Title    string  `json:"title"`
	Category string  `json:"category"`
	DueDate  *string `json:"due_date"` // YYYY-MM-DD or null
	Notes    string  `json:"notes"`
}

type EmailInput struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
	From    string `json:"from"`
	Date    string `json:"date"`
	Body    string `json:"body"`
}

type MessageInput struct {
	ID         string `json:"id"`
	SenderName string `json:"sender_name"`
	Date       string `json:"date"`
	Body       string `json:"body"`
}

type Client struct {
	apiKey string
	http   *http.Client
}

func NewClient() *Client {
	return &Client{
		apiKey: os.Getenv("ANTHROPIC_API_KEY"),
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) ExtractTasks(ctx context.Context, emails []EmailInput) ([]ExtractedTask, error) {
	if len(emails) == 0 {
		return nil, nil
	}

	emailsJSON, err := json.Marshal(emails)
	if err != nil {
		return nil, fmt.Errorf("marshal emails: %w", err)
	}

	prompt := fmt.Sprintf(`You are a household task extraction assistant. Given a list of emails, identify any that require action and extract tasks.

For each actionable email, return a JSON object with:
- email_id: the email ID
- title: concise task title (max 60 chars)
- category: one of "Housekeeping", "Car", "Health", "Packages", "Finance"
- due_date: ISO date YYYY-MM-DD if mentioned or clearly inferrable, otherwise null
- notes: brief context (max 100 chars)

Extract tasks for:
- Package shipments, out-for-delivery, and pickup-ready notices (→ Packages; use the email date as due_date)
- Payment requests, invoices requesting payment, overdue notices (→ Finance)
- Appointment reminders where action is required (→ Health or Car)
- Subscriptions where you must manually renew or take action to avoid losing service (→ Finance)
- Financial or legal documents that require reading or a decision: annual reports, pension/provident fund statements, tax documents, insurance policy updates (→ Finance)
- Any soft signals: "reminder", "don't forget", "action required", "please respond"

Skip — do NOT create tasks for:
- Restaurant, hotel, or event reservations (these are confirmations, not todos)
- Doctor or therapist appointment confirmation
- Auto-renewal notifications ("your subscription will renew", "upcoming charge") — these are informational, no action needed
- Receipts where payment already happened
- Pre-paid bill receipts and payment confirmations
- Newsletters, promotions, and marketing emails
- General informational updates with no action needed
- Delivery confirmations where the item has already arrived

Return ONLY a valid JSON array, no other text. If no tasks found, return [].

Today's date: %s

Emails:
%s`, time.Now().Format("2006-01-02"), string(emailsJSON))

	reqBody := map[string]any{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 1024,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call claude: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("empty response from claude")
	}

	text := strings.TrimSpace(result.Content[0].Text)
	// Strip markdown code fences if present
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}

	var tasks []ExtractedTask
	if err := json.Unmarshal([]byte(text), &tasks); err != nil {
		return nil, fmt.Errorf("parse tasks JSON: %w", err)
	}

	return tasks, nil
}

func (c *Client) ExtractTasksFromMessages(ctx context.Context, messages []MessageInput) ([]ExtractedTask, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		return nil, fmt.Errorf("marshal messages: %w", err)
	}

	prompt := fmt.Sprintf(`You are a household task extraction assistant. Given a list of WhatsApp messages, identify any that contain action items.

For each actionable message, return a JSON object with:
- email_id: the message ID (reuse this field as a correlation key)
- title: concise task title (max 60 chars)
- category: one of "Housekeeping", "Car", "Health", "Packages", "Finance", "Social"
- due_date: ISO date YYYY-MM-DD if mentioned or clearly inferrable, otherwise null
- notes: brief context (max 100 chars)

Create a task when:
- You made a commitment: "I'll call you", "I'll send that", "I'll check on it"
- Someone asked you to do something: "can you buy X?", "don't forget to Y"
- A scheduled event or appointment is mentioned: "party Saturday", "dentist Thursday"
- Time-sensitive info: "the offer expires tomorrow", "last day is Sunday"

Skip:
- Casual conversation ("how are you?", "good morning", reactions)
- News forwards and jokes
- General chit-chat with no actionable content

Return ONLY a valid JSON array. If no tasks found, return [].

Today's date: %s

Messages:
%s`, time.Now().Format("2006-01-02"), string(messagesJSON))

	reqBody := map[string]any{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 1024,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call claude: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("empty response from claude")
	}

	text := strings.TrimSpace(result.Content[0].Text)
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}

	var tasks []ExtractedTask
	if err := json.Unmarshal([]byte(text), &tasks); err != nil {
		return nil, fmt.Errorf("parse tasks JSON: %w", err)
	}

	return tasks, nil
}
