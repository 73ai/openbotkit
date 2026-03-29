package client

import (
	"context"
	"encoding/json"
)

func (c *Client) Notifications(ctx context.Context, count int, cursor string) (json.RawMessage, error) {
	vars := map[string]any{
		"count":                  count,
		"includePromotedContent": false,
	}
	if cursor != "" {
		vars["cursor"] = cursor
	}
	return c.graphqlGET(ctx, "Notifications", vars, DefaultFeatures())
}
