package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/time/rate"
	gapi "google.golang.org/api/gmail/v1"
)

func composeRawMessage(input ComposeInput) (string, error) {
	if len(input.To) == 0 {
		return "", fmt.Errorf("at least one recipient is required")
	}

	var msg strings.Builder
	msg.WriteString("From: " + input.Account + "\r\n")
	msg.WriteString("To: " + strings.Join(input.To, ", ") + "\r\n")
	if len(input.Cc) > 0 {
		msg.WriteString("Cc: " + strings.Join(input.Cc, ", ") + "\r\n")
	}
	if len(input.Bcc) > 0 {
		msg.WriteString("Bcc: " + strings.Join(input.Bcc, ", ") + "\r\n")
	}
	msg.WriteString("Subject: " + input.Subject + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(input.Body)

	encoded := base64.URLEncoding.EncodeToString([]byte(msg.String()))
	return encoded, nil
}

func SendEmail(srv *gapi.Service, input ComposeInput, limiter *rate.Limiter) (*SendResult, error) {
	raw, err := composeRawMessage(input)
	if err != nil {
		return nil, fmt.Errorf("compose message: %w", err)
	}

	limiter.Wait(context.Background())
	sent, err := srv.Users.Messages.Send("me", &gapi.Message{Raw: raw}).Do()
	if err != nil {
		return nil, fmt.Errorf("send email: %w", err)
	}

	return &SendResult{
		MessageID: sent.Id,
		ThreadID:  sent.ThreadId,
	}, nil
}

func CreateDraft(srv *gapi.Service, input ComposeInput, limiter *rate.Limiter) (*DraftResult, error) {
	raw, err := composeRawMessage(input)
	if err != nil {
		return nil, fmt.Errorf("compose message: %w", err)
	}

	limiter.Wait(context.Background())
	draft, err := srv.Users.Drafts.Create("me", &gapi.Draft{
		Message: &gapi.Message{Raw: raw},
	}).Do()
	if err != nil {
		return nil, fmt.Errorf("create draft: %w", err)
	}

	return &DraftResult{
		DraftID:   draft.Id,
		MessageID: draft.Message.Id,
		ThreadID:  draft.Message.ThreadId,
	}, nil
}
