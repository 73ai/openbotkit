package gmail

import (
	"context"
	"net/http"
	"time"
)

type Email struct {
	MessageID   string
	Account     string
	From        string
	To          string
	Subject     string
	Date        time.Time
	Body        string
	HTMLBody    string
	Attachments []Attachment
}

type Attachment struct {
	Filename  string
	MimeType  string
	Data      []byte
	SavedPath string
}

// GmailProvider is the subset of provider.Provider that Gmail needs.
// Using an interface avoids circular imports with provider/google.
type GmailProvider interface {
	Client(ctx context.Context, account string, scopes []string) (*http.Client, error)
	Accounts(ctx context.Context) ([]string, error)
}

type Config struct {
	Provider GmailProvider
}

type SyncOptions struct {
	Full                bool
	After               string // YYYY/MM/DD (Gmail API format)
	Account             string
	DownloadAttachments bool
	AttachmentsDir      string
}

type SyncResult struct {
	Fetched int
	Skipped int
	Errors  int
}

type ListOptions struct {
	Account string
	From    string
	Subject string
	After   string // YYYY-MM-DD
	Before  string // YYYY-MM-DD
	Limit   int
	Offset  int
}

type FetchQuery struct {
	From  string
	After string
	Query string // raw Gmail query (takes precedence over From/After)
}
