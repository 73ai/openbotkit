package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/73ai/openbotkit/store"
)

// TimelineFetcher abstracts the X client for testability.
type TimelineFetcher interface {
	HomeTimeline(ctx context.Context, count int, cursor string) (json.RawMessage, error)
	HomeLatestTimeline(ctx context.Context, count int, cursor string) (json.RawMessage, error)
}

func Sync(ctx context.Context, db *store.DB, fetcher TimelineFetcher, opts SyncOptions) (*SyncResult, error) {
	if err := Migrate(db); err != nil {
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	timelineType := opts.TimelineType
	if timelineType == "" {
		timelineType = "following"
	}

	result := &SyncResult{}

	var cursor string
	if !opts.Full {
		state, err := GetSyncState(db, timelineType)
		if err != nil {
			return nil, fmt.Errorf("get sync state: %w", err)
		}
		if state != nil {
			cursor = state.Cursor
		}
	}

	const pageSize = 20
	for {
		var raw json.RawMessage
		var err error

		switch timelineType {
		case "foryou":
			raw, err = fetcher.HomeTimeline(ctx, pageSize, cursor)
		default:
			raw, err = fetcher.HomeLatestTimeline(ctx, pageSize, cursor)
		}
		if err != nil {
			return result, fmt.Errorf("fetch timeline: %w", err)
		}

		tweets, nextCursor, err := ParseTimelineResponse(raw)
		if err != nil {
			return result, fmt.Errorf("parse timeline: %w", err)
		}

		if len(tweets) == 0 {
			break
		}

		for i := range tweets {
			tw := &tweets[i]
			exists, err := TweetExists(db, tw.TweetID)
			if err != nil {
				slog.Error("check exists", "tweet_id", tw.TweetID, "error", err)
				result.Errors++
				continue
			}
			if exists && !opts.Full {
				result.Skipped++
				continue
			}

			if err := SaveTweet(db, tw); err != nil {
				slog.Error("save tweet", "tweet_id", tw.TweetID, "error", err)
				result.Errors++
				continue
			}
			result.Fetched++
		}

		if nextCursor == "" || nextCursor == cursor {
			break
		}
		cursor = nextCursor

		// Stop after first page for non-full syncs with existing state
		if !opts.Full {
			break
		}
	}

	if cursor != "" {
		if err := SaveSyncState(db, timelineType, cursor); err != nil {
			slog.Error("save sync state", "error", err)
		}
	}

	return result, nil
}
