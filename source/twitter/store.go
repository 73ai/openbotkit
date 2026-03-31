package twitter

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/73ai/openbotkit/store"
)

func SaveTweet(db *store.DB, tweet *Tweet) error {
	_, err := db.Exec(
		db.Rebind(`INSERT INTO tweets (tweet_id, user_id, user_name, user_full_name, text, created_at,
		 in_reply_to, conversation_id, retweet_count, like_count, reply_count,
		 is_retweet, retweeted_from, quoted_tweet_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (tweet_id) DO UPDATE SET
		 retweet_count = excluded.retweet_count,
		 like_count = excluded.like_count,
		 reply_count = excluded.reply_count`),
		tweet.TweetID, tweet.UserID, tweet.UserName, tweet.UserFullName,
		tweet.Text, tweet.CreatedAt.UTC(),
		tweet.InReplyTo, tweet.ConversationID,
		tweet.RetweetCount, tweet.LikeCount, tweet.ReplyCount,
		tweet.IsRetweet, tweet.RetweetedFrom, tweet.QuotedTweetID,
	)
	if err != nil {
		return fmt.Errorf("save tweet: %w", err)
	}
	return nil
}

func TweetExists(db *store.DB, tweetID string) (bool, error) {
	var count int
	err := db.QueryRow(
		db.Rebind("SELECT COUNT(*) FROM tweets WHERE tweet_id = ?"),
		tweetID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check tweet exists: %w", err)
	}
	return count > 0, nil
}

func GetTweet(db *store.DB, tweetID string) (*Tweet, error) {
	var t Tweet
	err := db.QueryRow(
		db.Rebind(`SELECT tweet_id, user_id, user_name, user_full_name, text, created_at,
		 in_reply_to, conversation_id, retweet_count, like_count, reply_count,
		 is_retweet, retweeted_from, quoted_tweet_id
		 FROM tweets WHERE tweet_id = ?`),
		tweetID,
	).Scan(&t.TweetID, &t.UserID, &t.UserName, &t.UserFullName,
		&t.Text, &t.CreatedAt,
		&t.InReplyTo, &t.ConversationID,
		&t.RetweetCount, &t.LikeCount, &t.ReplyCount,
		&t.IsRetweet, &t.RetweetedFrom, &t.QuotedTweetID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tweet: %w", err)
	}
	return &t, nil
}

func ListTweets(db *store.DB, opts ListOptions) ([]Tweet, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	query := `SELECT tweet_id, user_id, user_name, user_full_name, text, created_at,
		 in_reply_to, conversation_id, retweet_count, like_count, reply_count,
		 is_retweet, retweeted_from, quoted_tweet_id
		 FROM tweets ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := db.Query(db.Rebind(query), limit, opts.Offset)
	if err != nil {
		return nil, fmt.Errorf("list tweets: %w", err)
	}
	defer rows.Close()

	tweets := []Tweet{}
	for rows.Next() {
		var t Tweet
		err := rows.Scan(&t.TweetID, &t.UserID, &t.UserName, &t.UserFullName,
			&t.Text, &t.CreatedAt,
			&t.InReplyTo, &t.ConversationID,
			&t.RetweetCount, &t.LikeCount, &t.ReplyCount,
			&t.IsRetweet, &t.RetweetedFrom, &t.QuotedTweetID)
		if err != nil {
			return nil, fmt.Errorf("scan tweet: %w", err)
		}
		tweets = append(tweets, t)
	}
	return tweets, rows.Err()
}

func SearchTweets(db *store.DB, query string, limit int) ([]Tweet, error) {
	if limit <= 0 {
		limit = 50
	}
	pattern := "%" + strings.ToLower(query) + "%"

	rows, err := db.Query(
		db.Rebind(`SELECT tweet_id, user_id, user_name, user_full_name, text, created_at,
		 in_reply_to, conversation_id, retweet_count, like_count, reply_count,
		 is_retweet, retweeted_from, quoted_tweet_id
		 FROM tweets WHERE LOWER(text) LIKE ? OR LOWER(user_name) LIKE ?
		 ORDER BY created_at DESC LIMIT ?`),
		pattern, pattern, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search tweets: %w", err)
	}
	defer rows.Close()

	tweets := []Tweet{}
	for rows.Next() {
		var t Tweet
		if err := rows.Scan(&t.TweetID, &t.UserID, &t.UserName, &t.UserFullName,
			&t.Text, &t.CreatedAt,
			&t.InReplyTo, &t.ConversationID,
			&t.RetweetCount, &t.LikeCount, &t.ReplyCount,
			&t.IsRetweet, &t.RetweetedFrom, &t.QuotedTweetID); err != nil {
			return nil, fmt.Errorf("scan tweet: %w", err)
		}
		tweets = append(tweets, t)
	}
	return tweets, rows.Err()
}

func CountTweets(db *store.DB) (int64, error) {
	var count int64
	err := db.QueryRow("SELECT COUNT(*) FROM tweets").Scan(&count)
	return count, err
}

func LastSyncTime(db *store.DB) (*time.Time, error) {
	var raw sql.NullString
	err := db.QueryRow("SELECT MAX(fetched_at) FROM tweets").Scan(&raw)
	if err != nil {
		return nil, err
	}
	if !raw.Valid || raw.String == "" {
		return nil, nil
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	} {
		if t, err := time.Parse(layout, raw.String); err == nil {
			return &t, nil
		}
	}
	return nil, nil
}

func GetSyncState(db *store.DB, timelineType string) (*SyncState, error) {
	var s SyncState
	err := db.QueryRow(
		db.Rebind("SELECT timeline_type, cursor, updated_at FROM x_sync_state WHERE timeline_type = ?"),
		timelineType,
	).Scan(&s.TimelineType, &s.Cursor, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get sync state: %w", err)
	}
	return &s, nil
}

func SaveSyncState(db *store.DB, timelineType, cursor string) error {
	_, err := db.Exec(
		db.Rebind(`INSERT INTO x_sync_state (timeline_type, cursor, updated_at)
		 VALUES (?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT (timeline_type) DO UPDATE SET cursor = ?, updated_at = CURRENT_TIMESTAMP`),
		timelineType, cursor, cursor,
	)
	if err != nil {
		return fmt.Errorf("save sync state: %w", err)
	}
	return nil
}
