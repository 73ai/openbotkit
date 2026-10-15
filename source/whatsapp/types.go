package whatsapp

import "time"

const DefaultAccount = "default"

type Message struct {
	Account    string
	MessageID  string
	ChatJID    string
	SenderJID  string
	SenderName string
	Text       string
	Timestamp  time.Time
	MediaType  string
	MediaPath  string
	IsGroup    bool
	IsFromMe   bool
	ReplyToID  string
}

type Chat struct {
	Account       string
	JID           string
	Name          string
	IsGroup       bool
	LastMessageAt *time.Time
}

type Contact struct {
	Account      string
	JID          string
	Phone        string
	FirstName    string
	FullName     string
	PushName     string
	BusinessName string
}

type Config struct {
	Account       string
	SessionDBPath string
	DataDriver    string
	DataDSN       string
}

type SyncOptions struct {
	Account string
	Follow  bool
}

type SyncResult struct {
	Received        int
	HistoryMessages int
	Errors          int
}

type SendInput struct {
	Account string
	ChatJID string
	Text    string
}

type SendResult struct {
	MessageID string
	Timestamp time.Time
}

type ListOptions struct {
	Account string
	ChatJID string
	After   string // YYYY-MM-DD
	Before  string // YYYY-MM-DD
	Limit   int
	Offset  int
}
