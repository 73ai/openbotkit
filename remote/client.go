package remote

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL  string
	username string
	password string
	http     *http.Client
}

func NewClient(baseURL, username, password string) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

// Health checks server connectivity.
func (c *Client) Health() (map[string]string, error) {
	var resp map[string]string
	if err := c.get("/api/health", &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// DB executes a SQL query against a source database.
type DBResponse struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

func (c *Client) DB(source, sql string) (*DBResponse, error) {
	body := map[string]string{"sql": sql}
	var resp DBResponse
	if err := c.post(fmt.Sprintf("/api/db/%s", source), body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// MemoryItem represents a memory from the server.
type MemoryItem struct {
	ID        int64  `json:"id"`
	Content   string `json:"content"`
	Category  string `json:"category"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
}

// MemoryList retrieves all memories, optionally filtered by category.
func (c *Client) MemoryList(category string) ([]MemoryItem, error) {
	path := "/api/memory"
	if category != "" {
		path += "?category=" + url.QueryEscape(category)
	}
	var items []MemoryItem
	if err := c.get(path, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// MemoryAdd adds a new memory. Returns the new ID.
func (c *Client) MemoryAdd(content, category, source string) (int64, error) {
	body := map[string]string{
		"content":  content,
		"category": category,
		"source":   source,
	}
	var resp struct {
		ID int64 `json:"id"`
	}
	if err := c.post("/api/memory", body, &resp); err != nil {
		return 0, err
	}
	return resp.ID, nil
}

// MemoryDelete deletes a memory by ID.
func (c *Client) MemoryDelete(id int64) error {
	return c.delete(fmt.Sprintf("/api/memory/%d", id))
}

// AppleNotesPush sends notes to the remote server.
func (c *Client) AppleNotesPush(notes any) error {
	return c.post("/api/applenotes/push", notes, nil)
}

// IMessagePush sends iMessage messages to the remote server.
func (c *Client) IMessagePush(messages any) error {
	return c.post("/api/imessage/push", messages, nil)
}

// GmailSendResult holds the response from a Gmail send.
type GmailSendResult struct {
	MessageID string `json:"message_id"`
	ThreadID  string `json:"thread_id"`
}

// GmailSend sends an email via the remote server.
func (c *Client) GmailSend(to, cc, bcc []string, subject, body, account string) (*GmailSendResult, error) {
	req := map[string]any{
		"to": to, "cc": cc, "bcc": bcc,
		"subject": subject, "body": body, "account": account,
	}
	var resp GmailSendResult
	if err := c.post("/api/gmail/send", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GmailDraftResult holds the response from creating a Gmail draft.
type GmailDraftResult struct {
	DraftID   string `json:"draft_id"`
	MessageID string `json:"message_id"`
	ThreadID  string `json:"thread_id"`
}

// GmailDraft creates a draft email via the remote server.
func (c *Client) GmailDraft(to, cc, bcc []string, subject, body, account string) (*GmailDraftResult, error) {
	req := map[string]any{
		"to": to, "cc": cc, "bcc": bcc,
		"subject": subject, "body": body, "account": account,
	}
	var resp GmailDraftResult
	if err := c.post("/api/gmail/draft", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GmailSyncResult holds the response from a Gmail sync.
type GmailSyncResult struct {
	Fetched int `json:"fetched"`
	Skipped int `json:"skipped"`
	Errors  int `json:"errors"`
}

// GmailSync triggers a Gmail sync on the remote server.
func (c *Client) GmailSync(full bool, after, account string, daysWindow int) (*GmailSyncResult, error) {
	req := map[string]any{
		"full": full, "after": after,
		"account": account, "days_window": daysWindow,
	}
	var resp GmailSyncResult
	if err := c.post("/api/gmail/sync", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// WhatsAppSendResult holds the response from sending a WhatsApp message.
type WhatsAppSendResult struct {
	MessageID string `json:"message_id"`
	Timestamp string `json:"timestamp"`
}

// MemoryExtractResult holds the response from memory extraction.
type MemoryExtractResult struct {
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Deleted int `json:"deleted"`
	Skipped int `json:"skipped"`
}

// MemoryExtract runs memory extraction on the remote server.
func (c *Client) MemoryExtract(last int) (*MemoryExtractResult, error) {
	req := map[string]int{"last": last}
	var resp MemoryExtractResult
	if err := c.post("/api/memory/extract", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// WaitWhatsAppAuth polls the WhatsApp auth endpoint until authentication
// completes. Returns nil when authenticated, or an error on timeout (5 min).
func (c *Client) WaitWhatsAppAuth() error {
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		var resp struct {
			Authenticated bool `json:"authenticated"`
		}
		if err := c.get("/auth/whatsapp/api/qr", &resp); err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		if resp.Authenticated {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timed out waiting for WhatsApp authentication")
}

// WhatsAppSend sends a WhatsApp message via the remote server.
func (c *Client) WhatsAppSend(to, text string) (*WhatsAppSendResult, error) {
	req := map[string]string{"to": to, "text": text}
	var resp WhatsAppSendResult
	if err := c.post("/api/whatsapp/send", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) get(path string, result any) error {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, result)
}

func (c *Client) post(path string, body any, result any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequest("POST", c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, result)
}

func (c *Client) delete(path string) error {
	req, err := http.NewRequest("DELETE", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) do(req *http.Request, result any) error {
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return fmt.Errorf("server error (%d): %s", resp.StatusCode, errResp.Error)
		}
		return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
