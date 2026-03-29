package twitter

import "time"

type Tweet struct {
	TweetID        string
	UserID         string
	UserName       string
	UserFullName   string
	Text           string
	CreatedAt      time.Time
	InReplyTo      string
	ConversationID string
	RetweetCount   int
	LikeCount      int
	ReplyCount     int
	IsRetweet      bool
	RetweetedFrom  string
	QuotedTweetID  string
}

type User struct {
	UserID     string
	ScreenName string
	FullName   string
	Bio        string
	Verified   bool
	Followers  int
	Following  int
}

type SyncOptions struct {
	Full         bool
	TimelineType string // "foryou" or "following"
}

type SyncResult struct {
	Fetched int
	Skipped int
	Errors  int
}

type SyncState struct {
	TimelineType string
	Cursor       string
	UpdatedAt    time.Time
}

type ListOptions struct {
	TimelineType string
	Limit        int
	Offset       int
}

type Config struct {
	EndpointsPath string
}
