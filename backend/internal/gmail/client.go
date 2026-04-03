package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gmailv1 "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type Email struct {
	ID      string
	Subject string
	From    string
	Date    time.Time
	Snippet string
	Body    string
}

type Client struct {
	config *oauth2.Config
}

func NewClient(clientID, clientSecret, redirectURL string) *Client {
	return &Client{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{gmailv1.GmailReadonlyScope},
			Endpoint:     google.Endpoint,
		},
	}
}

func (c *Client) AuthURL() string {
	return c.config.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

func (c *Client) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return c.config.Exchange(ctx, code)
}

func (c *Client) GetUserEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	svc, err := c.service(ctx, token)
	if err != nil {
		return "", err
	}
	profile, err := svc.Users.GetProfile("me").Do()
	if err != nil {
		return "", fmt.Errorf("get profile: %w", err)
	}
	return profile.EmailAddress, nil
}

func (c *Client) FetchEmails(ctx context.Context, token *oauth2.Token, since time.Time) ([]Email, *oauth2.Token, error) {
	ts := c.config.TokenSource(ctx, token)
	svc, err := gmailv1.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, nil, fmt.Errorf("create gmail service: %w", err)
	}

	newToken, _ := ts.Token() // get refreshed token if any

	query := fmt.Sprintf("after:%s", since.Format("2006/01/02"))

	var emails []Email
	pageToken := ""

	for {
		call := svc.Users.Messages.List("me").Q(query).MaxResults(50)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			return nil, nil, fmt.Errorf("list messages: %w", err)
		}

		log.Printf("gmail: listed %d messages (page)", len(resp.Messages))
		for _, m := range resp.Messages {
			email, err := c.fetchMessage(svc, m.Id)
			if err != nil {
				log.Printf("gmail: fetchMessage %s error: %v", m.Id, err)
				continue
			}
			emails = append(emails, email)
		}

		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return emails, newToken, nil
}

func (c *Client) service(ctx context.Context, token *oauth2.Token) (*gmailv1.Service, error) {
	ts := c.config.TokenSource(ctx, token)
	return gmailv1.NewService(ctx, option.WithTokenSource(ts))
}

func (c *Client) fetchMessage(svc *gmailv1.Service, id string) (Email, error) {
	msg, err := svc.Users.Messages.Get("me", id).Format("full").Do()
	if err != nil {
		return Email{}, fmt.Errorf("get message %s: %w", id, err)
	}

	email := Email{
		ID:      id,
		Snippet: msg.Snippet,
	}

	for _, h := range msg.Payload.Headers {
		switch h.Name {
		case "Subject":
			email.Subject = h.Value
		case "From":
			email.From = h.Value
		case "Date":
			if t, err := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", h.Value); err == nil {
				email.Date = t
			} else if t, err := time.Parse("Mon, 2 Jan 2006 15:04:05 +0000 (UTC)", h.Value); err == nil {
				email.Date = t
			}
		}
	}

	email.Body = extractBody(msg.Payload)
	if email.Body == "" {
		email.Body = email.Snippet
	}

	return email, nil
}

func extractBody(part *gmailv1.MessagePart) string {
	if part == nil {
		return ""
	}
	if part.MimeType == "text/plain" && part.Body != nil && part.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err == nil {
			return truncate(string(data), 500)
		}
	}
	for _, p := range part.Parts {
		if body := extractBody(p); body != "" {
			return body
		}
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

