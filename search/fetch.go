package search

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"golang.org/x/time/rate"
	"google.golang.org/api/gmail/v1"
)

// Email holds a fully parsed Gmail message.
type Email struct {
	MessageID   string
	Account     string
	From        string
	To          string
	Subject     string
	Date        time.Time
	Body        string // plain text body
	HTMLBody    string // HTML body
	Attachments []Attachment
}

// Attachment holds attachment metadata and data.
type Attachment struct {
	Filename  string
	MimeType  string
	Data      []byte
	SavedPath string // populated after saving to disk
}

// NewRateLimiter returns a rate limiter for Gmail API calls.
// 15 requests/sec stays well within the 250 quota-units/sec budget.
func NewRateLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Limit(15), 1)
}

// SearchIDs returns message IDs matching a FetchQuery.
func SearchIDs(srv *gmail.Service, query FetchQuery, limiter *rate.Limiter) ([]string, error) {
	var ids []string
	qStr := query.Build()
	pageToken := ""

	for {
		limiter.Wait(context.Background())
		req := srv.Users.Messages.List("me").Q(qStr)
		if pageToken != "" {
			req.PageToken(pageToken)
		}
		res, err := req.Do()
		if err != nil {
			return nil, fmt.Errorf("search emails (query=%q): %w", qStr, err)
		}
		for _, msg := range res.Messages {
			ids = append(ids, msg.Id)
		}
		if res.NextPageToken == "" {
			break
		}
		pageToken = res.NextPageToken
	}
	return ids, nil
}

// FetchEmail retrieves a full message and parses headers, body, and attachment metadata.
func FetchEmail(srv *gmail.Service, accountEmail string, msgID string, limiter *rate.Limiter) (*Email, error) {
	limiter.Wait(context.Background())
	msg, err := srv.Users.Messages.Get("me", msgID).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("fetch message %s: %w", msgID, err)
	}

	email := &Email{
		MessageID: msgID,
		Account:   accountEmail,
	}

	// Parse headers
	for _, h := range msg.Payload.Headers {
		switch strings.ToLower(h.Name) {
		case "from":
			email.From = h.Value
		case "to":
			email.To = h.Value
		case "subject":
			email.Subject = h.Value
		case "date":
			if t, err := parseDate(h.Value); err == nil {
				email.Date = t
			}
		}
	}

	// Parse body and attachments recursively
	parseParts(srv, msgID, msg.Payload, email, limiter)

	return email, nil
}

func parseParts(srv *gmail.Service, msgID string, part *gmail.MessagePart, email *Email, limiter *rate.Limiter) {
	// If this part has a body with data, extract it
	if part.Body != nil && part.Body.Data != "" && part.Filename == "" {
		data, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err == nil {
			switch {
			case strings.HasPrefix(part.MimeType, "text/plain"):
				email.Body += string(data)
			case strings.HasPrefix(part.MimeType, "text/html"):
				email.HTMLBody += string(data)
			}
		}
	}

	// If this part is an attachment
	if part.Filename != "" && part.Body != nil && part.Body.AttachmentId != "" {
		att := Attachment{
			Filename: part.Filename,
			MimeType: part.MimeType,
		}
		// Fetch attachment data
		limiter.Wait(context.Background())
		attData, err := srv.Users.Messages.Attachments.Get("me", msgID, part.Body.AttachmentId).Do()
		if err == nil {
			decoded, err := base64.URLEncoding.DecodeString(attData.Data)
			if err == nil {
				att.Data = decoded
			}
		}
		email.Attachments = append(email.Attachments, att)
	}

	// Recurse into sub-parts
	for _, sub := range part.Parts {
		parseParts(srv, msgID, sub, email, limiter)
	}
}

func parseDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 -0700 (MST)",
		"2 Jan 2006 15:04:05 -0700",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}
