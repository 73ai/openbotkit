package twitter

import "github.com/73ai/openbotkit/store"

const schemaSQLite = `
CREATE TABLE IF NOT EXISTS tweets (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	tweet_id TEXT NOT NULL UNIQUE,
	user_id TEXT NOT NULL,
	user_name TEXT,
	user_full_name TEXT,
	text TEXT,
	created_at DATETIME,
	in_reply_to TEXT,
	conversation_id TEXT,
	retweet_count INTEGER DEFAULT 0,
	like_count INTEGER DEFAULT 0,
	reply_count INTEGER DEFAULT 0,
	is_retweet BOOLEAN DEFAULT FALSE,
	retweeted_from TEXT,
	quoted_tweet_id TEXT,
	fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tweets_created ON tweets(created_at);
CREATE INDEX IF NOT EXISTS idx_tweets_user ON tweets(user_id);
CREATE INDEX IF NOT EXISTS idx_tweets_conversation ON tweets(conversation_id);

CREATE TABLE IF NOT EXISTS x_sync_state (
	timeline_type TEXT PRIMARY KEY,
	cursor TEXT NOT NULL,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

const schemaPostgres = `
CREATE TABLE IF NOT EXISTS tweets (
	id BIGSERIAL PRIMARY KEY,
	tweet_id TEXT NOT NULL UNIQUE,
	user_id TEXT NOT NULL,
	user_name TEXT,
	user_full_name TEXT,
	text TEXT,
	created_at TIMESTAMPTZ,
	in_reply_to TEXT,
	conversation_id TEXT,
	retweet_count INTEGER DEFAULT 0,
	like_count INTEGER DEFAULT 0,
	reply_count INTEGER DEFAULT 0,
	is_retweet BOOLEAN DEFAULT FALSE,
	retweeted_from TEXT,
	quoted_tweet_id TEXT,
	fetched_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tweets_created ON tweets(created_at);
CREATE INDEX IF NOT EXISTS idx_tweets_user ON tweets(user_id);
CREATE INDEX IF NOT EXISTS idx_tweets_conversation ON tweets(conversation_id);

CREATE TABLE IF NOT EXISTS x_sync_state (
	timeline_type TEXT PRIMARY KEY,
	cursor TEXT NOT NULL,
	updated_at TIMESTAMPTZ DEFAULT NOW()
);
`

func Migrate(db *store.DB) error {
	schema := schemaSQLite
	if db.IsPostgres() {
		schema = schemaPostgres
	}
	_, err := db.Exec(schema)
	return err
}
