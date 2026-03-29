package client

import (
	"context"
	"encoding/json"
)

func (c *Client) Search(ctx context.Context, query string, count int, cursor string) (json.RawMessage, error) {
	vars := map[string]any{
		"rawQuery":    query,
		"count":       count,
		"querySource": "typed_query",
		"product":     "Latest",
	}
	if cursor != "" {
		vars["cursor"] = cursor
	}
	return c.graphqlGET(ctx, "SearchTimeline", vars, DefaultFeatures())
}
