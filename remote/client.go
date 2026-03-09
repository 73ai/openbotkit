package remote

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		path += "?category=" + category
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

func (c *Client) get(path string, result interface{}) error {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, result)
}

func (c *Client) post(path string, body interface{}, result interface{}) error {
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

func (c *Client) do(req *http.Request, result interface{}) error {
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
